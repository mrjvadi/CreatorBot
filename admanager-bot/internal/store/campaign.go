package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

func (s *Store) CreateCampaign(ctx context.Context, c *models.Campaign) error {
	c.ID = newID()
	c.InstanceID = s.instanceID
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	c.Status = models.CampaignDraft
	_, err := s.col(colCampaigns).InsertOne(ctx, c)
	return err
}

func (s *Store) FindCampaign(ctx context.Context, id string) (*models.Campaign, error) {
	var c models.Campaign
	err := s.col(colCampaigns).FindOne(ctx,
		s.f(bson.E{Key: "_id", Value: id},
			bson.E{Key: "deleted_at", Value: bson.D{{Key: "$exists", Value: false}}}),
		&c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) ListCampaigns(ctx context.Context, status models.CampaignStatus, pg, ps int) ([]models.Campaign, error) {
	filter := bson.D{
		{Key: "instance_id", Value: s.instanceID},
		{Key: "deleted_at", Value: bson.D{{Key: "$exists", Value: false}}},
	}
	if status != "" {
		filter = append(filter, bson.E{Key: "status", Value: status})
	}
	var list []models.Campaign
	err := s.col(colCampaigns).Find(ctx, filter, &list,
		sortDesc("created_at"),
		skip(int64((pg-1)*ps)),
		limit(int64(ps)),
	)
	return list, err
}

func (s *Store) ListActiveCampaigns(ctx context.Context) ([]models.Campaign, error) {
	now := time.Now()
	filter := bson.D{
		{Key: "instance_id", Value: s.instanceID},
		{Key: "status", Value: models.CampaignRunning},
		{Key: "deleted_at", Value: bson.D{{Key: "$exists", Value: false}}},
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "end_at", Value: bson.D{{Key: "$exists", Value: false}}}},
			bson.D{{Key: "end_at", Value: bson.D{{Key: "$gt", Value: now}}}},
		}},
	}
	var list []models.Campaign
	err := s.col(colCampaigns).Find(ctx, filter, &list)
	return list, err
}

// UpdateCampaign فیلدهای دلخواه یک کمپین را به‌روزرسانی می‌کند.
func (s *Store) UpdateCampaign(ctx context.Context, id string, fields bson.D) error {
	return s.col(colCampaigns).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(fields),
	)
}

func (s *Store) UpdateCampaignStatus(ctx context.Context, id string, status models.CampaignStatus) error {
	return s.col(colCampaigns).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{{Key: "status", Value: status}}),
	)
}

// IncrCampaignImpressions تعداد نمایش‌های یک کمپین را پس از هر ارسال موفق
// یک واحد افزایش می‌دهد.
func (s *Store) IncrCampaignImpressions(ctx context.Context, id string) error {
	return s.col(colCampaigns).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		bson.D{
			{Key: "$inc", Value: bson.D{{Key: "total_impressions", Value: 1}}},
			{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now()}}},
		},
	)
}

func (s *Store) DeleteCampaign(ctx context.Context, id string) error {
	now := time.Now()
	return s.col(colCampaigns).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{{Key: "deleted_at", Value: now}}),
	)
}

// ── Advertisement ────────────────────────────────────────────────

func (s *Store) CreateAd(ctx context.Context, ad *models.Advertisement) error {
	ad.ID = newID()
	ad.InstanceID = s.instanceID
	ad.CreatedAt = time.Now()
	ad.UpdatedAt = time.Now()
	ad.IsActive = true
	if ad.Replies == nil {
		ad.Replies = []models.AdReply{} // آرایه‌ی خالی تا آپدیت بعدی روی null نخورد
	}
	_, err := s.col(colAds).InsertOne(ctx, ad)
	return err
}

// UpdateAd فیلدهای دلخواه یک تبلیغ را به‌روزرسانی می‌کند (نام، تنظیمات و…).
func (s *Store) UpdateAd(ctx context.Context, id string, fields bson.D) error {
	return s.col(colAds).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(fields),
	)
}

// ReplaceAdMain پست اصلی تبلیغ را با پیام جدید جایگزین می‌کند.
func (s *Store) ReplaceAdMain(ctx context.Context, id string, mainMessageID int) error {
	return s.UpdateAd(ctx, id, bson.D{{Key: "main_message_id", Value: mainMessageID}})
}

// ReplaceAdReplies کل لیست ریپلی‌های یک تبلیغ را جایگزین می‌کند (برای
// افزودن/حذف/تغییرِ ترتیب/ویرایش مدت‌زمانِ هرکدام).
func (s *Store) ReplaceAdReplies(ctx context.Context, id string, replies []models.AdReply) error {
	if replies == nil {
		replies = []models.AdReply{}
	}
	return s.UpdateAd(ctx, id, bson.D{{Key: "replies", Value: replies}})
}

func (s *Store) FindAd(ctx context.Context, id string) (*models.Advertisement, error) {
	var ad models.Advertisement
	err := s.col(colAds).FindOne(ctx, s.f(bson.E{Key: "_id", Value: id}), &ad)
	if err != nil {
		return nil, err
	}
	return &ad, nil
}

func (s *Store) ListAdsByCampaign(ctx context.Context, campaignID string) ([]models.Advertisement, error) {
	var list []models.Advertisement
	err := s.col(colAds).Find(ctx,
		s.f(bson.E{Key: "campaign_id", Value: campaignID},
			bson.E{Key: "is_active", Value: true}),
		&list,
	)
	return list, err
}

// AppendAdReply یک پیام ریپلی (به‌همراه مدت‌زمان نمایش خودش، به دقیقه) به
// انتهای لیست ریپلی‌های تبلیغ اضافه می‌کند. به‌جای $push کل آرایه را $set
// می‌کنیم تا روی اسناد قدیمیِ دارای مقدار null هم بدون خطا کار کند.
func (s *Store) AppendAdReply(ctx context.Context, id string, messageID, durationMinutes int) error {
	ad, err := s.FindAd(ctx, id)
	if err != nil || ad == nil {
		return err
	}
	next := append(ad.Replies, models.AdReply{MessageID: messageID, DurationMinutes: durationMinutes})
	return s.col(colAds).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{{Key: "replies", Value: next}}),
	)
}

func (s *Store) DeleteAd(ctx context.Context, id string) error {
	return s.col(colAds).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{{Key: "is_active", Value: false}}),
	)
}
