package models

import "time"

// BroadcastMsg رکورد یک پیام همگانیِ ارسال‌شده برای حذف خودکار بعد از زمان مشخص.
type BroadcastMsg struct {
	Base     `bson:",inline"`
	Code     string    `bson:"code"` // کد همگانیِ مربوطه
	ChatID   int64     `bson:"chat_id"`
	MsgID    int       `bson:"msg_id"`
	DeleteAt time.Time `bson:"delete_at"`
}

// BroadcastJob وضعیت یک کار ارسال همگانی برای پیگیری زنده.
type BroadcastJob struct {
	Base      `bson:",inline"`
	Code      string     `bson:"code"` // کد کوتاه برای پیگیری/کنترل
	Mode      string     `bson:"mode"` // copy | forward
	Preview   string     `bson:"preview"`
	Total     int        `bson:"total"`
	Sent      int        `bson:"sent"`
	Failed    int        `bson:"failed"`
	Blocked   int        `bson:"blocked"`
	Done      bool       `bson:"done"`
	Canceled  bool       `bson:"canceled"`
	StartedAt time.Time  `bson:"started_at"`
	EndedAt   *time.Time `bson:"ended_at,omitempty"`
}

// Remaining تعداد باقی‌مانده.
func (b *BroadcastJob) Remaining() int {
	r := b.Total - b.Sent - b.Failed - b.Blocked
	if r < 0 {
		return 0
	}
	return r
}
