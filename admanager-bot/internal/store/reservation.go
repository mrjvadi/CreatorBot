package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

func (s *Store) CreateReservation(ctx context.Context, r *models.Reservation) error {
	r.ID = newID()
	r.InstanceID = s.instanceID
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	r.Status = models.ReservationPending
	_, err := s.col(colReservations).InsertOne(ctx, r)
	return err
}

func (s *Store) FindReservation(ctx context.Context, id string) (*models.Reservation, error) {
	var r models.Reservation
	err := s.col(colReservations).FindOne(ctx, s.f(bson.E{Key: "_id", Value: id}), &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) ListReservationsByChannel(ctx context.Context, channelID string, from, to time.Time) ([]models.Reservation, error) {
	filter := bson.D{
		{Key: "instance_id", Value: s.instanceID},
		{Key: "channel_id", Value: channelID},
		{Key: "scheduled_at", Value: bson.D{
			{Key: "$gte", Value: from},
			{Key: "$lt", Value: to},
		}},
		{Key: "status", Value: bson.D{{Key: "$in", Value: []models.ReservationStatus{
			models.ReservationPending,
			models.ReservationConfirmed,
		}}}},
	}
	var list []models.Reservation
	err := s.col(colReservations).Find(ctx, filter, &list, sortAsc("scheduled_at"))
	return list, err
}

// HasRecentReservation بررسی می‌کند آیا برای این کمپین/کانال رزروی پس از
// زمان since ساخته شده (برای فاصله‌گذاری بین ارسال‌ها).
func (s *Store) HasRecentReservation(ctx context.Context, campaignID, channelID string, since time.Time) (bool, error) {
	filter := s.f(
		bson.E{Key: "campaign_id", Value: campaignID},
		bson.E{Key: "channel_id", Value: channelID},
		bson.E{Key: "scheduled_at", Value: bson.D{{Key: "$gte", Value: since}}},
	)
	var list []models.Reservation
	if err := s.col(colReservations).Find(ctx, filter, &list, limit(1)); err != nil {
		return false, err
	}
	return len(list) > 0, nil
}

// CountReservations تعداد رزروهای یک کمپین در یک کانال با وضعیت‌های مشخص.
func (s *Store) CountReservations(ctx context.Context, campaignID, channelID string, statuses ...models.ReservationStatus) (int, error) {
	filter := s.f(
		bson.E{Key: "campaign_id", Value: campaignID},
		bson.E{Key: "channel_id", Value: channelID},
		bson.E{Key: "status", Value: bson.D{{Key: "$in", Value: statuses}}},
	)
	var list []models.Reservation
	if err := s.col(colReservations).Find(ctx, filter, &list); err != nil {
		return 0, err
	}
	return len(list), nil
}

// SetReservationMainPosted بعد از ارسال موفق پست اصلی صدا زده می‌شود:
// رزرو را sent علامت می‌زند، پست اصلی را تنها پیام زنده‌ی فعلی می‌کند و به
// تاریخچه اضافه می‌کند. CurrentReplyIndex روی -1 (هنوز ریپلی‌ای نیامده) است.
func (s *Store) SetReservationMainPosted(ctx context.Context, id string, mainMessageID int) error {
	now := time.Now()
	return s.col(colReservations).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{
			{Key: "status", Value: models.ReservationSent},
			{Key: "sent_at", Value: now},
			{Key: "live_message_ids", Value: []int{mainMessageID}},
			{Key: "current_reply_index", Value: -1},
			{Key: "posted_message_ids", Value: []int{mainMessageID}},
		}),
	)
}

// SetReservationReplyPosted بعد از ارسال یک ریپلیِ جدید (که جای ریپلیِ قبلی
// را می‌گیرد) صدا زده می‌شود: پست اصلی در LiveMessageIDs نگه داشته می‌شود،
// ریپلیِ تازه جایگزین ریپلیِ قبلی در همان لیست می‌شود، و به تاریخچه اضافه
// می‌شود.
func (s *Store) SetReservationReplyPosted(ctx context.Context, id string, replyMessageID, replyIndex int) error {
	res, err := s.FindReservation(ctx, id)
	if err != nil || res == nil {
		return err
	}
	main := 0
	if len(res.LiveMessageIDs) > 0 {
		main = res.LiveMessageIDs[0]
	}
	live := []int{}
	if main != 0 {
		live = append(live, main)
	}
	live = append(live, replyMessageID)
	history := append(res.PostedMessageIDs, replyMessageID)
	return s.col(colReservations).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{
			{Key: "live_message_ids", Value: live},
			{Key: "current_reply_index", Value: replyIndex},
			{Key: "posted_message_ids", Value: history},
		}),
	)
}

