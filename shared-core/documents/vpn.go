package documents

import (
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VPNSubscription اشتراک VPN — داده عملیاتی.
type VPNSubscription struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	UserID      int64     `bson:"user_id"`      // Telegram ID
	PanelID     string    `bson:"panel_id"`     // UUID از PostgreSQL
	PlanID      string    `bson:"plan_id"`      // UUID از PostgreSQL
	Username    string    `bson:"username"`     // نام کاربری روی پنل
	Status      string    `bson:"status"`       // active, expired, disabled
	ExpiresAt   time.Time `bson:"expires_at"`
	DataLimitGB float64   `bson:"data_limit_gb"`
	UsedDataGB  float64   `bson:"used_data_gb"`
	Links       []string  `bson:"links"`
}

// VPNPaymentReceipt رسید پرداخت کارت به کارت.
type VPNPaymentReceipt struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	ports.DocBase
	UserID      int64     `bson:"user_id"`
	Amount      float64   `bson:"amount"`
	Gateway     string    `bson:"gateway"` // zarinpal, nowpayments, card
	Status      string    `bson:"status"`  // pending, done, failed
	RefCode     string    `bson:"ref_code"`
	ReceiptPhoto string   `bson:"receipt_photo,omitempty"` // file_id برای کارت
	PlanID      string    `bson:"plan_id"`
	PaidAt      *time.Time `bson:"paid_at,omitempty"`
}
