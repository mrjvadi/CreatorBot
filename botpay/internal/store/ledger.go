// Package store — ledger.go
// پیاده‌سازی Double-Entry Ledger برای botpay.
// قانون طلایی: مجموع همه entry ها همیشه صفر است.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RecordTransfer یک انتقال دوطرفه ثبت می‌کند.
// همیشه به صورت atomic: debit از fromWallet + credit به toWallet.
func (s *Store) RecordTransfer(ctx context.Context,
	fromWalletID, toWalletID uuid.UUID,
	amountNano int64, txID uuid.UUID, ref, note string,
) (*TransferPair, error) {

	if amountNano <= 0 {
		return nil, fmt.Errorf("amount must be positive, got %d", amountNano)
	}
	if fromWalletID == toWalletID {
		return nil, fmt.Errorf("cannot transfer to same wallet")
	}

	var pair TransferPair

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// ── موجودی‌های فعلی ────────────────────────────────
		var fromWallet, toWallet Wallet
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			First(&fromWallet, "id = ?", fromWalletID).Error; err != nil {
			return fmt.Errorf("from wallet: %w", err)
		}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			First(&toWallet, "id = ?", toWalletID).Error; err != nil {
			return fmt.Errorf("to wallet: %w", err)
		}

		// ── بررسی موجودی کافی ─────────────────────────────
		if fromWallet.TONBalance < amountNano {
			return fmt.Errorf("insufficient balance: have %d, need %d",
				fromWallet.TONBalance, amountNano)
		}

		now := time.Now()

		// ── Debit از fromWallet ────────────────────────────
		pair.Debit = LedgerEntry{
			ID:            uuid.New(),
			CreatedAt:     now,
			TransactionID: txID,
			WalletID:      fromWalletID,
			Type:          EntryDebit,
			AmountNano:    amountNano,
			BalanceAfter:  fromWallet.TONBalance - amountNano,
			Ref:           ref,
			Note:          note,
		}

		// ── Credit به toWallet ─────────────────────────────
		pair.Credit = LedgerEntry{
			ID:            uuid.New(),
			CreatedAt:     now,
			TransactionID: txID,
			WalletID:      toWalletID,
			Type:          EntryCredit,
			AmountNano:    amountNano,
			BalanceAfter:  toWallet.TONBalance + amountNano,
			Ref:           ref,
			Note:          note,
		}

		// ── زنجیره‌ی هش: آخرین بلوک را با قفل بگیر ─────────
		prevSeq, prevHash, cerr := s.lastChainEntryTx(tx)
		if cerr != nil {
			return fmt.Errorf("chain tip: %w", cerr)
		}
		// debit به انتهای زنجیره، credit به debit وصل می‌شود
		linkEntry(&pair.Debit, prevSeq, prevHash)
		linkEntry(&pair.Credit, pair.Debit.Seq, pair.Debit.Hash)

		// ── ذخیره هر دو entry ─────────────────────────────
		if err := tx.Create(&pair.Debit).Error; err != nil {
			return fmt.Errorf("create debit: %w", err)
		}
		if err := tx.Create(&pair.Credit).Error; err != nil {
			return fmt.Errorf("create credit: %w", err)
		}

		// ── آپدیت موجودی‌ها ────────────────────────────────
		if err := tx.Model(&Wallet{}).Where("id = ?", fromWalletID).
			Update("ton_balance", fromWallet.TONBalance-amountNano).Error; err != nil {
			return fmt.Errorf("update from balance: %w", err)
		}
		if err := tx.Model(&Wallet{}).Where("id = ?", toWalletID).
			Update("ton_balance", toWallet.TONBalance+amountNano).Error; err != nil {
			return fmt.Errorf("update to balance: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &pair, nil
}

// RecordDeposit واریز از blockchain ثبت می‌کند.
// به wallet پلتفرم credit می‌زند.
func (s *Store) RecordDeposit(ctx context.Context,
	walletID uuid.UUID, amountNano int64,
	txID uuid.UUID, txHash, fromAddr string,
) (*LedgerEntry, error) {

	var entry LedgerEntry

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var w Wallet
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			First(&w, "id = ?", walletID).Error; err != nil {
			return err
		}

		entry = LedgerEntry{
			ID:            uuid.New(),
			CreatedAt:     time.Now(),
			TransactionID: txID,
			WalletID:      walletID,
			Type:          EntryCredit,
			AmountNano:    amountNano,
			BalanceAfter:  w.TONBalance + amountNano,
			Ref:           txHash,
			Note:          "deposit from " + fromAddr,
		}

		// زنجیره‌ی هش
		prevSeq, prevHash, cerr := s.lastChainEntryTx(tx)
		if cerr != nil {
			return fmt.Errorf("chain tip: %w", cerr)
		}
		linkEntry(&entry, prevSeq, prevHash)

		if err := tx.Create(&entry).Error; err != nil {
			return err
		}

		return tx.Model(&Wallet{}).Where("id = ?", walletID).
			Update("ton_balance", w.TONBalance+amountNano).Error
	})

	return &entry, err
}

// GetLedgerEntries تاریخچه entries یک wallet.
func (s *Store) GetLedgerEntries(ctx context.Context,
	walletID uuid.UUID, limit int,
) ([]LedgerEntry, error) {
	var entries []LedgerEntry
	err := s.db.WithContext(ctx).
		Where("wallet_id = ?", walletID).
		Order("created_at DESC").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}

// VerifyLedgerBalance یکپارچگی ledger را بررسی می‌کند.
// مجموع credit - debit باید برابر موجودی فعلی باشد.
func (s *Store) VerifyLedgerBalance(ctx context.Context, walletID uuid.UUID) (bool, error) {
	var wallet Wallet
	if err := s.db.WithContext(ctx).
		First(&wallet, "id = ?", walletID).Error; err != nil {
		return false, err
	}

	var creditSum, debitSum int64
	s.db.WithContext(ctx).Model(&LedgerEntry{}).
		Where("wallet_id = ? AND type = ?", walletID, EntryCredit).
		Select("COALESCE(SUM(amount_nano), 0)").Scan(&creditSum)
	s.db.WithContext(ctx).Model(&LedgerEntry{}).
		Where("wallet_id = ? AND type = ?", walletID, EntryDebit).
		Select("COALESCE(SUM(amount_nano), 0)").Scan(&debitSum)

	calculated := creditSum - debitSum
	return calculated == wallet.TONBalance, nil
}
