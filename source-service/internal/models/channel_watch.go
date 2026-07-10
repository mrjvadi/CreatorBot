package models

// ChannelWatch is a persisted "if SourceChannel posts, forward it to
// DestChannel" rule (real-time watch), scoped to the Telegram account
// (Phone) that owns it — a shared Postgres can hold rules for several
// worker accounts at once. Deactivating a watch sets Active to false rather
// than deleting the row, so history/audit isn't lost.
type ChannelWatch struct {
	Base
	Phone         string `gorm:"index;not null"`
	SourceChannel string `gorm:"not null"`
	DestChannel   string `gorm:"not null"`
	Active        bool   `gorm:"not null;default:true"`
}
