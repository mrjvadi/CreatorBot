// Package heartbeat وضعیت سرور را به‌صورت دوره‌ای به botmanager ارسال می‌کند.
// هیچ exec.Command اینجا وجود ندارد — همه از Docker SDK و /proc خوانده می‌شود.
package heartbeat

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mrjvadi/creatorbot/agentmanager/internal/docker"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Runner heartbeat ها را در فواصل منظم ارسال می‌کند.
type Runner struct {
	serverID string
	nc       *natsclient.Client
	docker   *docker.Client
	log      ports.Logger
	interval time.Duration
}

func New(serverID string, nc *natsclient.Client, d *docker.Client, log ports.Logger, interval time.Duration) *Runner {
	return &Runner{
		serverID: serverID,
		nc:       nc,
		docker:   d,
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
	// لیست container ها از Docker SDK (نه exec.Command)
	listCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	containers, err := r.docker.ListContainers(listCtx)
	cancel()
	if err != nil {
		r.log.Error("list containers failed", ports.F("err", err))
		containers = nil
	}

	msg := protocol.HeartbeatMsg{
		Type:       protocol.MsgHeartbeat,
		ServerID:   r.serverID,
		Timestamp:  time.Now().Unix(),
		Containers: containers,
	}

	pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pubCancel()

	if err := r.nc.Publish(pubCtx, protocol.HeartbeatSubject(r.serverID), msg); err != nil {
		if pubCtx.Err() == nil {
			r.log.Error("heartbeat publish failed", ports.F("err", err))
		}
	} else {
		r.log.Info("heartbeat sent", ports.F("containers", len(containers)))
	}
}

// SystemStats اطلاعات سیستم را از /proc می‌خواند — بدون اجرای هیچ دستور خارجی.
func SystemStats() map[string]string {
	stats := map[string]string{}

	// Load average از /proc/loadavg
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		if parts := strings.Fields(string(data)); len(parts) >= 1 {
			stats["load_1m"] = parts[0]
		}
	}

	// RAM از /proc/meminfo (کیلوبایت)
	if f, err := os.Open("/proc/meminfo"); err == nil {
		defer f.Close()
		memInfo := map[string]int64{}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}
			key := strings.TrimSuffix(parts[0], ":")
			val, err := strconv.ParseInt(parts[1], 10, 64)
			if err == nil {
				memInfo[key] = val
			}
		}
		total := memInfo["MemTotal"]
		avail := memInfo["MemAvailable"]
		if total > 0 {
			used := total - avail
			stats["ram_total_mb"] = strconv.FormatInt(total/1024, 10)
			stats["ram_used_mb"] = strconv.FormatInt(used/1024, 10)
			stats["ram_pct"] = strconv.FormatFloat(float64(used)/float64(total)*100, 'f', 1, 64)
		}
	}

	return stats
}
