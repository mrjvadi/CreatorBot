package models

import "time"

// ── Subscription Plan ─────────────────────────────────────────

type SubPlan struct {
	Base      `bson:",inline"`
	Name      string  `bson:"name"`
	Price     float64 `bson:"price"` // تومان یا TON
	Days      int     `bson:"days"`
	IsActive  bool    `bson:"is_active"`
	SortOrder int     `bson:"sort_order"`
}

// ── Payment ───────────────────────────────────────────────────

type PaymentStatus string
type PaymentGateway string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentConfirmed PaymentStatus = "confirmed"
	PaymentFailed    PaymentStatus = "failed"

	GatewayZarinpal PaymentGateway = "zarinpal"
	GatewayZibal    PaymentGateway = "zibal"
	GatewayCard     PaymentGateway = "card"
	GatewayTON      PaymentGateway = "ton"
	GatewayTRON     PaymentGateway = "tron"
	GatewayStars    PaymentGateway = "stars"
)

type Payment struct {
	Base        `bson:",inline"`
	UserID      string         `bson:"user_id"`
	TelegramID  int64          `bson:"telegram_id"`
	PlanID      string         `bson:"plan_id"`
	Gateway     PaymentGateway `bson:"gateway"`
	Amount      float64        `bson:"amount"`
	Status      PaymentStatus  `bson:"status"`
	Authority   string         `bson:"authority"`
	TxHash      string         `bson:"tx_hash"`
	CardRef     string         `bson:"card_ref"`
	Stars       int            `bson:"stars"`
	ConfirmedAt *time.Time     `bson:"confirmed_at,omitempty"`
}
