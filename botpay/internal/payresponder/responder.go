// Package payresponder سرویس botpay را به‌عنوان NATS responder راه می‌اندازد.
// همه‌ی سرویس‌ها برای موجودی/پرداخت از این طریق (request/reply) با botpay حرف می‌زنند.
// هیچ سرویسی مستقیم به DB کیف پول دسترسی ندارد.
package payresponder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"

	"github.com/mrjvadi/creatorbot/botpay/internal/wallet"
	walletStore "github.com/mrjvadi/creatorbot/botpay/internal/store"
)

// Responder درخواست‌های NATS را به wallet service وصل می‌کند.
type Responder struct {
	nc     *natsclient.Client
	wallet *wallet.Service
	cache  ports.Cache // botpay مستقیم موجودی را در Redis می‌نویسد
	log    ports.Logger
}

func New(nc *natsclient.Client, w *wallet.Service, cache ports.Cache, log ports.Logger) *Responder {
	return &Responder{nc: nc, wallet: w, cache: cache, log: log}
}

// writeBalanceCache موجودی کاربر را مستقیم در Redis می‌نویسد (botpay = تنها نویسنده).
func (r *Responder) writeBalanceCache(ctx context.Context, w *walletStore.Wallet) {
	if r.cache == nil {
		return
	}
	resp := protocol.BalanceResponse{
		TelegramID: w.TelegramID,
		TONBalance: wallet.NanoToTON(w.TONBalance),
		Credit:     wallet.NanoToTON(w.Credit),
		Total:      wallet.NanoToTON(w.TONBalance + w.Credit),
		Frozen:     wallet.NanoToTON(w.Frozen),
		TONAddress: w.TONAddress,
	}
	if data, err := json.Marshal(resp); err == nil {
		_ = r.cache.Set(ctx, fmt.Sprintf("wallet:%d", w.TelegramID), string(data), 5*time.Minute)
	}
}

// publishPayCompleted رویداد اتمام پرداخت را به سرویس درخواست‌کننده می‌فرستد.
func (r *Responder) publishPayCompleted(ev protocol.PayCompletedEvent) {
	_ = r.nc.PublishCore(protocol.PayCompletedSubject(ev.ServiceID), ev)
}

// Start همه‌ی responder ها را روی NATS ثبت می‌کند (با queue group برای load balancing).
func (r *Responder) Start() error {
	if r.nc == nil {
		return fmt.Errorf("payresponder: nats client is nil")
	}

	if err := r.nc.QueueRespond(protocol.SubjPayBalance, protocol.SubjPayQueue, r.handleBalance); err != nil {
		return err
	}
	if err := r.nc.QueueRespond(protocol.SubjPayAuthorize, protocol.SubjPayQueue, r.handleAuthorize); err != nil {
		return err
	}
	if err := r.nc.QueueRespond(protocol.SubjPayDeduct, protocol.SubjPayQueue, r.handleDeduct); err != nil {
		return err
	}
	if err := r.nc.QueueRespond(protocol.SubjPayCredit, protocol.SubjPayQueue, r.handleCredit); err != nil {
		return err
	}
	if err := r.nc.QueueRespond(protocol.SubjPayTransfer, protocol.SubjPayQueue, r.handleTransfer); err != nil {
		return err
	}

	r.log.Info("payresponder started — listening on pay.* subjects")
	return nil
}

// authorize بررسی می‌کند سرویس درخواست‌کننده مجاز است.
// service_id یا یکی از سرویس‌های اصلی است، یا یک bot instance فعال (bot_<BotID>).
func (r *Responder) authorize(ctx context.Context, serviceID string) bool {
	return r.wallet.Store().ValidateServiceID(ctx, serviceID)
}

// ── handlers ──────────────────────────────────────────────────

