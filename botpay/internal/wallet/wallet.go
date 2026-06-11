// Package wallet منطق کسب‌وکار کیف پول را پیاده‌سازی می‌کند.
package wallet

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/botpay/internal/consensus"
	"github.com/mrjvadi/creatorbot/botpay/internal/store"
	"github.com/mrjvadi/creatorbot/botpay/internal/ton"
)

// NATS subjects برای رویدادهای botpay
const (
	SubjectDeposit  = "botpay.deposit"        // واریز انجام شد
	SubjectPaid     = "botpay.paid"           // پرداخت به سرویس انجام شد
	SubjectInvoice  = "botpay.invoice.%s"     // invoice تأیید شد
)

// MinWithdrawNano حداقل مقدار برداشت (0.1 TON).
const MinWithdrawNano = 100_000_000

// NetworkFeeNano کارمزد شبکه TON برای برداشت.
const NetworkFeeNano = 10_000_000 // 0.01 TON

// Service service layer کیف پول.
type Service struct {
	store       *store.Store
	notify      Notifier
	nc          *natsclient.Client
	log         ports.Logger
	masterAddr  string
	guard       *consensus.Guard // محافظ consensus
}

func New(st *store.Store, nc *natsclient.Client, log ports.Logger, masterAddr string, guard *consensus.Guard) *Service {
	return &Service{store: st, nc: nc, log: log, masterAddr: masterAddr, guard: guard}
}

// SetNotifier ربات تلگرام را برای اعلان فوری ست می‌کند.
func (s *Service) SetNotifier(n Notifier) { s.notify = n }

// GetOrCreate wallet کاربر را پیدا یا می‌سازد.
func (s *Service) GetOrCreate(ctx context.Context, telegramID int64) (*store.Wallet, error) {
	// آدرس اختصاصی = آدرس master + نشان‌دهنده کاربر در comment
	// (در TON همه به یک آدرس می‌فرستند ولی comment متفاوت است)
	return s.store.GetOrCreateWallet(ctx, telegramID, s.masterAddr)
}

