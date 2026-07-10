package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

const colBroadcastMsgs = "broadcast_msgs"

// AddBroadcastMsg یک پیام همگانی را برای حذف خودکار ثبت می‌کند.
func (s *Store) AddBroadcastMsg(ctx context.Context, code string, chatID int64, msgID int, deleteAt time.Time) {
	m := &models.BroadcastMsg{Code: code, ChatID: chatID, MsgID: msgID, DeleteAt: deleteAt}
	m.ID = newID()
	m.InstanceID = s.instanceID
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	_, err := s.col(colBroadcastMsgs).InsertOne(ctx, m)
	s.logErr("AddBroadcastMsg", err)
}

// DueBroadcastMsgs پیام‌هایی که زمان حذفشان رسیده را برمی‌گرداند.
func (s *Store) DueBroadcastMsgs(ctx context.Context, limit int) ([]models.BroadcastMsg, error) {
	var msgs []models.BroadcastMsg
	filter := s.f(bson.E{Key: "delete_at", Value: bson.D{{Key: "$lte", Value: time.Now()}}})
	err := s.col(colBroadcastMsgs).Find(ctx, filter, &msgs, ports_limit(int64(limit)))
	return msgs, err
}

// BroadcastMsgsByCode پیام‌های یک همگانی خاص را برمی‌گرداند (برای حذف فوری).
func (s *Store) BroadcastMsgsByCode(ctx context.Context, code string, limit int) ([]models.BroadcastMsg, error) {
	var msgs []models.BroadcastMsg
	err := s.col(colBroadcastMsgs).Find(ctx,
		s.f(bson.E{Key: "code", Value: code}), &msgs, ports_limit(int64(limit)))
	return msgs, err
}

// CountBroadcastMsgsByCode تعداد پیام‌های باقی‌مانده‌ی یک همگانی.
func (s *Store) CountBroadcastMsgsByCode(ctx context.Context, code string) int64 {
	n, err := s.col(colBroadcastMsgs).CountDocuments(ctx, s.f(bson.E{Key: "code", Value: code}))
	s.logErr("CountBroadcastMsgsByCode", err)
	return n
}

// DeleteBroadcastMsgRecord رکورد را پس از حذف پیام پاک می‌کند.
func (s *Store) DeleteBroadcastMsgRecord(ctx context.Context, id string) {
	s.logErr("DeleteBroadcastMsgRecord", s.col(colBroadcastMsgs).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: id})))
}

// ── وضعیت کارهای همگانی ───────────────────────────────────────

const colBroadcastJobs = "broadcast_jobs"

// CreateBroadcastJob یک رکورد وضعیت جدید می‌سازد و (id, code) را برمی‌گرداند.
func (s *Store) CreateBroadcastJob(ctx context.Context, code, mode, preview string, total int) string {
	j := &models.BroadcastJob{Code: code, Mode: mode, Preview: preview, Total: total, StartedAt: time.Now()}
	j.ID = newID()
	j.InstanceID = s.instanceID
	j.CreatedAt = time.Now()
	j.UpdatedAt = time.Now()
	_, err := s.col(colBroadcastJobs).InsertOne(ctx, j)
	s.logErr("CreateBroadcastJob", err)
	return j.ID
}

// UpdateBroadcastJob شمارنده‌های یک کار را به‌روز می‌کند.
func (s *Store) UpdateBroadcastJob(ctx context.Context, id string, sent, failed, blocked int, done bool) {
	fields := bson.D{
		{Key: "sent", Value: sent},
		{Key: "failed", Value: failed},
		{Key: "blocked", Value: blocked},
		{Key: "done", Value: done},
	}
	if done {
		fields = append(fields, bson.E{Key: "ended_at", Value: time.Now()})
	}
	s.logErr("UpdateBroadcastJob", s.col(colBroadcastJobs).UpdateOne(ctx, s.f(bson.E{Key: "_id", Value: id}), set(fields)))
}

// SetBroadcastJobCanceled یک کار را به حالت لغو می‌برد.
func (s *Store) SetBroadcastJobCanceled(ctx context.Context, id string) {
	s.logErr("SetBroadcastJobCanceled", s.col(colBroadcastJobs).UpdateOne(ctx, s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{{Key: "canceled", Value: true}})))
}

// GetBroadcastJob یک کار را با شناسه برمی‌گرداند.
func (s *Store) GetBroadcastJob(ctx context.Context, id string) (*models.BroadcastJob, error) {
	var j models.BroadcastJob
	err := s.col(colBroadcastJobs).FindOne(ctx, s.f(bson.E{Key: "_id", Value: id}), &j)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// FindBroadcastJobByCode یک کار را با کد کوتاه برمی‌گرداند.
func (s *Store) FindBroadcastJobByCode(ctx context.Context, code string) (*models.BroadcastJob, error) {
	var j models.BroadcastJob
	err := s.col(colBroadcastJobs).FindOne(ctx, s.f(bson.E{Key: "code", Value: code}), &j)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// ListBroadcastJobs آخرین کارهای همگانی را برمی‌گرداند.
func (s *Store) ListBroadcastJobs(ctx context.Context, limit int) ([]models.BroadcastJob, error) {
	var jobs []models.BroadcastJob
	err := s.col(colBroadcastJobs).Find(ctx, s.f(), &jobs,
		ports_sortDesc("started_at"), ports_limit(int64(limit)))
	return jobs, err
}
