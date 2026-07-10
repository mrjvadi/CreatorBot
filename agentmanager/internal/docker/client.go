// Package docker یک wrapper امن روی Docker SDK است.
// به‌جای اجرای دستور CLI، مستقیماً با Docker daemon از طریق socket کار می‌کند.
// این روش در برابر command injection کاملاً ایمن است چون هیچ string‌ای
// به shell پاس داده نمی‌شود — همه‌ی پارامترها strongly-typed هستند.
package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	dockerclient "github.com/docker/docker/client"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// SecurityPolicy پیش‌فرض‌های امنیتی و محدودیت منابع این سرور را نگه می‌دارد.
// این مقادیر از .env خوانده می‌شوند و می‌توانند per-deploy توسط
// protocol.DeploySettings بازنویسی شوند.
//
// نکته‌ی مهم (۲۰۲۶-۰۷-۰۴): whitelist محلیِ image (قبلاً این‌جا، به‌صورت
// AllowedImages []string با prefix matching) حذف شد و جایش را یک سرویس
// مرکزی جدید گرفت: image-registry. حالا هر agentmanager قبل از هر deploy
// از آن سرویس می‌پرسد «آیا این image:tag مجاز است؟» — به‌جای یک لیست محلی
// که هر سرور جدا نگه می‌داشت. رجوع به ImageChecker پایین و
// image-registry/README.md.
type SecurityPolicy struct {
	// DefaultMemoryMB سقف حافظه‌ی پیش‌فرض هر container به مگابایت (۰ = نامحدود).
	DefaultMemoryMB int64
	// DefaultCPUs تعداد هسته‌ی پیش‌فرض (۰ = نامحدود).
	DefaultCPUs float64
	// DefaultPidsLimit حداکثر پردازه‌ی پیش‌فرض؛ ضد fork-bomb (۰ = نامحدود).
	DefaultPidsLimit int64
	// ReadonlyRootfs آیا فایل‌سیستم ریشه به‌صورت پیش‌فرض فقط‌خواندنی باشد.
	ReadonlyRootfs bool
	// DefaultTmpfsMB اندازه‌ی tmpfs برای /tmp وقتی rootfs فقط‌خواندنی است.
	DefaultTmpfsMB int64
}

// ImageChecker تنها چیزی است که Client از سرویس image-registry نیاز دارد —
// یک interface کوچک، تا تست‌نویسی راحت باشد و docker package مستقیم به
// جزئیات HTTP وابسته نشود. پیاده‌سازی واقعی: agentmanager/internal/registryclient.Client.
type ImageChecker interface {
	IsAllowed(ctx context.Context, name, tag string) (bool, error)
}

// DeployDefaults دانش لوکالِ این سرور برای هر container ای که deploy می‌شود.
//
// چرا این‌جا و نه در DeployCommand؟ آدرس‌های زیرساخت (Mongo/NATS/Redis/...)
// و secret های اتصال، مالِ همین سرورند — botmanager روی سرور دیگری است و
// نه باید آن‌ها را بداند و نه باید از NATS عبورشان بدهد. botmanager فقط
// env مخصوص app را می‌فرستد (BOT_TOKEN, INSTANCE_ID, ...)؛ بقیه از این‌جا
// تزریق می‌شود.
type DeployDefaults struct {
	// BaseEnv در env هر container گذاشته می‌شود؛ اشتراک همه‌ی service type ها
	// (NATS/Redis/Mongo آدرس، ENCRYPTION_KEY و مانند آن).
	BaseEnv map[string]string
	// TypeEnvDir دایرکتوری با فایل‌های per-service-type (مثلاً uploader.env,
	// vpn-bot.env, archive-bot.env). اگر فایل <TypeEnvDir>/<cmd.ImageName>.env
	// وجود داشت، روی BaseEnv اعمال می‌شود — برای DSN های متفاوت هر نوع ربات.
	TypeEnvDir string
	// DefaultNetwork وقتی DeployCommand.NetworkName خالی باشد استفاده می‌شود —
	// بدون آن container به شبکه‌ی bridge پیش‌فرض می‌رود و اسم‌هایی مثل
	// "nats"/"mongodb" را resolve نمی‌کند.
	DefaultNetwork string
}

// Client یک wrapper امن روی Docker SDK است.
type Client struct {
	cli      *dockerclient.Client
	log      ports.Logger
	policy   SecurityPolicy
	defaults DeployDefaults
	registry ImageChecker
}