func (r *Responder) handleBalance(data []byte) (any, error) {
	var req protocol.BalanceRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return protocol.BalanceResponse{Error: "bad request"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if !r.authorize(ctx, req.ServiceID) {
		r.log.Warn("pay.balance: unauthorized", ports.F("service", req.ServiceID))
		return protocol.BalanceResponse{Error: "unauthorized"}, nil
	}

	w, err := r.wallet.GetOrCreate(ctx, req.TelegramID)
	if err != nil {
		return protocol.BalanceResponse{Error: err.Error()}, nil
	}
	return protocol.BalanceResponse{
		TelegramID: req.TelegramID,
		TONBalance: wallet.NanoToTON(w.TONBalance),
		Credit:     wallet.NanoToTON(w.Credit),
		Total:      wallet.NanoToTON(w.TONBalance + w.Credit),
		Frozen:     wallet.NanoToTON(w.Frozen),
		TONAddress: w.TONAddress,
	}, nil
}

func (r *Responder) handleAuthorize(data []byte) (any, error) {
	var req protocol.AuthorizeRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return protocol.AuthorizeResponse{Error: "bad request"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !r.authorize(ctx, req.ServiceID) {
		return protocol.AuthorizeResponse{Authorized: false, Error: "unauthorized"}, nil
	}
	// سرویس معتبر است → مطمئن شو wallet برای کاربر وجود دارد
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := r.wallet.GetOrCreate(ctx, req.TelegramID); err != nil {
		return protocol.AuthorizeResponse{Authorized: false, Error: err.Error()}, nil
	}
	return protocol.AuthorizeResponse{Authorized: true}, nil
}

func (r *Responder) handleDeduct(data []byte) (any, error) {
	var req protocol.DeductRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return protocol.DeductResponse{Error: "bad request"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !r.authorize(ctx, req.ServiceID) {
		r.log.Warn("pay.deduct: unauthorized", ports.F("service", req.ServiceID))
		return protocol.DeductResponse{Error: "unauthorized"}, nil
	}

	nano := wallet.TONToNano(req.AmountTON)
	ref := req.Ref
	if ref == "" {
		ref = req.IdempotencyKey
	}
	if ref == "" {
		ref = req.Reason
	}
	tx, err := r.wallet.Pay(ctx, req.TelegramID, nano, req.ServiceID, ref, req.Reason)
	if err != nil {
		return protocol.DeductResponse{Success: false, Error: err.Error()}, nil
	}

	// موجودی جدید + بروزرسانی مستقیم Redis توسط botpay
	w, _ := r.wallet.GetOrCreate(ctx, req.TelegramID)
	newBal := 0.0
	if w != nil {
		newBal = wallet.NanoToTON(w.TONBalance + w.Credit)
		r.writeBalanceCache(ctx, w)
	}
	r.publishWalletUpdated(req.TelegramID, "payment")

	// event به سرویس درخواست‌کننده (pay.completed.<service_id>)
	txID := ""
	if tx != nil {
		txID = tx.ID.String()
	}
	r.publishPayCompleted(protocol.PayCompletedEvent{
		ServiceID:  req.ServiceID,
		TelegramID: req.TelegramID,
		AmountTON:  req.AmountTON,
		Reason:     req.Reason,
		Ref:        req.Ref,
		Metadata:   req.Metadata,
		TxID:       txID,
		Success:    true,
	})

	return protocol.DeductResponse{Success: true, NewBalance: newBal}, nil
}

func (r *Responder) handleCredit(data []byte) (any, error) {
	var req protocol.DeductRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return protocol.DeductResponse{Error: "bad request"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !r.authorize(ctx, req.ServiceID) {
		return protocol.DeductResponse{Error: "unauthorized"}, nil
	}

	w, err := r.wallet.GetOrCreate(ctx, req.TelegramID)
	if err != nil {
		return protocol.DeductResponse{Error: err.Error()}, nil
	}
	nano := wallet.TONToNano(req.AmountTON)
	if err := r.wallet.Store().AddCredit(ctx, w.ID, nano, req.Reason); err != nil {
		return protocol.DeductResponse{Success: false, Error: err.Error()}, nil
	}

	w2, _ := r.wallet.GetOrCreate(ctx, req.TelegramID)
	newBal := 0.0
	if w2 != nil {
		newBal = wallet.NanoToTON(w2.TONBalance + w2.Credit)
		r.writeBalanceCache(ctx, w2)
	}
	r.publishWalletUpdated(req.TelegramID, "refund")
	return protocol.DeductResponse{Success: true, NewBalance: newBal}, nil
}

// publishWalletUpdated به همه سرویس‌ها خبر می‌دهد موجودی کاربر تغییر کرد
// تا کش Redis خود را باطل کنند.
func (r *Responder) publishWalletUpdated(telegramID int64, reason string) {
	_ = r.nc.PublishCore(protocol.SubjWalletUpdated, protocol.WalletUpdatedEvent{
		TelegramID: telegramID,
		Reason:     reason,
	})
}

func (r *Responder) handleTransfer(data []byte) (any, error) {
	var req struct {
		protocol.PayRequest
		ToTelegramID int64   `json:"to_telegram_id"`
		AmountTON    float64 `json:"amount_ton"`
		Desc         string  `json:"desc"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return map[string]any{"error": "bad request"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !r.authorize(ctx, req.ServiceID) {
		return map[string]any{"error": "unauthorized"}, nil
	}

	nano := wallet.TONToNano(req.AmountTON)
	if err := r.wallet.Transfer(ctx, req.TelegramID, req.ToTelegramID, nano, req.Desc); err != nil {
		return map[string]any{"error": err.Error()}, nil
	}
	r.publishWalletUpdated(req.TelegramID, "transfer")
	r.publishWalletUpdated(req.ToTelegramID, "transfer")
	return map[string]any{"success": true}, nil
}
