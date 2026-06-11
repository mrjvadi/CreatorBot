// Package queue مدیریت صف deploy با concurrent limit.
// هر سرور حداکثر N deploy هم‌زمان می‌تواند انجام دهد.
// درخواست‌های اضافی در NATS منتظر می‌مانند تا slot آزاد شود.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

const (
	// MaxConcurrentDeploys حداکثر deploy هم‌زمان روی هر سرور.
	// با این مقدار Docker کمترین فشار را تجربه می‌کند.
	MaxConcurrentDeploys = 3

	// DeployTimeout حداکثر زمان برای یک deploy.
	DeployTimeout = 5 * time.Minute
)

// ExecuteFn تابعی که یک DeployCommand را اجرا می‌کند.
type ExecuteFn func(ctx context.Context, cmd protocol.DeployCommand) error

// Worker صف deploy را با semaphore مدیریت می‌کند.
type Worker struct {
	serverID string
	nc       *natsclient.Client
	execute  ExecuteFn
	log      ports.Logger

	// semaphore — حداکثر MaxConcurrentDeploys هم‌زمان
	sem chan struct{}

	mu      sync.Mutex
	active  int // تعداد deploy های فعال
	queued  int // تعداد در صف
}

func New(serverID string, nc *natsclient.Client, execute ExecuteFn, log ports.Logger) *Worker {
	return &Worker{
		serverID: serverID,
		nc:       nc,
		execute:  execute,
		log:      log,
		sem:      make(chan struct{}, MaxConcurrentDeploys),
	}
}

// Start شروع به دریافت از NATS و پردازش می‌کند.
func (w *Worker) Start(ctx context.Context) {
	subject := protocol.DeploySubject(w.serverID)

	w.log.Info("deploy worker started",
		ports.F("server", w.serverID),
		ports.F("max_concurrent", MaxConcurrentDeploys))

	// QueueSubscribe — اگه چند agentmanager روی یه سرور باشن، load balance می‌شه
	w.nc.QueueSubscribe(subject, "deploy-workers", func(data []byte) {
		var cmd protocol.DeployCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			w.log.Error("worker: unmarshal", ports.F("err", err))
			return
		}

		w.mu.Lock()
		w.queued++
		w.mu.Unlock()

		w.log.Info("deploy queued",
			ports.F("container", cmd.ContainerName),
			ports.F("active", w.active),
			ports.F("queued", w.queued))

		// اجرا در goroutine جداگانه — بلاک نمی‌کند
		go w.process(ctx, cmd)
	})

	<-ctx.Done()
	w.log.Info("deploy worker stopped")
}

// process یک deploy را با semaphore اجرا می‌کند.
func (w *Worker) process(ctx context.Context, cmd protocol.DeployCommand) {
	defer func() {
		w.mu.Lock()
		w.queued--
		w.mu.Unlock()
	}()

	// منتظر گرفتن slot
	select {
	case w.sem <- struct{}{}: // slot گرفتیم
		defer func() { <-w.sem }() // بعد از اتمام slot رو آزاد کن
	case <-ctx.Done():
		w.publishResult(ctx, cmd, false, "server shutting down")
		return
	}

	w.mu.Lock()
	w.active++
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		w.active--
		w.mu.Unlock()
	}()

	w.log.Info("deploy started",
		ports.F("container", cmd.ContainerName),
		ports.F("active_now", w.active))

	// deploy با timeout
	deployCtx, cancel := context.WithTimeout(ctx, DeployTimeout)
	defer cancel()

	start := time.Now()
	err := w.execute(deployCtx, cmd)
	elapsed := time.Since(start)

	if err != nil {
		w.log.Error("deploy failed",
			ports.F("container", cmd.ContainerName),
			ports.F("elapsed", elapsed.String()),
			ports.F("err", err))
		w.publishResult(ctx, cmd, false, err.Error())
		return
	}

	w.log.Info("deploy done",
		ports.F("container", cmd.ContainerName),
		ports.F("elapsed", elapsed.String()))
	w.publishResult(ctx, cmd, true, "")
}

// publishResult نتیجه را به apimanager/botmanager ارسال می‌کند.
func (w *Worker) publishResult(ctx context.Context, cmd protocol.DeployCommand, success bool, errMsg string) {
	result := protocol.ResultMsg{
		Type:          protocol.MsgResult,
		ServerID:      w.serverID,
		ContainerName: cmd.ContainerName,
		CommandType:   string(cmd.Type),
		Success:       success,
		Error:         errMsg,
		Timestamp:     time.Now().Unix(),
	}

	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	w.nc.Publish(pubCtx, protocol.ResultSubject(w.serverID), result)
}

// Status وضعیت فعلی worker.
func (w *Worker) Status() (active, queued int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.active, w.queued
}

// HealthCheck بررسی سلامت — اگه صف خیلی پر شد هشدار بده.
func (w *Worker) HealthCheck() string {
	active, queued := w.Status()
	if queued > MaxConcurrentDeploys*3 {
		return fmt.Sprintf("WARNING: queue length %d is high", queued)
	}
	return fmt.Sprintf("ok (active:%d queued:%d)", active, queued)
}