// NewClient یک Client جدید با اتصال به Docker daemon محلی می‌سازد.
// آدرس socket از متغیر محیطی DOCKER_HOST خوانده می‌شود (پیش‌فرض: unix:///var/run/docker.sock).
// registry سرویس image-registry است — تنها منبع تصمیم «این image مجاز است؟».
func NewClient(log ports.Logger, policy SecurityPolicy, defaults DeployDefaults, registry ImageChecker) (*Client, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("docker sdk: %w", err)
	}
	return &Client{cli: cli, log: log, policy: policy, defaults: defaults, registry: registry}, nil
}

// isImageAllowed از image-registry می‌پرسد آیا ref ("name:tag") مجاز است.
// Fail-closed در هر خطا (سرویس در دسترس نیست، IP این سرور مجاز نیست، ...)
// — دقیقاً همان فلسفه‌ی قدیمیِ «whitelist خالی/نامعتبر = رد همه‌چیز»، فقط
// حالا منبع تصمیم مرکزی است نه یک فایل .env محلی روی هر سرور.
func (c *Client) isImageAllowed(ctx context.Context, ref string) bool {
	name, tag, ok := splitImageRef(ref)
	if !ok {
		c.log.Warn("image ref has no tag — rejecting", ports.F("ref", ref))
		return false
	}
	allowed, err := c.registry.IsAllowed(ctx, name, tag)
	if err != nil {
		c.log.Error("image-registry check failed — rejecting (fail-closed)",
			ports.F("ref", ref), ports.F("err", err))
		return false
	}
	return allowed
}

// splitImageRef "name:tag" را به دو بخش تقسیم می‌کند — از آخرین ":" تا
// آدرس‌های registry با پورت (مثل "host:5000/name:tag") درست پارس شوند.
func splitImageRef(ref string) (name, tag string, ok bool) {
	i := strings.LastIndex(ref, ":")
	if i < 0 || i == len(ref)-1 {
		return "", "", false
	}
	return ref[:i], ref[i+1:], true
}

// Deploy یک container را از روی image می‌سازد و اجرا می‌کند.
// این متد idempotent است — اگه container قبلی وجود داشت حذف می‌شود.
func (c *Client) Deploy(ctx context.Context, cmd protocol.DeployCommand) (string, error) {
	ref := cmd.ImageName + ":" + cmd.ImageTag

	// ── ۱. کنترل whitelist (از سرویس مرکزی image-registry) ──────
	// فقط image هایی که در استخر ثبت‌شده و فعال هستند اجازه‌ی اجرا دارند.
	if !c.isImageAllowed(ctx, ref) {
		return "", fmt.Errorf("image %q در image-registry مجاز نیست", ref)
	}

	// ── ۲. فقط image محلی (بدون pull از اینترنت) ─────────────────
	// هیچ‌گاه به registry وصل نمی‌شویم؛ image باید از قبل روی سرور باشد.
	// این جلوی اجرای image آلوده یا دستکاری‌شده از بیرون را می‌گیرد.
	if _, _, err := c.cli.ImageInspectWithRaw(ctx, ref); err != nil {
		if dockerclient.IsErrNotFound(err) {
			return "", fmt.Errorf("image %q روی این سرور موجود نیست (pull غیرفعال است؛ ابتدا image را روی سرور بسازید/لود کنید)", ref)
		}
		return "", fmt.Errorf("inspect image %q: %w", ref, err)
	}

	// ── ۳. حذف container قدیمی (در صورت وجود) ────────────────
	_, _ = c.Remove(ctx, cmd.ContainerName)

	// ── ۴. ساخت env slice (strongly typed — بدون injection) ───
	// ترتیب merge (هر لایه بر قبلی برنده است):
	// ۱. BaseEnv (زیرساخت مشترک: NATS/Redis/Mongo)
	// ۲. TypeEnv (per-service-type: DSN اختصاصی هر نوع ربات)
	// ۳. cmd.EnvVars (مقادیر app-specific از botmanager: BOT_TOKEN, INSTANCE_ID, ...)
	typeEnv, typeErr := parseEnvFileIfExists(c.defaults.TypeEnvDir + "/" + cmd.ImageName + ".env")
	if typeErr != nil {
		c.log.Warn("type env file unreadable — using base only",
			ports.F("image", cmd.ImageName), ports.F("err", typeErr))
		typeEnv = map[string]string{}
	}
	env := mergeEnv(mergeEnvMaps(c.defaults.BaseEnv, typeEnv), cmd.EnvVars)

	// ── ۵. NetworkConfig ────────────────────────────────────────
	netName := cmd.NetworkName
	if netName == "" {
		netName = c.defaults.DefaultNetwork
	}
	var netCfg *network.NetworkingConfig
	if netName != "" {
		netCfg = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				netName: {},
			},
		}
	}

	// ── ۶. ادغام پیش‌فرض‌های سرور با تنظیمات per-deploy ──────────
	hostCfg := c.buildHostConfig(cmd.Settings)

	// ── ۷. Create container (سخت‌گیری امنیتی اعمال‌شده) ──────────
	// Label managedLabel روی هر container ای که خود agentmanager می‌سازد
	// گذاشته می‌شود؛ Stop/Remove/Restart قبل از اجرا این label را چک
	// می‌کنند تا یک دستور NATS دلخواه نتواند container های زیرساختی پلتفرم
	// (postgres, nats, agentmanager, botpay, ...) را که با این مسیر ساخته
	// نشده‌اند حذف/متوقف کند.
	resp, err := c.cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: ref,
			Env:   env,
			Labels: map[string]string{
				managedLabel:    "true",
				managedNameAttr: cmd.ContainerName,
			},
		},
		hostCfg,
		netCfg,
		nil, // platform — nil = default
		cmd.ContainerName,
	)
	if err != nil {
		return "", fmt.Errorf("create %q: %w", cmd.ContainerName, err)
	}

	// ── ۶. Start container ──────────────────────────────────────
	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("start %q: %w", cmd.ContainerName, err)
	}

	shortID := resp.ID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	c.log.Info("deployed",
		ports.F("container", cmd.ContainerName),
		ports.F("id", shortID),
		ports.F("image", ref))
	return resp.ID, nil
}

