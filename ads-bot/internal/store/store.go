package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) *Store { return &Store{db: db} }

// ── Publisher ──────────────────────────────────────────────

func (s *Store) UpsertPublisher(ctx context.Context, telegramID int64, username, firstName string) (*Publisher, error) {
	var p Publisher
	err := s.db.WithContext(ctx).
		Where(Publisher{TelegramID: telegramID}).
		Assign(Publisher{Username: username, FirstName: firstName}).
		FirstOrCreate(&p).Error
	return &p, err
}

func (s *Store) FindPublisher(ctx context.Context, telegramID int64) (*Publisher, error) {
	var p Publisher
	err := s.db.WithContext(ctx).Where("telegram_id = ?", telegramID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (s *Store) UpdateBalance(ctx context.Context, publisherID uuid.UUID, delta float64) error {
	return s.db.WithContext(ctx).Model(&Publisher{}).
		Where("id = ?", publisherID).
		UpdateColumn("balance", gorm.Expr("balance + ?", delta)).Error
}

func (s *Store) ListPublishers(ctx context.Context) ([]Publisher, error) {
	var list []Publisher
	return list, s.db.WithContext(ctx).Order("created_at DESC").Find(&list).Error
}

// ── Campaign ───────────────────────────────────────────────

func (s *Store) CreateCampaign(ctx context.Context, c *Campaign) error {
	return s.db.WithContext(ctx).Create(c).Error
}

func (s *Store) FindCampaign(ctx context.Context, id uuid.UUID) (*Campaign, error) {
	var c Campaign
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (s *Store) UpdateCampaign(ctx context.Context, c *Campaign) error {
	return s.db.WithContext(ctx).Save(c).Error
}

func (s *Store) FindCampaignsByPublisher(ctx context.Context, publisherID uuid.UUID) ([]Campaign, error) {
	var list []Campaign
	return list, s.db.WithContext(ctx).
		Where("publisher_id = ?", publisherID).
		Order("created_at DESC").Find(&list).Error
}

func (s *Store) FindPendingCampaigns(ctx context.Context) ([]Campaign, error) {
	var list []Campaign
	return list, s.db.WithContext(ctx).
		Where("status = ?", CampaignPending).
		Order("created_at ASC").Find(&list).Error
}

func (s *Store) FindActiveCampaigns(ctx context.Context) ([]Campaign, error) {
	var list []Campaign
	return list, s.db.WithContext(ctx).
		Where("status = ? AND (end_at IS NULL OR end_at > ?)", CampaignActive, time.Now()).
		Order("cpj DESC").Find(&list).Error
}

func (s *Store) ApproveCampaign(ctx context.Context, id uuid.UUID, reviewerID int64) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Campaign{}).Where("id = ?", id).Updates(map[string]any{
		"status":      CampaignActive,
		"reviewed_at": &now,
		"reviewer_id": reviewerID,
	}).Error
}

func (s *Store) RejectCampaign(ctx context.Context, id uuid.UUID, reviewerID int64, note string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Campaign{}).Where("id = ?", id).Updates(map[string]any{
		"status":      CampaignRejected,
		"review_note": note,
		"reviewed_at": &now,
		"reviewer_id": reviewerID,
	}).Error
}

func (s *Store) AddJoinCount(ctx context.Context, campaignID uuid.UUID, joins int, cost float64) error {
	return s.db.WithContext(ctx).Model(&Campaign{}).Where("id = ?", campaignID).Updates(map[string]any{
		"join_count": gorm.Expr("join_count + ?", joins),
		"spent":      gorm.Expr("spent + ?", cost),
	}).Error
}

// ── AdChannel ──────────────────────────────────────────────

func (s *Store) CreateChannel(ctx context.Context, ch *AdChannel) error {
	return s.db.WithContext(ctx).Create(ch).Error
}

func (s *Store) FindChannelByTelegramID(ctx context.Context, channelID int64) (*AdChannel, error) {
	var ch AdChannel
	err := s.db.WithContext(ctx).Where("channel_id = ?", channelID).First(&ch).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ch, err
}

func (s *Store) ListActiveChannels(ctx context.Context, minCPJ float64) ([]AdChannel, error) {
	var list []AdChannel
	return list, s.db.WithContext(ctx).
		Where("is_active = true AND is_verified = true AND cpj_rate <= ?", minCPJ).
		Order("member_count DESC").Find(&list).Error
}

func (s *Store) ListChannelsByOwner(ctx context.Context, ownerID uuid.UUID) ([]AdChannel, error) {
	var list []AdChannel
	return list, s.db.WithContext(ctx).Where("owner_id = ?", ownerID).Find(&list).Error
}

func (s *Store) VerifyChannel(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&AdChannel{}).Where("id = ?", id).Update("is_verified", true).Error
}

// ── Impression ─────────────────────────────────────────────

func (s *Store) CreateImpression(ctx context.Context, imp *Impression) error {
	return s.db.WithContext(ctx).Create(imp).Error
}

