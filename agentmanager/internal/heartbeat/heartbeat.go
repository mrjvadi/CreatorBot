// Package heartbeat وضعیت سرور را به‌صورت دوره‌ای به botmanager ارسال می‌کند.
package heartbeat

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mrjvadi/creatorbot/shared-core/agent"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Runner heartbeat ها را در فواصل منظم ارسال می‌کند.
type Runner struct {
	serverID string
	channel  string
	notifier ports.Notifier
	log      ports.Logger
	interval time.Duration
}

func New(serverID string, notifier ports.Notifier, log ports.Logger, interval time.Duration) *Runner {
	return &Runner{
		serverID: serverID,
		channel:  "botmanager",
		notifier: notifier,
		log:      log,
		interval: interval,
	}
}

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.log.Info("heartbeat started",
		ports.F("server", r.serverID),
		ports.F("interval", r.interval.String()))

	r.send(ctx) // اولین heartbeat فوری

	for {
		select {
		case <-ctx.Done():
			r.log.Info("heartbeat stopped")
			return
		case <-ticker.C:
			r.send(ctx)
		}
	}
}

func (r *Runner) send(ctx context.Context) {
	containers := listContainers()
	payload := agent.NewHeartbeat(r.serverID, containers)

	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := r.notifier.Publish(pubCtx, r.channel, payload); err != nil {
		r.log.Error("heartbeat publish failed", ports.F("err", err))
	} else {
		r.log.Info("heartbeat sent", ports.F("containers", len(containers)))
	}
}

func listContainers() []agent.ContainerStatus {
	out, err := exec.Command("docker", "ps", "-a",
		"--format", "{{.Names}}\t{{.Image}}\t{{.State}}\t{{.Status}}").Output()
	if err != nil {
		return nil
	}
	var containers []agent.ContainerStatus
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		containers = append(containers, agent.ContainerStatus{
			Name:   parts[0],
			Image:  parts[1],
			State:  parts[2],
			Status: parts[3],
		})
	}
	return containers
}

// SystemStats اطلاعات سیستم (load average, RAM) را برمی‌گرداند.
func SystemStats() map[string]string {
	stats := map[string]string{}
	if out, err := exec.Command("cat", "/proc/loadavg").Output(); err == nil {
		if parts := strings.Fields(string(out)); len(parts) >= 1 {
			stats["load_1m"] = parts[0]
		}
	}
	if out, err := exec.Command("free", "-m").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "Mem:") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					stats["ram_total_mb"] = parts[1]
					stats["ram_used_mb"] = parts[2]
					if total, e1 := strconv.ParseFloat(parts[1], 64); e1 == nil {
						if used, e2 := strconv.ParseFloat(parts[2], 64); e2 == nil && total > 0 {
							stats["ram_pct"] = strconv.FormatFloat(used/total*100, 'f', 1, 64)
						}
					}
				}
				break
			}
		}
	}
	return stats
}
