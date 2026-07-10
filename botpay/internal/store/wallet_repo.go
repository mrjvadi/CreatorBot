package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PayHandle کامنت شخصی و یکتای کاربر برای واریز روی ولت مشترک.
// از telegram_id (که یکتا است) مشتق می‌شود تا یکتایی تضمین شود و هیچ‌گاه
// با pay_handle خالی برخورد unique-constraint رخ ندهد.
func PayHandle(telegramID int64) string {
	return "u_" + strconv.FormatInt(telegramID, 36)
}

// GetOrCreateWallet کیف پول کاربر را پیدا یا می‌سازد.
// همه کاربرها یک masterAddr مشترک دارند — فقط telegram_id یکتا است و هر
// کیف پول یک کامنت شخصی یکتا (pay_handle) دریافت می‌کند.
func (s *Store) GetOrCreateWallet(ctx context.Context, telegramID int64, tonAddress string) (*Wallet, error) {
	handle := PayHandle(telegramID)
	var w Wallet
	result := s.db.WithContext(ctx).
		Where(Wallet{TelegramID: telegramID}).
		Attrs(Wallet{TONAddress: tonAddress, IsActive: true, PayHandle: handle}).
		FirstOrCreate(&w)
	if result.Error != nil {
		return &w, result.Error
	}
	// backfill برای کیف‌پول‌های قدیمی که با pay_handle خالی ساخته شده بودند.
	if w.PayHandle == "" {
		if err := s.db.WithContext(ctx).Model(&w).Update("pay_handle", handle).Error; err == nil {
			w.PayHandle = handle
		}
	}
	return &w, nil
}

// GetWalletByPayHandle کیف پول را بر اساس کامنت شخصی پیدا می‌کند.
func (s *Store) GetWalletByPayHandle(ctx context.Context, handle string) (*Wallet, error) {
	var w Wallet
	err := s.db.WithContext(ctx).Where("pay_handle = ?", handle).First(&w).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &w, err
}

// SetWalletLang زبان رابط کاربر را ذخیره می‌کند.
func (s *Store) SetWalletLang(ctx context.Context, telegramID int64, lang string) error {
	return s.db.WithContext(ctx).Model(&Wallet{}).
		Where("telegram_id = ?", telegramID).
		Update("lang", lang).Error
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
// amountNano باید مثبت باشد (chi چک هم در payresponder انجام می‌شود، ولی این
// یک لایه‌ی دوم دفاعی است تا فراخوانی مستقیم آینده هم اشتباهاً مبلغ منفی را
// نپذیرد — منفی یعنی "کسر" در واقع اعتبار اضافه می‌کند، دقیقاً برعکسِ مقصود).
// serviceID+ref با هم idempotency key هستند: اگر تراکنشی از قبل با همین جفت
// ثبت شده باشد (مثلاً به‌خاطر retry سمت کلاینت روی timeout)، دوباره کسر
// نمی‌شود — همان تراکنش قبلی برگردانده می‌شود.
func (s *Store) Deduct(ctx context.Context, walletID uuid.UUID, amountNano int64, serviceID, ref, desc string) (*Transaction, error) {
	if amountNano <= 0 {
		return nil, fmt.Errorf("invalid amount: must be positive")
	}
	var tx *Transaction
	err := s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		if ref != "" {
			var existing Transaction
			err := db.Where("wallet_id = ? AND service_id = ? AND ref = ? AND type = ?",
				walletID, serviceID, ref, TxPayment).First(&existing).Error
			if err == nil {
				tx = &existing // تکراری — idempotent no-op، دوباره کسر نمی‌شود
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

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
			TxHash:      "int-" + uuid.New().String(),
			Description: desc,
			ConfirmedAt: &now,
		}
		return db.Create(tx).Error
	})
	return tx, err
}

// AddCredit افزایش اعتبار داخلی توسط ادمین.
func (s *Store) AddCredit(ctx context.Context, walletID uuid.UUID, amountNano int64, desc string) error {
	if amountNano <= 0 {
		return fmt.Errorf("invalid amount: must be positive")
	}
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
			TxHash:      "int-" + uuid.New().String(),
			Description: desc,
			ConfirmedAt: &now,
		}).Error
	})
}

// ListTransactions تراکنش‌های اخیر یک کیف پول را برمی‌گرداند.
func (s *Store) ListTransactions(ctx context.Context, walletID uuid.UUID, limit int) ([]Transaction, error) {
	var txs []Transaction
	err := s.db.WithContext(ctx).
		Where("wallet_id = ?", walletID).
		Order("created_at DESC").
		Limit(limit).
		Find(&txs).Error
	return txs, err
}
