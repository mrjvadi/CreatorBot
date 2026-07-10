package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ── Channel ──────────────────────────────────────────────────────

func (s *Store) CreateChannel(ctx context.Context, ch *models.Channel) error {
	ch.ID = newID()
	ch.InstanceID = s.instanceID
	ch.CreatedAt = time.Now()
	ch.UpdatedAt = time.Now()
	ch.Status = models.ChannelPending
	_, err := s.col(colChannels).InsertOne(ctx, ch)
	return err
}

func (s *Store) FindChannel(ctx context.Context, id string) (*models.Channel, error) {
	var ch models.Channel
	err := s.col(colChannels).FindOne(ctx, s.f(bson.E{Key: "_id", Value: id}), &ch)
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (s *Store) FindChannelByTelegramID(ctx context.Context, telegramID int64) (*models.Channel, error) {
	var ch models.Channel
	err := s.col(colChannels).FindOne(ctx, s.f(bson.E{Key: "telegram_id", Value: telegramID}), &ch)
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (s *Store) ListChannels(ctx context.Context, status models.ChannelStatus, pg, ps int) ([]models.Channel, error) {
	filter := s.f()
	if status != "" {
		filter = s.f(bson.E{Key: "status", Value: status})
	}
	var list []models.Channel
	err := s.col(colChannels).Find(ctx, filter, &list,
		sortDesc("created_at"),
		skip(int64((pg-1)*ps)),
		limit(int64(ps)),
	)
	return list, err
}

// UpdateChannel فیلدهای دلخواه یک کانال را به‌روزرسانی می‌کند.
func (s *Store) UpdateChannel(ctx context.Context, id string, fields bson.D) error {
	return s.col(colChannels).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(fields),
	)
}

func (s *Store) UpdateChannelStatus(ctx context.Context, id string, status models.ChannelStatus) error {
	return s.col(colChannels).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{{Key: "status", Value: status}}),
	)
}

func (s *Store) UpdateChannelStats(ctx context.Context, id string, memberCount, avgViews int, engageRate float64) error {
	return s.col(colChannels).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{
			{Key: "member_count", Value: memberCount},
			{Key: "avg_views", Value: avgViews},
			{Key: "engage_rate", Value: engageRate},
		}),
	)
}

func (s *Store) DeleteChannel(ctx context.Context, id string) error {
	now := time.Now()
	return s.col(colChannels).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		set(bson.D{{Key: "deleted_at", Value: now}}),
	)
}

// ── Tag ──────────────────────────────────────────────────────────

func (s *Store) CreateTag(ctx context.Context, tag *models.Tag) error {
	tag.ID = newID()
	tag.InstanceID = s.instanceID
	tag.CreatedAt = time.Now()
	tag.IsActive = true
	_, err := s.col(colTags).InsertOne(ctx, tag)
	return err
}

func (s *Store) ListTags(ctx context.Context) ([]models.Tag, error) {
	var list []models.Tag
	err := s.col(colTags).Find(ctx,
		s.f(bson.E{Key: "is_active", Value: true}),
		&list,
		sortAsc("name"),
	)
	return list, err
}

func (s *Store) SetTagInactive(ctx context.Context, id string) error {
	return s.col(colTags).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: id}),
		bson.D{{Key: "$set", Value: bson.D{{Key: "is_active", Value: false}}}},
	)
}

func (s *Store) FindTagBySlug(ctx context.Context, slug string) (*models.Tag, error) {
	var tag models.Tag
	err := s.col(colTags).FindOne(ctx, s.f(bson.E{Key: "slug", Value: slug}), &tag)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (s *Store) ListChannelsByTag(ctx context.Context, tagID string, minMembers int) ([]models.Channel, error) {
	filter := bson.D{
		{Key: "instance_id", Value: s.instanceID},
		{Key: "tag_ids", Value: bson.D{{Key: "$in", Value: []string{tagID}}}},
		{Key: "status", Value: models.ChannelActive},
		{Key: "deleted_at", Value: bson.D{{Key: "$exists", Value: false}}},
	}
	if minMembers > 0 {
		filter = append(filter, bson.E{Key: "member_count", Value: bson.D{{Key: "$gte", Value: minMembers}}})
	}
	var list []models.Channel
	err := s.col(colChannels).Find(ctx, filter, &list,
		ports.WithSort(bson.D{{Key: "member_count", Value: -1}}),
	)
	return list, err
}
