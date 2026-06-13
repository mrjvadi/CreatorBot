package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) *Store { return &Store{db: db} }

// ── Wallet ─────────────────────────────────────────────────

func (s *Store) GetOrCreateWallet(ctx context.Context, telegramID int64, tonAddress string) (*Wallet, error) {
	var w Wallet
	err := s.db.WithContext(ctx).
		Where("telegram_id = ?", telegramID).
		First(&w).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		w = Wallet{
			TelegramID: telegramID,
			TONAddress: tonAddress,
			IsActive:   true,
		}
		if err := s.db.WithContext(ctx).Create(&w).Error; err != nil {
			return nil, err
		}
		return &w, nil
	}
	return &w, err
}

func (s *Store) GetWallet(ctx context.Context, telegramID int64) (*Wallet, error) {
	var w Wallet
	err := s.db.WithContext(ctx).Where("telegram_id = ?", telegramID).First(&w).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &w, err
}

func (s *Store) GetWalletByID(ctx context.Context, id uuid.UUID) (*Wallet, error) {
	var w Wallet
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&w).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &w, err
}

func (s *Store) GetWalletByAddress(ctx context.Context, address string) (*Wallet, error) {
	var w Wallet
	err := s.db.WithContext(ctx).Where("ton_address = ?", address).First(&w).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &w, err
}

// Deposit واریز TON به wallet — atomic با transaction.
func (s *Store) Deposit(ctx context.Context, walletID uuid.UUID, amountNano int64, txHash, fromAddr, desc string) (*Transaction, error) {
	var tx *Transaction
	err := s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		// بررسی تکراری نبودن tx
		var existing Transaction
		if err := db.Where("tx_hash = ?", txHash).First(&existing).Error; err == nil {
			tx = &existing
			return nil // تکراری — ok
		}

		// آپدیت موجودی
		if err := db.Model(&Wallet{}).
			Where("id = ?", walletID).
			UpdateColumn("ton_balance", gorm.Expr("ton_balance + ?", amountNano)).
			Error; err != nil {
			return err
		}

		// ثبت تراکنش
		now := time.Now()
		tx = &Transaction{
			WalletID:    walletID,
			Type:        TxDeposit,
			Status:      TxConfirmed,
			Amount:      amountNano,
			TxHash:      txHash,
			FromAddress: fromAddr,
			Description: desc,
			ConfirmedAt: &now,
		}
		return db.Create(tx).Error
	})
	return tx, err
}

// Deduct کسر از موجودی — ابتدا از اعتبار، سپس TON.
func (s *Store) Deduct(ctx context.Context, walletID uuid.UUID, amountNano int64, serviceID, ref, desc string) (*Transaction, error) {
	var tx *Transaction
	err := s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		var w Wallet
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", walletID).First(&w).Error; err != nil {
			return err
		}

		if !w.HasEnough(amountNano) {
			return fmt.Errorf("insufficient balance")
		}

		// ابتدا از اعتبار کم کن
		var creditUsed, tonUsed int64
		if w.Credit >= amountNano {
			creditUsed = amountNano
		} else {
			creditUsed = w.Credit
			tonUsed = amountNano - creditUsed
		}

		updates := map[string]any{}
		if creditUsed > 0 {
			updates["credit"] = gorm.Expr("credit - ?", creditUsed)
		}
		if tonUsed > 0 {
			updates["ton_balance"] = gorm.Expr("ton_balance - ?", tonUsed)
		}

		if err := db.Model(&Wallet{}).Where("id = ?", walletID).
			Updates(updates).Error; err != nil {
			return err
		}

		now := time.Now()
		tx = &Transaction{
			WalletID:    walletID,
			Type:        TxPayment,
			Status:      TxConfirmed,
			Amount:      -amountNano,
			ServiceID:   serviceID,
			Ref:         ref,
			Description: desc,
			ConfirmedAt: &now,
		}
		return db.Create(tx).Error
	})
	return tx, err
}

