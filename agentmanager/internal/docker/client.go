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
// protocol.DeploySettings بازنویسی شوند (به‌جز whitelist که همیشه اجباری است).
type SecurityPolicy struct {
	// AllowedImages لیست prefix های مجاز برای image. خالی = هیچ image ای مجاز نیست.
	// مثال: ["registry.local/creatorbot/", "creatorbot/"].
	AllowedImages []string
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

// Client یک wrapper امن روی Docker SDK است.
type Client struct {
	cli    *dockerclient.Client
	log    ports.Logger
	policy SecurityPolicy
}

// NewClient یک Client جدید با اتصال به Docker daemon محلی می‌سازد.
// آدرس socket از متغیر محیطی DOCKER_HOST خوانده می‌شود (پیش‌فرض: unix:///var/run/docker.sock).
// policy پیش‌فرض‌های امنیتی/منابع و whitelist مجاز image را تعیین می‌کند.
func NewClient(log ports.Logger, policy SecurityPolicy) (*Client, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("docker sdk: %w", err)
	}
	if len(policy.AllowedImages) == 0 {
		log.Warn("image whitelist خالی است — هیچ deploy ای مجاز نخواهد بود؛ ALLOWED_IMAGES را تنظیم کنید")
	}
	return &Client{cli: cli, log: log, policy: policy}, nil
}

// isImageAllowed بررسی می‌کند که آیا image در whitelist است.
// اگر whitelist خالی باشد، هیچ image ای مجاز نیست (fail-safe).
func (c *Client) isImageAllowed(ref string) bool {
	for _, p := range c.policy.AllowedImages {
		if p != "" && strings.HasPrefix(ref, p) {
			return true
		}
	}
	return false
}

// Deploy یک container را از روی image می‌سازد و اجرا می‌کند.
// این متد idempotent است — اگه container قبلی وجود داشت حذف می‌شود.
func (c *Client) Deploy(ctx context.Context, cmd protocol.DeployCommand) (string, error) {
	ref := cmd.ImageName + ":" + cmd.ImageTag

	// ── ۱. کنترل whitelist ──────────────────────────────────────
	// فقط image هایی که در لیست مجاز سرور هستند اجازه‌ی اجرا دارند.
	if !c.isImageAllowed(ref) {
		return "", fmt.Errorf("image %q در whitelist مجاز نیست", ref)
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
	env := make([]string, 0, len(cmd.EnvVars))
	for k, v := range cmd.EnvVars {
		// فقط k=v — هیچ shell expansion نیست
		env = append(env, k+"="+v)
	}

	// ── ۵. NetworkConfig ────────────────────────────────────────
	var netCfg *network.NetworkingConfig
	if cmd.NetworkName != "" {
		netCfg = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				cmd.NetworkName: {},
			},
		}
	}

	// ── ۶. ادغام پیش‌فرض‌های سرور با تنظیمات per-deploy ──────────
	hostCfg := c.buildHostConfig(cmd.Settings)

	// ── ۷. Create container (سخت‌گیری امنیتی اعمال‌شده) ──────────
	resp, err := c.cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: ref,
			Env:   env,
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

// Stop یک container را متوقف می‌کند.
func (c *Client) Stop(ctx context.Context, containerID string) (string, error) {
	timeout := 10
	if err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return "", fmt.Errorf("stop %q: %w", containerID, err)
	}
	return "stopped", nil
}

// Remove یک container را حذف می‌کند (force — حتی اگه در حال اجرا باشد).
func (c *Client) Remove(ctx context.Context, containerID string) (string, error) {
	if err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		// اگه container وجود نداشت خطا نمی‌دهیم — idempotent
		if dockerclient.IsErrNotFound(err) {
			return "not found", nil
		}
		return "", fmt.Errorf("remove %q: %w", containerID, err)
	}
	return "removed", nil
}

// Restart یک container را ری‌استارت می‌کند.
func (c *Client) Restart(ctx context.Context, containerID string) (string, error) {
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
