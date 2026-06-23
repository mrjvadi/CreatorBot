package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mrjvadi/creatorbot/agentmanager/internal/docker"
	"github.com/mrjvadi/creatorbot/agentmanager/internal/heartbeat"
	"github.com/mrjvadi/creatorbot/agentmanager/internal/queue"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/metrics"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	ServerID     string `mapstructure:"SERVER_ID"`
	NatsURL      string `mapstructure:"NATS_URL"`
	NatsUser     string `mapstructure:"NATS_USERNAME"`
	NatsPass     string `mapstructure:"NATS_PASSWORD"`
	HeartbeatSec int    `mapstructure:"HEARTBEAT_INTERVAL_SEC"`

	// ── تنظیمات امنیتی/منابع پیش‌فرض container ها ──
	// AllowedImages لیست prefix های مجاز، جداشده با کاما (whitelist اجباری).
	AllowedImages    string  `mapstructure:"ALLOWED_IMAGES"`
	DefaultMemoryMB  int64   `mapstructure:"DEFAULT_MEMORY_MB"`
	DefaultCPUs      float64 `mapstructure:"DEFAULT_CPUS"`
	DefaultPidsLimit int64   `mapstructure:"DEFAULT_PIDS_LIMIT"`
	ReadonlyRootfs   bool    `mapstructure:"READONLY_ROOTFS"`
	DefaultTmpfsMB   int64   `mapstructure:"DEFAULT_TMPFS_MB"`
}

func main() {
	// پیش‌فرض‌های امن؛ اگر در .env تنظیم نشوند همین‌ها می‌مانند.
	cfg := Config{
		DefaultMemoryMB:  512,
		DefaultCPUs:      1.0,
		DefaultPidsLimit: 256,
		ReadonlyRootfs:   true,
		DefaultTmpfsMB:   64,
	}
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	if cfg.ServerID == "" {
		log.Fatal("SERVER_ID is required")
	}
	if cfg.HeartbeatSec == 0 {
		cfg.HeartbeatSec = 5
	}

	// ── ساخت SecurityPolicy از config ──
	var allowed []string
	for _, p := range strings.Split(cfg.AllowedImages, ",") {
		if p = strings.TrimSpace(p); p != "" {
			allowed = append(allowed, p)
		}
	}
	policy := docker.SecurityPolicy{
		AllowedImages:    allowed,
		DefaultMemoryMB:  cfg.DefaultMemoryMB,
		DefaultCPUs:      cfg.DefaultCPUs,
		DefaultPidsLimit: cfg.DefaultPidsLimit,
		ReadonlyRootfs:   cfg.ReadonlyRootfs,
		DefaultTmpfsMB:   cfg.DefaultTmpfsMB,
	}

	// ── NATS ─────────────────────────────────────────────────────
	nc, err := natsclient.New(natsclient.Config{
		URL:      cfg.NatsURL,
		Username: cfg.NatsUser,
		Password: cfg.NatsPass,
		Name:     "agentmanager",
	})
	if err != nil {
		log.Fatal("nats connect failed", ports.F("err", err))
	}
	defer nc.Close()

	// ── Docker SDK client ─────────────────────────────────────────
	// هیچ exec.Command وجود ندارد — مستقیم با Docker daemon از طریق socket
	dockerClient, err := docker.NewClient(log, policy)
	if err != nil {
		log.Fatal("docker sdk init failed", ports.F("err", err))
	}
	defer dockerClient.Close()
	log.Info("docker sdk connected")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── JetStream streams ─────────────────────────────────────────
	setupCtx, cancelSetup := context.WithTimeout(ctx, 10*time.Second)
	if err := nc.EnsureStream(setupCtx, protocol.StreamAgent, []string{"agent.>"}); err != nil {
		log.Fatal("ensure AGENT stream failed", ports.F("err", err))
	}
	if err := nc.EnsureStream(setupCtx, protocol.StreamDeploy, []string{"deploy.>"}); err != nil {
		log.Fatal("ensure DEPLOY stream failed", ports.F("err", err))
	}
	cancelSetup()
	log.Info("jetstream streams ready")

	// ── Deploy Worker Pool ────────────────────────────────────────
	worker := queue.New(cfg.ServerID, nc, func(ctx context.Context, cmd protocol.DeployCommand) error {
		_, err := handleCommand(ctx, nc, cfg.ServerID, cmd, dockerClient, log)
		return err
	}, log)
	go worker.Start(ctx)
	log.Info("deploy worker started",
		ports.F("subject", protocol.DeploySubject(cfg.ServerID)),
		ports.F("max_concurrent", queue.MaxConcurrentDeploys))

	// ── Heartbeat (Docker SDK — نه exec.Command) ──────────────────
	hb := heartbeat.New(
		cfg.ServerID, nc, dockerClient, log,
		time.Duration(cfg.HeartbeatSec)*time.Second,
	)
	go hb.Run(ctx)

	log.Info("agentmanager started",
		ports.F("server_id", cfg.ServerID),
		ports.F("nats", cfg.NatsURL))

	// ── Metrics + Health ──────────────────────────────────────────
	metrics.ServeMetrics(":9093")
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok":true,"service":"agentmanager"}`))
		})
		if err := http.ListenAndServe(":8096", mux); err != nil {
			log.Error("health server failed", ports.F("err", err))
		}
	}()

	<-ctx.Done()
	log.Info("agentmanager stopped")
}

// handleCommand یک DeployCommand را با Docker SDK اجرا می‌کند.
// هیچ exec.Command یا shell string concatenation وجود ندارد.
func handleCommand(
	ctx context.Context,
	nc *natsclient.Client,
	serverID string,
	cmd protocol.DeployCommand,
	dockerClient *docker.Client,
	log ports.Logger,
) (string, error) {

	log.Info("executing command",
		ports.F("type", cmd.Type),
		ports.F("container", cmd.ContainerName))

	var out string
	var execErr error

	switch cmd.Type {
	case protocol.MsgDeploy:
		out, execErr = dockerClient.Deploy(ctx, cmd)
	case protocol.MsgStop:
		out, execErr = dockerClient.Stop(ctx, cmd.ContainerID)
	case protocol.MsgRemove:
		out, execErr = dockerClient.Remove(ctx, cmd.ContainerID)
	case protocol.MsgRestart:
		out, execErr = dockerClient.Restart(ctx, cmd.ContainerID)
	default:
		execErr = fmt.Errorf("unknown command: %s", cmd.Type)
	}

	// ── نتیجه را به botmanager publish می‌کنیم ───────────────────
	errStr := ""
	if execErr != nil {
		errStr = execErr.Error()
	}
	result := protocol.ResultMsg{
		Type:          protocol.MsgResult,
		ServerID:      serverID,
		ContainerName: cmd.ContainerName,
		CommandType:   string(cmd.Type),
		Success:       execErr == nil,
		Output:        out,
		Error:         errStr,
		Timestamp:     time.Now().Unix(),
	}
	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = nc.Publish(pubCtx, protocol.ResultSubject(serverID), result)

	if execErr != nil {
		log.Error("command failed",
			ports.F("type", cmd.Type),
			ports.F("err", execErr))
		if cmd.Type == protocol.MsgDeploy {
			metrics.IncDeploy("bot", "failed")
		}
	} else {
		log.Info("command done",
			ports.F("type", cmd.Type),
			ports.F("container", cmd.ContainerName))
		if cmd.Type == protocol.MsgDeploy {
			metrics.IncDeploy("bot", "success")
			metrics.ActiveInstances.Inc()
		}
	}

	return out, execErr
}