// AddCredit افزایش اعتبار داخلی توسط ادمین.
func (s *Store) AddCredit(ctx context.Context, walletID uuid.UUID, amountNano int64, desc string) error {
	return s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		if err := db.Model(&Wallet{}).Where("id = ?", walletID).
			UpdateColumn("credit", gorm.Expr("credit + ?", amountNano)).Error; err != nil {
			return err
		}
		now := time.Now()
		return db.Create(&Transaction{
			WalletID:    walletID,
			Type:        TxCreditAdd,
			Status:      TxConfirmed,
			Amount:      amountNano,
			Description: desc,
			ConfirmedAt: &now,
		}).Error
	})
}

// ── Invoice ────────────────────────────────────────────────

func (s *Store) CreateInvoice(ctx context.Context, inv *Invoice) error {
	return s.db.WithContext(ctx).Create(inv).Error
}

func (s *Store) FindInvoiceByCode(ctx context.Context, code string) (*Invoice, error) {
	var inv Invoice
	err := s.db.WithContext(ctx).Where("code = ?", code).First(&inv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &inv, err
}

// FindPendingInvoiceByCode فقط invoiceهای pending را برمی‌گرداند.
func (s *Store) FindPendingInvoiceByCode(ctx context.Context, code string) (*Invoice, error) {
	var inv Invoice
	err := s.db.WithContext(ctx).Where("code = ? AND status = 'pending'", code).First(&inv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &inv, err
}

func (s *Store) ConfirmInvoice(ctx context.Context, invoiceID uuid.UUID, txHash string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Invoice{}).
		Where("id = ?", invoiceID).
		Updates(map[string]any{
			"status":   InvoicePaid,
			"tx_hash":  txHash,
			"paid_at":  &now,
		}).Error
}

func (s *Store) ExpireOldInvoices(ctx context.Context) error {
	return s.db.WithContext(ctx).Model(&Invoice{}).
		Where("status = 'pending' AND expires_at < ?", time.Now()).
		Update("status", InvoiceExpired).Error
}

// ── Transaction history ────────────────────────────────────

func (s *Store) ListTransactions(ctx context.Context, walletID uuid.UUID, limit int) ([]Transaction, error) {
	var txs []Transaction
	err := s.db.WithContext(ctx).
		Where("wallet_id = ?", walletID).
		Order("created_at DESC").
		Limit(limit).
		Find(&txs).Error
	return txs, err
}

// ── Withdraw ───────────────────────────────────────────────

func (s *Store) CreateWithdraw(ctx context.Context, req *WithdrawRequest) error {
	return s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		// بلوک کردن موجودی
		if err := db.Model(&Wallet{}).Where("id = ?", req.WalletID).
			UpdateColumn("frozen", gorm.Expr("frozen + ?", req.Amount+req.Fee)).
			Error; err != nil {
			return err
		}
		return db.Create(req).Error
	})
}

func (s *Store) ListPendingWithdrawals(ctx context.Context) ([]WithdrawRequest, error) {
	var reqs []WithdrawRequest
	err := s.db.WithContext(ctx).
		Where("status = 'pending'").
		Order("created_at ASC").
		Find(&reqs).Error
	return reqs, err
}

func (s *Store) CompleteWithdraw(ctx context.Context, id uuid.UUID, txHash string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		var req WithdrawRequest
		if err := db.Where("id = ?", id).First(&req).Error; err != nil {
			return err
		}

		// کسر از موجودی و آزاد کردن frozen
		if err := db.Model(&Wallet{}).Where("id = ?", req.WalletID).
			Updates(map[string]any{
				"ton_balance": gorm.Expr("ton_balance - ?", req.Amount+req.Fee),
				"frozen":      gorm.Expr("frozen - ?", req.Amount+req.Fee),
			}).Error; err != nil {
			return err
		}

		// آپدیت وضعیت
		return db.Model(&WithdrawRequest{}).Where("id = ?", id).
			Updates(map[string]any{
				"status":       WithdrawDone,
				"tx_hash":      txHash,
				"processed_at": &now,
			}).Error
	})
}

