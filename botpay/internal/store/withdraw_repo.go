package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateWithdraw درخواست برداشت را ثبت و موجودی را freeze می‌کند.
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

// CompleteWithdraw برداشت را نهایی می‌کند: کسر از موجودی و آزادسازی frozen.
func (s *Store) CompleteWithdraw(ctx context.Context, id uuid.UUID, txHash string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		var req WithdrawRequest
		if err := db.Where("id = ?", id).First(&req).Error; err != nil {
			return err
		}

		if err := db.Model(&Wallet{}).Where("id = ?", req.WalletID).
			Updates(map[string]any{
				"ton_balance": gorm.Expr("ton_balance - ?", req.Amount+req.Fee),
				"frozen":      gorm.Expr("frozen - ?", req.Amount+req.Fee),
			}).Error; err != nil {
			return err
		}

		return db.Model(&WithdrawRequest{}).Where("id = ?", id).
			Updates(map[string]any{
				"status":       WithdrawDone,
				"tx_hash":      txHash,
				"processed_at": &now,
			}).Error
	})
}

// RejectWithdraw برداشت را رد می‌کند و frozen را آزاد می‌کند.
func (s *Store) RejectWithdraw(ctx context.Context, id uuid.UUID, reason string) error {
	return s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		var req WithdrawRequest
		if err := db.Where("id = ?", id).First(&req).Error; err != nil {
			return err
		}
		// آزاد کردن frozen
		if err := db.Model(&Wallet{}).Where("id = ?", req.WalletID).
			UpdateColumn("frozen", gorm.Expr("frozen - ?", req.Amount+req.Fee)).Error; err != nil {
			return err
		}

		now := time.Now()
		return db.Model(&WithdrawRequest{}).Where("id = ?", id).
			Updates(map[string]any{
				"status":       WithdrawRejected,
				"admin_note":   reason,
				"processed_at": &now,
			}).Error
	})
}