// ── Stats ──────────────────────────────────────────────────

type Stats struct {
	TotalCampaigns  int64
	ActiveCampaigns int64
	TotalChannels   int64
	TotalSpent      float64
	TotalJoins      int64
}

func (s *Store) GetStats(ctx context.Context) (*Stats, error) {
	var st Stats
	s.db.WithContext(ctx).Model(&Campaign{}).Count(&st.TotalCampaigns)
	s.db.WithContext(ctx).Model(&Campaign{}).Where("status = ?", CampaignActive).Count(&st.ActiveCampaigns)
	s.db.WithContext(ctx).Model(&AdChannel{}).Where("is_verified = true").Count(&st.TotalChannels)
	var sums struct {
		TotalSpent float64
		TotalJoins int64
	}
	s.db.WithContext(ctx).Model(&Campaign{}).
		Select("SUM(spent) as total_spent, SUM(join_count) as total_joins").Scan(&sums)
	st.TotalSpent = sums.TotalSpent
	st.TotalJoins = sums.TotalJoins
	return &st, nil
}

func (s *Store) FindPublisherByID(ctx context.Context, id uuid.UUID) (*Publisher, error) {
	var p Publisher
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

// ── AdConfig ────────────────────────────────────────────────

func (s *Store) GetConfig(ctx context.Context) (*AdConfig, error) {
	var cfg AdConfig
	err := s.db.WithContext(ctx).Where("is_active = true").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// seed default
		def := DefaultConfig()
		s.db.WithContext(ctx).Create(&def)
		return &def, nil
	}
	return &cfg, err
}

func (s *Store) UpdateConfig(ctx context.Context, cfg *AdConfig) error {
	return s.db.WithContext(ctx).Save(cfg).Error
}

// ── Category ────────────────────────────────────────────────

func (s *Store) ListCategories(ctx context.Context) ([]ChannelCategory, error) {
	var cats []ChannelCategory
	return cats, s.db.WithContext(ctx).Where("is_active = true").Order("label").Find(&cats).Error
}

func (s *Store) FindCategoryByName(ctx context.Context, name string) (*ChannelCategory, error) {
	var cat ChannelCategory
	err := s.db.WithContext(ctx).Where("name = ?", name).First(&cat).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
	return &cat, err
}

func (s *Store) SeedCategories(ctx context.Context) error {
	for _, cat := range DefaultCategories() {
		var existing ChannelCategory
		err := s.db.WithContext(ctx).Where("name = ?", cat.Name).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.db.WithContext(ctx).Create(&cat)
		}
	}
	return nil
}

// ── Channel ────────────────────────────────────────────────

func (s *Store) UpdateChannelAnalysis(ctx context.Context, id uuid.UUID, score int, fakePercent float64, realMembers int, effectiveCPJ float64) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&AdChannel{}).Where("id = ?", id).Updates(map[string]any{
		"score":            score,
		"fake_percent":     fakePercent,
		"real_members":     realMembers,
		"effective_cpj":    effectiveCPJ,
		"last_analyzed_at": &now,
	}).Error
}

func (s *Store) ListChannelsByScore(ctx context.Context, minScore int, categoryID *uuid.UUID, cpjBudget float64) ([]AdChannel, error) {
	var list []AdChannel
	q := s.db.WithContext(ctx).
		Where("is_active = true AND status = 'verified' AND score >= ? AND effective_cpj <= ?", minScore, cpjBudget)
	if categoryID != nil {
		q = q.Where("category_id = ?", *categoryID)
	}
	return list, q.Order("score DESC").Find(&list).Error
}

func (s *Store) SaveMemberAnalysis(ctx context.Context, m *MemberAnalysis) error {
	return s.db.WithContext(ctx).
		Where(MemberAnalysis{ChannelID: m.ChannelID, TelegramID: m.TelegramID}).
		Assign(*m).FirstOrCreate(m).Error
}

func (s *Store) GetChannelFakeStats(ctx context.Context, channelID int64) (total, fake int64, err error) {
	s.db.WithContext(ctx).Model(&MemberAnalysis{}).Where("channel_id = ?", channelID).Count(&total)
	s.db.WithContext(ctx).Model(&MemberAnalysis{}).Where("channel_id = ? AND is_fake = true", channelID).Count(&fake)
	return
}

func (s *Store) FindChannelByID(ctx context.Context, id uuid.UUID) (*AdChannel, error) {
	var ch AdChannel
	err := s.db.WithContext(ctx).Preload("Category").Where("id = ?", id).First(&ch).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
	return &ch, err
}

func (s *Store) ListUnverifiedChannels(ctx context.Context) ([]AdChannel, error) {
	var list []AdChannel
	return list, s.db.WithContext(ctx).
		Where("is_verified = false AND is_active = true").
		Order("created_at ASC").Find(&list).Error
}

func (s *Store) DeactivateChannel(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&AdChannel{}).Where("id = ?", id).
		Update("is_active", false).Error
}