// CreateDepositInvoice یک invoice جدید برای واریز می‌سازد.
// کاربر باید amount TON به masterAddr با comment = invoice.Code بفرستد.
func (s *Service) CreateDepositInvoice(ctx context.Context, telegramID int64, amountNano int64, serviceID, ref string) (*store.Invoice, error) {
	w, err := s.GetOrCreate(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	code := genInvoiceCode()
	inv := &store.Invoice{
		WalletID:  w.ID,
		Code:      code,
		Amount:    amountNano,
		ServiceID: serviceID,
		Ref:       ref,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := s.store.CreateInvoice(ctx, inv); err != nil {
		return nil, err
	}

	s.log.Info("invoice created",
		ports.F("code", code),
		ports.F("amount_ton", float64(amountNano)/1e9),
		ports.F("telegram_id", telegramID))

	return inv, nil
}

// HandleDeposit پرداخت رسیده از TON blockchain را پردازش می‌کند.
// این تابع توسط ton.Watcher صدا زده می‌شود.
func (s *Service) HandleDeposit(ctx context.Context, event ton.DepositEvent) error {
	// پیدا کردن invoice با comment
	var w *store.Wallet
	var inv *store.Invoice

	if event.InvoiceCode != "" {
		var err error
		inv, err = s.store.FindPendingInvoiceByCode(ctx, event.InvoiceCode)
		if err != nil {
			return err
		}
	}

	if inv != nil {
		// بارگذاری wallet کامل با ID
		var err2 error
		w, err2 = s.store.GetWalletByID(ctx, inv.WalletID)
		if err2 != nil || w == nil {
			s.log.Error("HandleDeposit: wallet not found", ports.F("wallet_id", inv.WalletID))
			return fmt.Errorf("wallet not found for invoice")
		}
		event.WalletID = inv.WalletID.String()
	} else {
		// invoice نداشت → سعی کن کاربر را پیدا کن
		// (ممکنه کاربر بدون invoice واریز کرده باشه)
		s.log.Info("deposit without invoice",
			ports.F("from", event.FromAddr),
			ports.F("amount", event.AmountTON))
		// فعلاً نادیده می‌گیریم — در production log بزن و به ادمین اطلاع بده
		return nil
	}

	// ── بررسی consensus برای واریز ───────────────────────────
	if s.guard != nil {
		if err := s.guard.CheckDeposit(ctx, w.TONAddress, w.TelegramID, event.AmountNano, event.TxHash); err != nil {
			s.log.Error("deposit rejected by consensus", ports.F("err", err))
			return err
		}
	}

	// ثبت deposit در wallet
	tx, err := s.store.Deposit(ctx, w.ID, event.AmountNano,
		event.TxHash, event.FromAddr, "TON deposit")
	if err != nil {
		return fmt.Errorf("deposit to wallet: %w", err)
	}

	// تأیید invoice
	if inv != nil {
		s.store.ConfirmInvoice(ctx, inv.ID, event.TxHash)

		// publish رویداد برای سرویس درخواست‌دهنده
		s.nc.PublishCore(
			fmt.Sprintf(SubjectInvoice, inv.Code),
			map[string]any{
				"invoice_id":   inv.ID.String(),
				"code":         inv.Code,
				"ref":          inv.Ref,
				"service_id":   inv.ServiceID,
				"amount_nano":  event.AmountNano,
				"wallet_id":    w.ID.String(),
				"tx_hash":      event.TxHash,
			},
		)
	}

	// publish رویداد عمومی واریز
	s.nc.PublishCore(SubjectDeposit, ton.DepositEvent{
		WalletID:    w.ID.String(),
		AmountNano:  event.AmountNano,
		AmountTON:   event.AmountTON,
		TxHash:      event.TxHash,
		InvoiceCode: event.InvoiceCode,
	})

	s.log.Info("deposit processed",
		ports.F("wallet", w.ID),
		ports.F("amount_ton", event.AmountTON),
		ports.F("tx", event.TxHash))

	// ── اعلان فوری به کاربر ─────────────────────────────────
	if s.notify != nil {
		// موجودی جدید
		updatedWallet, _ := s.store.GetWalletByID(ctx, w.ID)
		newBalance := event.AmountTON
		if updatedWallet != nil {
			newBalance = updatedWallet.TotalTON()
		}
		msg := fmt.Sprintf(
			"✅ <b>واریز تأیید شد!</b>\n\n"+
				"💰 مبلغ دریافتی: <b>%.4f TON</b>\n"+
				"💳 موجودی جدید: <b>%.4f TON</b>",
			event.AmountTON, newBalance,
		)
		if err := s.notify.SendHTML(ctx, w.TelegramID, msg); err != nil {
			s.log.Error("notify deposit failed", ports.F("err", err))
		}
	}

	_ = tx
	return nil
}

// Pay کسر مبلغ از wallet برای پرداخت به سرویس.
// serviceID: شناسه سرویس (مثلاً "botmanager")
// ref: شناسه مرجع در آن سرویس (مثلاً plan_id)
func (s *Service) Pay(ctx context.Context, telegramID int64, amountNano int64, serviceID, ref, desc string) (*store.Transaction, error) {
	w, err := s.store.GetWallet(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, fmt.Errorf("wallet not found")
	}
	if !w.HasEnough(amountNano) {
		return nil, fmt.Errorf("insufficient balance: have %.4f TON, need %.4f TON",
			w.TotalTON(), float64(amountNano)/1e9)
	}

	// ── بررسی consensus قبل از کسر ──────────────────────────
	if s.guard != nil {
		if err := s.guard.CheckDeduct(ctx, w.TONAddress, telegramID, amountNano, serviceID, ref); err != nil {
			return nil, fmt.Errorf("consensus: %w", err)
		}
	}

	tx, err := s.store.Deduct(ctx, w.ID, amountNano, serviceID, ref, desc)
	if err != nil {
		return nil, err
	}

	// publish رویداد پرداخت
	s.nc.PublishCore(SubjectPaid, map[string]any{
		"wallet_id":  w.ID.String(),
		"service_id": serviceID,
		"ref":        ref,
		"amount_ton": float64(amountNano) / 1e9,
		"tx_id":      tx.ID.String(),
	})

	return tx, nil
}

// RequestWithdraw درخواست برداشت ایجاد می‌کند.
func (s *Service) RequestWithdraw(ctx context.Context, telegramID int64, toAddress string, amountNano int64, note string) (*store.WithdrawRequest, error) {
	if amountNano < MinWithdrawNano {
		return nil, fmt.Errorf("minimum withdrawal is %.1f TON", float64(MinWithdrawNano)/1e9)
	}

	totalNeeded := amountNano + NetworkFeeNano
	w, err := s.store.GetWallet(ctx, telegramID)
	if err != nil || w == nil {
		return nil, fmt.Errorf("wallet not found")
	}
	if !w.HasEnough(totalNeeded) {
		return nil, fmt.Errorf("insufficient balance (including %.4f TON fee)", float64(NetworkFeeNano)/1e9)
	}

	req := &store.WithdrawRequest{
		WalletID:  w.ID,
		ToAddress: toAddress,
		Amount:    amountNano,
		Fee:       NetworkFeeNano,
		Note:      note,
	}
	return req, s.store.CreateWithdraw(ctx, req)
}

// DepositInstructions دستورالعمل واریز برای کاربر.
func (s *Service) DepositInstructions(ctx context.Context, telegramID int64, amountNano int64, serviceID, ref string) (string, string, error) {
	inv, err := s.CreateDepositInvoice(ctx, telegramID, amountNano, serviceID, ref)
	if err != nil {
		return "", "", err
	}

	// آدرس پرداخت با deep link
	payURL := fmt.Sprintf(
		"ton://transfer/%s?amount=%d&text=%s",
		s.masterAddr, amountNano, inv.Code,
	)

	return inv.Code, payURL, nil
}

// ── helpers ────────────────────────────────────────────────

func genInvoiceCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return "PAY-" + strings.ToUpper(hex.EncodeToString(b))
}

// NanoToTON تبدیل nano-TON به TON.
func NanoToTON(nano int64) float64 { return float64(nano) / 1e9 }

// TONToNano تبدیل TON به nano-TON.
func TONToNano(ton float64) int64 { return int64(ton * 1e9) }

// Store برگرداندن store برای استفاده مستقیم.
func (s *Service) Store() *store.Store { return s.store }

// suppress
var _ = uuid.New