// buildHostConfig پیکربندی امنیتی container را می‌سازد:
// پیش‌فرض‌های سخت‌گیرانه‌ی سرور را اعمال می‌کند و در صورت وجود،
// با تنظیمات per-deploy (s) بازنویسی می‌کند.
func (c *Client) buildHostConfig(s *protocol.DeploySettings) *container.HostConfig {
	// شروع از پیش‌فرض‌های سرور
	memMB := c.policy.DefaultMemoryMB
	cpus := c.policy.DefaultCPUs
	pids := c.policy.DefaultPidsLimit
	readonly := c.policy.ReadonlyRootfs
	tmpfsMB := c.policy.DefaultTmpfsMB
	var capAdd strslice.StrSlice

	// override per-deploy (اختیاری)
	if s != nil {
		if s.MemoryMB > 0 {
			memMB = s.MemoryMB
		}
		if s.CPUs > 0 {
			cpus = s.CPUs
		}
		if s.PidsLimit > 0 {
			pids = s.PidsLimit
		}
		if s.ReadonlyRootfs != nil {
			readonly = *s.ReadonlyRootfs
		}
		if s.TmpfsSizeMB > 0 {
			tmpfsMB = s.TmpfsSizeMB
		}
		if len(s.CapAdd) > 0 {
			capAdd = strslice.StrSlice(s.CapAdd)
		}
	}

	hc := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyUnlessStopped,
		},
		// ── سخت‌گیری امنیتی ──
		// جلوگیری از بالا بردن سطح دسترسی (setuid/sudo داخل container بی‌اثر می‌شود)
		SecurityOpt: []string{"no-new-privileges:true"},
		// حذف همه‌ی capability های kernel؛ فقط موارد لازم دوباره اضافه می‌شوند
		CapDrop:    strslice.StrSlice{"ALL"},
		CapAdd:     capAdd,
		Privileged: false,
		// فایل‌سیستم ریشه فقط‌خواندنی → بدافزار نمی‌تواند روی image بنویسد
		ReadonlyRootfs: readonly,
		Resources:      container.Resources{},
	}

	// محدودیت منابع (۰ = اعمال نشود)
	if memMB > 0 {
		hc.Resources.Memory = memMB * 1024 * 1024
	}
	if cpus > 0 {
		hc.Resources.NanoCPUs = int64(cpus * 1e9)
	}
	if pids > 0 {
		p := pids
		hc.Resources.PidsLimit = &p
	}

	// وقتی rootfs فقط‌خواندنی است، یک tmpfs نوشتنی برای /tmp می‌دهیم
	// تا app هایی که فایل موقت می‌سازند کار کنند (noexec/nosuid برای امنیت).
	if readonly && tmpfsMB > 0 {
		hc.Tmpfs = map[string]string{
			"/tmp": fmt.Sprintf("rw,noexec,nosuid,size=%dm", tmpfsMB),
		}
	}

	return hc
}

