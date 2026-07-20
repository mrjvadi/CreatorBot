// Package sourceworker پاسخ‌دهیِ NATS برای قراردادِ source.worker.* را
// پیاده می‌کند (تعریفِ کامل در shared-core/protocol/source_worker.go).
// botmanager اینجا responder است؛ caller خودِ source-service است (ابزار
// داخلیِ MTProto/UserBot automation — نه یک BotInstance مشتری).
//
// این پکیج عمداً جدا از internal/tgbot است: هیچ‌کدام از سه subject به
// tele.Context نیاز ندارند؛ فقط به Store و دو رازِ مشترک (ServiceHMACSecret،
// EncryptKey) که main.go از قبل برای بخش‌های دیگر هم دارد.
package sourceworker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Config وابستگی‌های لازم برای پاسخ‌دهی.
type Config struct {
	// ServiceHMACSecret همان رازِ مادریِ pay.*/license.* — کلیدِ ارائه‌شده
	// توسط source-service با auth.ValidateServiceKey در برابر آن چک می‌شود.
	ServiceHMACSecret string
	// EncryptKey برای رمزگشاییِ AppHash/SessionKeyِ ذخیره‌شده در دیتابیس
	// (همان کلیدی که BotToken هم با آن رمز می‌شود).
	EncryptKey string
}

// Register هر سه responder/subscriber قرارداد را روی nc ثبت می‌کند. باید
// دقیقاً یک‌بار و فقط وقتی nc != nil از main.go صدا زده شود.
func Register(nc *natsclient.Client, st *store.Store, cfg Config, log ports.Logger) error {
	if err := nc.Respond(protocol.SubjSourceWorkerRegister, handleRegister(st, cfg, log)); err != nil {
		return err
	}
	if err := nc.Subscribe(protocol.SubjSourceWorkerHeartbeat, handleHeartbeat(st, cfg, log)); err != nil {
		return err
	}
	if err := nc.Respond(protocol.SubjSourceWorkerUpdate, handleUpdate(cfg, log)); err != nil {
		return err
	}
	log.Info("source-service worker responders registered")
	return nil
}

// handleRegister یک LicenseKey را به worker_id + اطلاعات تلگرام تبدیل می‌کند.
func handleRegister(st *store.Store, cfg Config, log ports.Logger) func([]byte) (any, error) {
	return func(data []byte) (any, error) {
		var req protocol.SourceWorkerRegisterRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return protocol.SourceWorkerRegisterResponse{Success: false, Error: "invalid request"}, nil
		}
		if !auth.ValidateServiceKey(cfg.ServiceHMACSecret, req.ServiceID, req.ServiceKey) {
			return protocol.SourceWorkerRegisterResponse{Success: false, Error: "unauthorized"}, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		swc, err := st.FindSourceWorkerConfigByLicenseKey(ctx, req.LicenseKey)
		if err != nil {
			log.Error("source worker register: lookup failed", ports.F("err", err))
			return protocol.SourceWorkerRegisterResponse{Success: false, Error: "internal error"}, nil
		}
		if swc == nil {
			return protocol.SourceWorkerRegisterResponse{Success: false, Error: "license not found"}, nil
		}
		if !swc.IsActive {
			return protocol.SourceWorkerRegisterResponse{Success: false, Error: "license inactive"}, nil
		}

		appHash, err := auth.Decrypt(swc.AppHash, cfg.EncryptKey)
		if err != nil {
			log.Error("source worker register: decrypt app_hash failed",
				ports.F("err", err), ports.F("worker_id", swc.WorkerID))
			return protocol.SourceWorkerRegisterResponse{Success: false, Error: "internal error"}, nil
		}
		sessionKey, err := auth.Decrypt(swc.SessionKey, cfg.EncryptKey)
		if err != nil {
			log.Error("source worker register: decrypt session_key failed",
				ports.F("err", err), ports.F("worker_id", swc.WorkerID))
			return protocol.SourceWorkerRegisterResponse{Success: false, Error: "internal error"}, nil
		}

		log.Info("source worker registered", ports.F("worker_id", swc.WorkerID))
		return protocol.SourceWorkerRegisterResponse{
			Success:  true,
			WorkerID: swc.WorkerID,
			Telegram: protocol.SourceWorkerTelegramCreds{
				AppID:      swc.AppID,
				AppHash:    appHash,
				Phone:      swc.Phone,
				SessionKey: sessionKey,
			},
		}, nil
	}
}

// handleHeartbeat فقط lastSeen/status را به‌روزرسانی می‌کند — fire-and-forget.
func handleHeartbeat(st *store.Store, cfg Config, log ports.Logger) func([]byte) {
	return func(data []byte) {
		var hb protocol.SourceWorkerHeartbeat
		if err := json.Unmarshal(data, &hb); err != nil {
			return
		}
		if hb.WorkerID == "" || !auth.ValidateServiceKey(cfg.ServiceHMACSecret, hb.ServiceID, hb.ServiceKey) {
			log.Warn("source worker heartbeat unauthorized")
			return
		}
		now := time.Now().Unix()
		if hb.Timestamp <= 0 || hb.Timestamp < now-120 || hb.Timestamp > now+30 {
			log.Warn("source worker heartbeat stale", ports.F("worker_id", hb.WorkerID))
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := st.UpdateSourceWorkerHeartbeat(ctx, hb.WorkerID, hb.Status); err != nil {
			log.Warn("source worker heartbeat update failed",
				ports.F("err", err), ports.F("worker_id", hb.WorkerID))
		}
	}
}

// handleUpdate رویدادِ source.worker.update را می‌پذیرد.
//
// محدودیتِ شناخته‌شده (عمدی، نه فراموشی): SourceWorkerUpdateRequest فقط یک
// correlation ID (متعلق به task.Envelope در خودِ source-service) حمل
// می‌کند — نه WorkerID و نه هیچ شناسه‌ی دیگری که به یک BotInstance/owner در
// botmanager وصل شود. تا وقتی سمتِ dispatchِ taskها (جایی که این correlation
// ID اول بار صادر می‌شود) در botmanager پیاده نشده، امکانِ «relay به رباتِ
// کاربر» که در source_worker.go توضیح داده شده وجود ندارد؛ فعلاً فقط لاگ
// می‌شود. وقتی آن سمت پیاده شد، اینجا باید یک جدولِ correlation-id →
// owner/chat اضافه و از همان‌جا پیام به کاربر ارسال شود.
func handleUpdate(cfg Config, log ports.Logger) func([]byte) (any, error) {
	return func(data []byte) (any, error) {
		var req protocol.SourceWorkerUpdateRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return protocol.SourceWorkerUpdateResponse{Success: false, Error: "invalid request"}, nil
		}
		if !auth.ValidateServiceKey(cfg.ServiceHMACSecret, req.ServiceID, req.ServiceKey) {
			return protocol.SourceWorkerUpdateResponse{Success: false, Error: "unauthorized"}, nil
		}
		log.Info("source worker update received", ports.F("id", req.ID), ports.F("tags", req.Tags))
		return protocol.SourceWorkerUpdateResponse{Success: true}, nil
	}
}
