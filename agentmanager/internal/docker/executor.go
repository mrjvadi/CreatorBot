// Package docker دستورات Docker دریافتی از Centrifugo را اجرا می‌کند.
package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"os/exec"

	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/agent"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Executor دستورات Docker را روی daemon محلی اجرا کرده و نتیجه را publish می‌کند.
type Executor struct {
	serverID string
	notifier ports.Notifier
	log      ports.Logger
}

func New(serverID string, notifier ports.Notifier, log ports.Logger) *Executor {
	return &Executor{serverID: serverID, notifier: notifier, log: log}
}

// Handle یک docker.Command را اجرا کرده و نتیجه را به کانال "botmanager" publish می‌کند.
func (e *Executor) Handle(ctx context.Context, cmd sharedocker.Command) error {
	e.log.Info("executing", ports.F("type", cmd.Type), ports.F("container", cmd.ContainerName+cmd.ContainerID))

	var out string
	var execErr error

	switch cmd.Type {
	case sharedocker.CmdDeploy:
		out, execErr = e.deploy(ctx, cmd)
	case sharedocker.CmdStart:
		out, execErr = e.run(ctx, "docker", "start", cmd.ContainerID)
	case sharedocker.CmdStop:
		out, execErr = e.run(ctx, "docker", "stop", "--time=10", cmd.ContainerID)
	case sharedocker.CmdRemove:
		out, execErr = e.run(ctx, "docker", "rm", "-f", cmd.ContainerID)
	case sharedocker.CmdLogs:
		out, execErr = e.run(ctx, "docker", "logs", "--tail=100", cmd.ContainerID)
	case sharedocker.CmdInspect:
		out, execErr = e.inspect(ctx, cmd.ContainerID)
	default:
		execErr = fmt.Errorf("unknown command: %s", cmd.Type)
	}

	// ارسال نتیجه به botmanager
	containerID := cmd.ContainerID
	if containerID == "" {
		containerID = cmd.ContainerName
	}
	errMsg := ""
	if execErr != nil {
		errMsg = execErr.Error()
	}
	result := agent.NewCommandResult(
		string(cmd.Type), e.serverID, containerID,
		execErr == nil, out, errMsg,
	)
	if e.notifier != nil {
		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := e.notifier.Publish(pubCtx, "botmanager", result); err != nil {
			e.log.Error("publish result failed", ports.F("err", err))
		}
	}

	return execErr
}

func (e *Executor) deploy(ctx context.Context, cmd sharedocker.Command) (string, error) {
	image := cmd.ImageName + ":" + cmd.ImageTag

	e.log.Info("pulling image", ports.F("image", image))
	if _, err := e.run(ctx, "docker", "pull", image); err != nil {
		return "", fmt.Errorf("pull %s: %w", image, err)
	}

	// idempotent: container قدیمی رو حذف کن
	e.run(ctx, "docker", "rm", "-f", cmd.ContainerName) //nolint

	args := []string{"run", "-d", "--restart=unless-stopped", "--name=" + cmd.ContainerName}
	if cmd.NetworkName != "" {
		args = append(args, "--network="+cmd.NetworkName)
	}
	for k, v := range cmd.EnvVars {
		args = append(args, "-e", k+"="+v)
	}
	args = append(args, image)

	out, err := e.run(ctx, "docker", args...)
	if err != nil {
		return out, fmt.Errorf("run: %w", err)
	}
	e.log.Info("deployed", ports.F("container", cmd.ContainerName), ports.F("image", image))
	return strings.TrimSpace(out), nil
}

func (e *Executor) inspect(ctx context.Context, containerID string) (string, error) {
	out, err := e.run(ctx, "docker", "inspect", "--format={{json .State}}", containerID)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(strings.TrimSpace(out)), "", "  "); err != nil {
		return out, nil
	}
	return buf.String(), nil
}

func (e *Executor) run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		e.log.Error("exec failed",
			ports.F("cmd", name+" "+strings.Join(args, " ")),
			ports.F("out", outStr))
		return outStr, fmt.Errorf("%s", outStr)
	}
	return outStr, nil
}
