package main

import (
	"context"
	"fmt"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/agentmanager/internal/queue"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
)

type Config struct {
	ServerID     string `mapstructure:"SERVER_ID"`
	NatsURL      string `mapstructure:"NATS_URL"`
	NatsUser     string `mapstructure:"NATS_USERNAME"`
	NatsPass     string `mapstructure:"NATS_PASSWORD"`
	HeartbeatSec int    `mapstructure:"HEARTBEAT_INTERVAL_SEC"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	if cfg.ServerID == "" {
		log.Fatal("SERVER_ID is required")
	}
	if cfg.HeartbeatSec == 0 {
		cfg.HeartbeatSec = 5
	}

	nc, err := natsclient.New(natsclient.Config{
		URL:      cfg.NatsURL,
		Username: cfg.NatsUser,
		Password: cfg.NatsPass,
	})
	if err != nil {
		log.Fatal("nats connect failed", ports.F("err", err))
	}
	defer nc.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── Deploy Worker Pool (concurrent limit) ───────────────
	worker := queue.New(cfg.ServerID, nc, func(ctx context.Context, cmd protocol.DeployCommand) error {
		_, err := handleCommand(ctx, nc, cfg.ServerID, cmd, log)
		return err
	}, log)
	go worker.Start(ctx)
	log.Info("deploy worker started",
		ports.F("subject", protocol.DeploySubject(cfg.ServerID)),
		ports.F("max_concurrent", queue.MaxConcurrentDeploys))

	// ── Heartbeat loop ────────────────────────────────────────
	go heartbeatLoop(ctx, nc, cfg.ServerID,
		time.Duration(cfg.HeartbeatSec)*time.Second, log)

	log.Info("agentmanager started",
		ports.F("server_id", cfg.ServerID),
		ports.F("nats", cfg.NatsURL))

	<-ctx.Done()
	log.Info("agentmanager stopped")
}

func handleCommand(ctx context.Context, nc *natsclient.Client, serverID string,
	cmd protocol.DeployCommand, log ports.Logger) (string, error) {

	log.Info("executing command",
		ports.F("type", cmd.Type),
		ports.F("container", cmd.ContainerName))

	var out string
	var execErr error
	var finalErr error

	switch cmd.Type {
	case protocol.MsgDeploy:
		out, execErr = deploy(ctx, cmd)
	case protocol.MsgStop:
		out, execErr = runCmd(ctx, "docker", "stop", "--time=10", cmd.ContainerID)
	case protocol.MsgRemove:
		out, execErr = runCmd(ctx, "docker", "rm", "-f", cmd.ContainerID)
	case protocol.MsgRestart:
		out, execErr = runCmd(ctx, "docker", "restart", cmd.ContainerID)
	default:
		execErr = fmt.Errorf("unknown command: %s", cmd.Type)
	}

	errStr := ""
	if execErr != nil {
		errStr = execErr.Error()
		finalErr = execErr
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
	nc.Publish(pubCtx, protocol.ResultSubject(serverID), result)

	if execErr != nil {
		log.Error("command failed",
			ports.F("type", cmd.Type), ports.F("err", execErr))
	} else {
		log.Info("command done",
			ports.F("type", cmd.Type), ports.F("container", cmd.ContainerName))
	}
	return result.Output, finalErr
}

func deploy(ctx context.Context, cmd protocol.DeployCommand) (string, error) {
	image := cmd.ImageName + ":" + cmd.ImageTag
	runCmd(ctx, "docker", "pull", image)
	runCmd(ctx, "docker", "rm", "-f", cmd.ContainerName)

	args := []string{"run", "-d", "--restart=unless-stopped",
		"--name=" + cmd.ContainerName}
	if cmd.NetworkName != "" {
		args = append(args, "--network="+cmd.NetworkName)
	}
	for k, v := range cmd.EnvVars {
		args = append(args, "-e", k+"="+v)
	}
	args = append(args, image)
	return runCmd(ctx, "docker", args...)
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		return s, fmt.Errorf("%s", s)
	}
	return s, nil
}

func heartbeatLoop(ctx context.Context, nc *natsclient.Client, serverID string,
	interval time.Duration, log ports.Logger) {

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	log.Info("heartbeat started",
		ports.F("server", serverID), ports.F("interval", interval))

	send := func() {
		msg := protocol.HeartbeatMsg{
			Type:       protocol.MsgHeartbeat,
			ServerID:   serverID,
			Timestamp:  time.Now().Unix(),
			Containers: listContainers(),
		}
		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := nc.Publish(pubCtx, protocol.HeartbeatSubject(serverID), msg); err != nil {
			// context canceled موقع shutdown نرمال است
			if pubCtx.Err() == nil {
				log.Error("heartbeat publish failed", ports.F("err", err))
			}
		} else {
			log.Info("heartbeat sent", ports.F("containers", len(msg.Containers)))
		}
	}

	send()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			send()
		}
	}
}

func listContainers() []protocol.ContainerStatus {
	out, err := exec.Command("docker", "ps", "-a",
		"--format", "{{.Names}}\t{{.Image}}\t{{.State}}\t{{.Status}}").Output()
	if err != nil {
		return nil
	}
	var list []protocol.ContainerStatus
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 || line == "" {
			continue
		}
		list = append(list, protocol.ContainerStatus{
			Name: parts[0], Image: parts[1],
			State: parts[2], Status: parts[3],
		})
	}
	return list
}
