package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
			"status":  InvoicePaid,
			"tx_hash": txHash,
			"paid_at": &now,
		}).Error
}

func (s *Store) ExpireOldInvoices(ctx context.Context) error {
	return s.db.WithContext(ctx).Model(&Invoice{}).
		Where("status = 'pending' AND expires_at < ?", time.Now()).
		Update("status", InvoiceExpired).Error
}