func (s *Store) RejectWithdraw(ctx context.Context, id uuid.UUID, reason string) error {
	return s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		var req WithdrawRequest
		if err := db.Where("id = ?", id).First(&req).Error; err != nil {
			return err
		}
		// آزاد کردن frozen
		db.Model(&Wallet{}).Where("id = ?", req.WalletID).
			UpdateColumn("frozen", gorm.Expr("frozen - ?", req.Amount+req.Fee))

		now := time.Now()
		return db.Model(&WithdrawRequest{}).Where("id = ?", id).
			Updates(map[string]any{
				"status":       WithdrawRejected,
				"admin_note":   reason,
				"processed_at": &now,
			}).Error
	})
}

// ── Stats ──────────────────────────────────────────────────

type Stats struct {
	TotalWallets   int64
	TotalDeposits  float64 // TON
	TotalPayments  float64
	PendingWithdraw int64
}

func (s *Store) GetStats(ctx context.Context) (*Stats, error) {
	var stats Stats
	s.db.WithContext(ctx).Model(&Wallet{}).Count(&stats.TotalWallets)

	var deposits, payments struct{ Sum int64 }
	s.db.WithContext(ctx).Model(&Transaction{}).
		Where("type = ? AND status = ?", TxDeposit, TxConfirmed).
		Select("SUM(amount) as sum").Scan(&deposits)
	s.db.WithContext(ctx).Model(&Transaction{}).
		Where("type = ? AND status = ?", TxPayment, TxConfirmed).
		Select("SUM(ABS(amount)) as sum").Scan(&payments)
	s.db.WithContext(ctx).Model(&WithdrawRequest{}).
		Where("status = ?", WithdrawPending).Count(&stats.PendingWithdraw)

	stats.TotalDeposits = float64(deposits.Sum) / 1e9
	stats.TotalPayments = float64(payments.Sum) / 1e9
	return &stats, nil
}

// ── Internal Transfer ──────────────────────────────────────

// Transfer مبلغ را از wallet کاربر A به wallet کاربر B منتقل می‌کند.
// atomic — هر دو آپدیت در یک transaction انجام می‌شود.
func (s *Store) Transfer(ctx context.Context, fromID, toID uuid.UUID, amountNano int64, desc string) (*Transaction, *Transaction, error) {
	var fromTx, toTx *Transaction

	err := s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		// قفل هر دو wallet
		var from, to Wallet
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", fromID).First(&from).Error; err != nil {
			return fmt.Errorf("from wallet: %w", err)
		}
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", toID).First(&to).Error; err != nil {
			return fmt.Errorf("to wallet: %w", err)
		}

		// بررسی موجودی
		if !from.HasEnough(amountNano) {
			return fmt.Errorf("insufficient balance")
		}

		// کسر از فرستنده (اول از credit، بعد TON)
		fromUpdates := map[string]any{}
		var creditUsed, tonUsed int64
		if from.Credit >= amountNano {
			creditUsed = amountNano
		} else {
			creditUsed = from.Credit
			tonUsed = amountNano - creditUsed
		}
		if creditUsed > 0 {
			fromUpdates["credit"] = gorm.Expr("credit - ?", creditUsed)
		}
		if tonUsed > 0 {
			fromUpdates["ton_balance"] = gorm.Expr("ton_balance - ?", tonUsed)
		}
		db.Model(&Wallet{}).Where("id = ?", fromID).Updates(fromUpdates)

		// افزایش به گیرنده (به TON)
		db.Model(&Wallet{}).Where("id = ?", toID).
			UpdateColumn("ton_balance", gorm.Expr("ton_balance + ?", amountNano))

		now := time.Now()
		fromTx = &Transaction{
			WalletID:    fromID,
			Type:        TxPayment,
			Status:      TxConfirmed,
			Amount:      -amountNano,
			Description: "انتقال به کاربر: " + desc,
			ConfirmedAt: &now,
		}
		toTx = &Transaction{
			WalletID:    toID,
			Type:        TxDeposit,
			Status:      TxConfirmed,
			Amount:      amountNano,
			Description: "دریافت از انتقال: " + desc,
			ConfirmedAt: &now,
		}
		if err := db.Create(fromTx).Error; err != nil {
			return err
		}
		return db.Create(toTx).Error
	})
	return fromTx, toTx, err
}
