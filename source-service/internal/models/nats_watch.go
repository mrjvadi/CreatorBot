package models

// NatsWatch is a persisted "if a message arrives on Subject, send it to
// DestChannel" rule — the NATS-triggered equivalent of ChannelWatch, for
// bridging external NATS events into Telegram.
type NatsWatch struct {
	Base
	Phone       string `gorm:"index;not null"`
	Subject     string `gorm:"not null"`
	DestChannel string `gorm:"not null"`
	Active      bool   `gorm:"not null;default:true"`
}
