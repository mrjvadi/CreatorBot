package store

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

func (s *Store) CreateJob(ctx context.Context, job *models.ScheduledJob) error {
	job.ID = newID()
	job.InstanceID = s.instanceID
	job.CreatedAt = time.Now()
	job.Status = models.JobPending
	job.MaxAttempts = 3
	_, err := s.col(colJobs).InsertOne(ctx, job)
	return err
}

// FetchDueJobs بازیابی jobهایی که زمان اجرایشان رسیده.
func (s *Store) FetchDueJobs(ctx context.Context) ([]models.ScheduledJob, error) {
	filter := bson.D{
		{Key: "instance_id", Value: s.instanceID},
		{Key: "status", Value: models.JobPending},
		{Key: "run_at", Value: bson.D{{Key: "$lte", Value: time.Now()}}},
	}
	var list []models.ScheduledJob
	err := s.col(colJobs).Find(ctx, filter, &list,
		sortAsc("run_at"),
		limit(50),
	)
	return list, err
}

func (s *Store) MarkJobRunning(ctx context.Context, id string) error {
	now := time.Now()
	return s.col(colJobs).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "status", Value: models.JobRunning},
				{Key: "started_at", Value: now},
			}},
			{Key: "$inc", Value: bson.D{{Key: "attempts", Value: 1}}},
		},
	)
}

func (s *Store) MarkJobDone(ctx context.Context, id string) error {
	now := time.Now()
	return s.col(colJobs).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{
			{Key: "status", Value: models.JobDone},
			{Key: "done_at", Value: now},
		}),
	)
}

func (s *Store) MarkJobFailed(ctx context.Context, id, errMsg string) error {
	return s.col(colJobs).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{
			{Key: "status", Value: models.JobFailed},
			{Key: "last_error", Value: errMsg},
		}),
	)
}

func (s *Store) RetryJob(ctx context.Context, id string, nextRun time.Time) error {
	return s.col(colJobs).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{
			{Key: "status", Value: models.JobPending},
			{Key: "run_at", Value: nextRun},
		}),
	)
}

func (s *Store) CancelJob(ctx context.Context, id string) error {
	return s.col(colJobs).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{{Key: "status", Value: models.JobCancelled}}),
	)
}

// CancelCampaignJobs لغو تمام jobهای باز یک کمپین.
// چون Collection.UpdateMany در interface نیست، با حلقه انجام می‌شود.
func (s *Store) CancelCampaignJobs(ctx context.Context, campaignID string) error {
	var jobs []models.ScheduledJob
	err := s.col(colJobs).Find(ctx,
		s.f(
			bson.E{Key: "campaign_id", Value: campaignID},
			bson.E{Key: "status", Value: models.JobPending},
		),
		&jobs,
	)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if e := s.CancelJob(ctx, j.ID); e != nil {
			return e
		}
	}
	return nil
}

// CancelReservationJobs لغو تمام jobهای «باز»ِ متعلق به یک رزروی مشخص —
// چه job ارسال ریپلیِ بعدی (Payload = "resID|idx") چه job حذف پایانی
// (Payload = resID). برای وقتی لازم است که یک چرخه زودتر از موعدِ
// طبیعی‌اش تمام شود (مثلاً پایان بازه‌ی روزانه، یا رسیدن سقفِ کلِ چرخه)
// و نباید jobهای قدیمیِ همان رزرو بعداً دوباره اجرا شوند.
func (s *Store) CancelReservationJobs(ctx context.Context, reservationID string) error {
	var jobs []models.ScheduledJob
	err := s.col(colJobs).Find(ctx,
		s.f(bson.E{Key: "status", Value: models.JobPending}),
		&jobs,
	)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if j.Payload == reservationID || strings.HasPrefix(j.Payload, reservationID+"|") {
			if e := s.CancelJob(ctx, j.ID); e != nil {
				return e
			}
		}
	}
	return nil
}