// SetReservationDeleteAt زمان حذف خودکارِ پایانِ چرخه را ذخیره می‌کند.
func (s *Store) SetReservationDeleteAt(ctx context.Context, id string, deleteAt time.Time) error {
	return s.col(colReservations).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{{Key: "delete_at", Value: deleteAt}}),
	)
}

// MarkReservationExpired یعنی پیام‌های زنده‌ی این رزرو پاک شده‌اند (چه در
// پایان طبیعیِ چرخه، چه به‌خاطر قطعِ زودهنگام در پایان بازه‌ی روزانه).
func (s *Store) MarkReservationExpired(ctx context.Context, id string) error {
	return s.col(colReservations).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{
			{Key: "status", Value: models.ReservationExpired},
			{Key: "live_message_ids", Value: []int{}},
		}),
	)
}

// ListLiveReservationsByCampaign رزروهای این کمپین را که هنوز پیام(های)
// زنده در کانال دارند (status=sent) برمی‌گرداند — برای قطعِ فوریِ چرخه‌ها
// هنگام پایان بازه‌ی روزانه.
func (s *Store) ListLiveReservationsByCampaign(ctx context.Context, campaignID string) ([]models.Reservation, error) {
	var list []models.Reservation
	err := s.col(colReservations).Find(ctx,
		s.f(
			bson.E{Key: "campaign_id", Value: campaignID},
			bson.E{Key: "status", Value: models.ReservationSent},
		),
		&list,
	)
	return list, err
}

// ListLiveReservationsByChannel رزروهای دارای پیام(های) زنده (status=sent)
// در یک کانال مشخص را برمی‌گرداند — برای بررسیِ «آیا تبلیغِ ثابتی هست که
// دیگر آخرین پیام کانال نیست» بعد از هر پستِ جدید.
func (s *Store) ListLiveReservationsByChannel(ctx context.Context, channelID string) ([]models.Reservation, error) {
	var list []models.Reservation
	err := s.col(colReservations).Find(ctx,
		s.f(
			bson.E{Key: "channel_id", Value: channelID},
			bson.E{Key: "status", Value: models.ReservationSent},
		),
		&list,
	)
	return list, err
}

// SetReservationLiveMessage بعد از بازارسالِ یک تبلیغِ «ثابت» (چون دیگر
// آخرین پیام کانال نبوده) صدا زده می‌شود: پیام(های) زنده‌ی جدید را جایگزین
// می‌کند و به تاریخچه اضافه می‌کند.
func (s *Store) SetReservationLiveMessage(ctx context.Context, id string, liveIDs []int) error {
	res, err := s.FindReservation(ctx, id)
	if err != nil || res == nil {
		return err
	}
	history := append(res.PostedMessageIDs, liveIDs...)
	return s.col(colReservations).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{
			{Key: "live_message_ids", Value: liveIDs},
			{Key: "posted_message_ids", Value: history},
		}),
	)
}

// LastReservationAt زمان آخرین رزرو این کمپین در یک کانال را برمی‌گرداند.
func (s *Store) LastReservationAt(ctx context.Context, campaignID, channelID string) (time.Time, bool) {
	var list []models.Reservation
	err := s.col(colReservations).Find(ctx,
		s.f(bson.E{Key: "campaign_id", Value: campaignID},
			bson.E{Key: "channel_id", Value: channelID}),
		&list,
		sortDesc("scheduled_at"),
		limit(1),
	)
	if err != nil || len(list) == 0 {
		return time.Time{}, false
	}
	return list[0].ScheduledAt, true
}

func (s *Store) MarkReservationFailed(ctx context.Context, id, errMsg string) error {
	return s.col(colReservations).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{
			{Key: "status", Value: models.ReservationFailed},
			{Key: "error", Value: errMsg},
		}),
	)
}

// CancelCampaignReservations لغو تمام رزروهای باز یک کمپین.
// چون Collection.UpdateMany در interface نیست، با حلقه انجام می‌شود.
func (s *Store) CancelCampaignReservations(ctx context.Context, campaignID string) error {
	var reservations []models.Reservation
	err := s.col(colReservations).Find(ctx,
		s.f(
			bson.E{Key: "campaign_id", Value: campaignID},
			bson.E{Key: "status", Value: bson.D{{Key: "$in", Value: []models.ReservationStatus{
				models.ReservationPending,
				models.ReservationConfirmed,
			}}}},
		),
		&reservations,
	)
	if err != nil {
		return err
	}
	for _, r := range reservations {
		if e := s.col(colReservations).UpdateOne(ctx,
			s.f(bson.E{Key: "_id", Value: r.ID}),
			set(bson.D{{Key: "status", Value: models.ReservationCancelled}}),
		); e != nil {
			return e
		}
	}
	return nil
}