// managedLabel/managedNameAttr علامت‌گذاری container هایی که خود
// agentmanager ساخته است — تنها این‌ها اجازه‌ی stop/remove/restart از طریق
// یک دستور NATS را دارند.
const (
	managedLabel    = "creatorbot.managed"
	managedNameAttr = "creatorbot.container_name"
)

// verifyManaged بررسی می‌کند این container توسط خود agentmanager ساخته شده
// (label creatorbot.managed=true را دارد). قبل از هر عملیات مخرب‌پذیر
// (stop/remove/restart) صدا زده می‌شود.
//
// چرا این لازم بود: قبلاً هر ContainerID/Name که در یک protocol.DeployCommand
// (که از NATS می‌آید) قید می‌شد بدون هیچ اعتبارسنجی به Docker پاس داده
// می‌شد. چون NATS بین همه‌ی سرویس‌ها یک username/password مشترک دارد (بدون
// per-subject ACL)، هر کلاینتی که به NATS دسترسی داشت می‌توانست
// {"type":"remove","container_id":"postgres"} بفرستد و کل پلتفرم را force-remove
// کند. این تابع آن مسیر را می‌بندد: فقط container هایی که خودمان با
// label مشخص ساخته‌ایم قابل‌حذف/توقف هستند.
func (c *Client) verifyManaged(ctx context.Context, containerID string) (bool, error) {
	info, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		if dockerclient.IsErrNotFound(err) {
			return false, nil // not found — جدا از "not managed" مدیریت می‌شود
		}
		return false, err
	}
	return info.Config != nil && info.Config.Labels[managedLabel] == "true", nil
}

// Stop یک container را متوقف می‌کند — فقط اگر توسط خود agentmanager ساخته شده باشد.
func (c *Client) Stop(ctx context.Context, containerID string) (string, error) {
	managed, err := c.verifyManaged(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("inspect %q: %w", containerID, err)
	}
	if !managed {
		c.log.Warn("refused to stop unmanaged/unknown container", ports.F("container_id", containerID))
		return "", fmt.Errorf("container %q is not managed by this agentmanager — refusing to stop", containerID)
	}
	timeout := 10
	if err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return "", fmt.Errorf("stop %q: %w", containerID, err)
	}
	return "stopped", nil
}

// Remove یک container را حذف می‌کند (force — حتی اگه در حال اجرا باشد) —
// فقط اگر توسط خود agentmanager ساخته شده باشد.
func (c *Client) Remove(ctx context.Context, containerID string) (string, error) {
	managed, err := c.verifyManaged(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("inspect %q: %w", containerID, err)
	}
	if !managed {
		// اگه container اصلاً وجود نداشت، idempotent (رفتار قبلی حفظ می‌شود).
		c.log.Warn("refused to remove unmanaged/unknown container", ports.F("container_id", containerID))
		return "not found or not managed", nil
	}
	if err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		if dockerclient.IsErrNotFound(err) {
			return "not found", nil
		}
		return "", fmt.Errorf("remove %q: %w", containerID, err)
	}
	return "removed", nil
}

// Restart یک container را ری‌استارت می‌کند — فقط اگر توسط خود agentmanager ساخته شده باشد.
func (c *Client) Restart(ctx context.Context, containerID string) (string, error) {
	managed, err := c.verifyManaged(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("inspect %q: %w", containerID, err)
	}
	if !managed {
		c.log.Warn("refused to restart unmanaged/unknown container", ports.F("container_id", containerID))
		return "", fmt.Errorf("container %q is not managed by this agentmanager — refusing to restart", containerID)
	}
	timeout := 10
	if err := c.cli.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return "", fmt.Errorf("restart %q: %w", containerID, err)
	}
	return "restarted", nil
}

// ListContainers لیست همه container ها را برمی‌گرداند (برای heartbeat).
func (c *Client) ListContainers(ctx context.Context) ([]protocol.ContainerStatus, error) {
	list, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	result := make([]protocol.ContainerStatus, 0, len(list))
	for _, ctr := range list {
		name := ""
		if len(ctr.Names) > 0 {
			// Docker اسم container ها را با "/" پیشوند می‌دهد
			name = strings.TrimPrefix(ctr.Names[0], "/")
		}
		result = append(result, protocol.ContainerStatus{
			Name:   name,
			Image:  ctr.Image,
			State:  ctr.State,
			Status: ctr.Status,
		})
	}
	return result, nil
}

// Close اتصال Docker SDK را می‌بندد.
func (c *Client) Close() error {
	return c.cli.Close()
}
