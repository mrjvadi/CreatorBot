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

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

const (
	// MaxConcurrentDeploys حداکثر deploy هم‌زمان روی هر سرور.
	// با این مقدار Docker کمترین فشار را تجربه می‌کند.
	MaxConcurrentDeploys = 3

	// MaxQueuedDeploys سقف backlog محلی؛ callback NATS پس از پرشدن backpressure می‌گیرد.
	MaxQueuedDeploys = 30

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
	verify   *Verifier

	jobs chan protocol.DeployCommand

	mu     sync.Mutex
	active int // تعداد deploy های فعال
	queued int // تعداد در صف
}

func New(serverID string, nc *natsclient.Client, execute ExecuteFn, log ports.Logger, verify *Verifier) *Worker {
	return &Worker{
		serverID: serverID,
		nc:       nc,
		execute:  execute,
		log:      log,
		verify:   verify,
		jobs:     make(chan protocol.DeployCommand, MaxQueuedDeploys),
	}
}

// Start شروع به دریافت از NATS و پردازش می‌کند.
func (w *Worker) Start(ctx context.Context) {
	subject := protocol.DeploySubject(w.serverID)

	w.log.Info("deploy worker started",
		ports.F("server", w.serverID),
		ports.F("max_concurrent", MaxConcurrentDeploys))

	for i := 0; i < MaxConcurrentDeploys; i++ {
		go w.runWorker(ctx)
	}

	// QueueSubscribe — callback فقط enqueue می‌کند؛ channel محدود backpressure می‌دهد.
	w.nc.QueueSubscribe(subject, "deploy-workers", func(data []byte) {
		var cmd protocol.DeployCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			w.log.Error("worker: unmarshal", ports.F("err", err))
			return
		}

		// اصالت/تازگی/یک‌بارمصرف را پیش از enqueue تأیید کن. دستور جعلی یا
		// replay شده بی‌صدا رد می‌شود (waiterِ مجاز ندارد) و فقط log می‌شود.
		if err := w.verify.Check(cmd); err != nil {
			w.log.Warn("deploy rejected",
				ports.F("container", cmd.ContainerName),
				ports.F("service_id", cmd.ServiceID),
				ports.F("reason", err.Error()))
			return
		}

		w.mu.Lock()
		w.queued++
		w.mu.Unlock()
		select {
		case w.jobs <- cmd:
			w.mu.Lock()
			active, queued := w.active, w.queued
			w.mu.Unlock()
			w.log.Info("deploy queued", ports.F("container", cmd.ContainerName),
				ports.F("active", active), ports.F("queued", queued))
		case <-ctx.Done():
			w.mu.Lock()
			w.queued--
			w.mu.Unlock()
			w.publishResult(context.Background(), cmd, false, "server shutting down")
		}
	})

	<-ctx.Done()
	w.log.Info("deploy worker stopped")
}

// runWorker یکی از workerهای ثابت pool است؛ تعداد goroutineها با backlog رشد نمی‌کند.
func (w *Worker) runWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-w.jobs:
			w.mu.Lock()
			w.queued--
			w.active++
			w.mu.Unlock()
			w.process(ctx, cmd)
			w.mu.Lock()
			w.active--
			w.mu.Unlock()
		}
	}
}

func (w *Worker) process(ctx context.Context, cmd protocol.DeployCommand) {
	w.log.Info("deploy started", ports.F("container", cmd.ContainerName))
	deployCtx, cancel := context.WithTimeout(ctx, DeployTimeout)
	defer cancel()
	start := time.Now()
	err := w.execute(deployCtx, cmd)
	elapsed := time.Since(start)
	if err != nil {
		w.log.Error("deploy failed", ports.F("container", cmd.ContainerName),
			ports.F("elapsed", elapsed.String()), ports.F("err", err))
		w.publishResult(ctx, cmd, false, err.Error())
		return
	}
	w.log.Info("deploy done", ports.F("container", cmd.ContainerName), ports.F("elapsed", elapsed.String()))
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
