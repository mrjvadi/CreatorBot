package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// ── Force Join Channels ───────────────────────────────────────

func (s *Store) AddForceJoinChannel(ctx context.Context, ch *models.ForceJoinChannel) error {
	if ch.ID == "" {
		ch.ID = newID()
	}
	ch.InstanceID = s.instanceID
	ch.CreatedAt = time.Now()
	ch.UpdatedAt = time.Now()
	_, err := s.col(colForceJoin).InsertOne(ctx, ch)
	return err
}

func (s *Store) ListForceJoinChannels(ctx context.Context) ([]models.ForceJoinChannel, error) {
	var chs []models.ForceJoinChannel
	err := s.col(colForceJoin).Find(ctx,
		s.f(bson.E{Key: "is_active", Value: true}), &chs,
		ports_sortAsc("sort_order"))
	return chs, err
}

func (s *Store) RemoveForceJoinChannel(ctx context.Context, id string) error {
	return s.col(colForceJoin).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: id}))
}

func (s *Store) FindForceJoinChannel(ctx context.Context, id string) (*models.ForceJoinChannel, error) {
	var ch models.ForceJoinChannel
	err := s.col(colForceJoin).FindOne(ctx, s.f(bson.E{Key: "_id", Value: id}), &ch)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// FindForceJoinByChat یک قفل را بر اساس chat_id برمی‌گرداند (برای رویداد عضویت).
func (s *Store) FindForceJoinByChat(ctx context.Context, chatID int64) (*models.ForceJoinChannel, error) {
	var ch models.ForceJoinChannel
	err := s.col(colForceJoin).FindOne(ctx, s.f(bson.E{Key: "chat_id", Value: chatID}), &ch)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (s *Store) UpdateForceJoinChannel(ctx context.Context, ch *models.ForceJoinChannel) error {
	return s.col(colForceJoin).UpdateOne(ctx, s.f(bson.E{Key: "_id", Value: ch.ID}),
		set(bson.D{
			{Key: "kind", Value: ch.Kind},
			{Key: "mode", Value: ch.Mode},
			{Key: "chat_id", Value: ch.ChatID},
			{Key: "title", Value: ch.Title},
			{Key: "username", Value: ch.Username},
			{Key: "invite_url", Value: ch.InviteURL},
			{Key: "bot_username", Value: ch.BotUsername},
			{Key: "bot_token", Value: ch.BotToken},
			{Key: "member_cap", Value: ch.MemberCap},
			{Key: "joined_count", Value: ch.JoinedCount},
			{Key: "check_bot", Value: ch.CheckBot},
			{Key: "is_active", Value: ch.IsActive},
			{Key: "sort_order", Value: ch.SortOrder},
		}))
}

// IncrLockJoined شمارش عضویت یک قفل را یک واحد زیاد و در صورت رسیدن به حد،
// قفل را غیرفعال می‌کند. خروجی: آیا به حد رسید و غیرفعال شد.
func (s *Store) IncrLockJoined(ctx context.Context, id string) bool {
	ch, err := s.FindForceJoinChannel(ctx, id)
	if err != nil {
		s.logErr("IncrLockJoined: find", err)
		return false
	}
	if ch == nil {
		return false
	}
	ch.JoinedCount++
	deactivated := false
	if ch.MemberCap > 0 && ch.JoinedCount >= ch.MemberCap {
		ch.IsActive = false
		deactivated = true
	}
	s.logErr("IncrLockJoined: update", s.UpdateForceJoinChannel(ctx, ch))
	return deactivated
}

// ── Preview Channels ──────────────────────────────────────────

func (s *Store) AddPreviewChannel(ctx context.Context, ch *models.PreviewChannel) error {
	if ch.ID == "" {
		ch.ID = newID()
	}
	ch.InstanceID = s.instanceID
	ch.CreatedAt = time.Now()
	ch.UpdatedAt = time.Now()
	_, err := s.col(colPreview).InsertOne(ctx, ch)
	return err
}

func (s *Store) ListPreviewChannels(ctx context.Context) ([]models.PreviewChannel, error) {
	var chs []models.PreviewChannel
	err := s.col(colPreview).Find(ctx,
		s.f(bson.E{Key: "is_active", Value: true}), &chs)
	return chs, err
}

func (s *Store) RemovePreviewChannel(ctx context.Context, id string) error {
	return s.col(colPreview).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: id}))
}
