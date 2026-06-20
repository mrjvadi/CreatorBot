package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// ══════════════════════════════════════════════════════════════
// Lock Rental — اجاره‌ی قفل کانال روی ربات‌های رایگان
// ══════════════════════════════════════════════════════════════

func (s *Store) CreateLockRental(ctx context.Context, r *LockRentalCampaign) error {
	return s.db.WithContext(ctx).Create(r).Error
}

func (s *Store) FindLockRental(ctx context.Context, id uuid.UUID) (*LockRentalCampaign, error) {
	var r LockRentalCampaign
	err := s.db.WithContext(ctx).First(&r, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &r, err
}

// FindActiveRentalByChannel کمپین اجاره‌ای فعال متصل به یک کانال هدف خاص
// را پیدا می‌کند — برای پرداخت per-join وقتی membership.joined می‌رسد.
func (s *Store) FindActiveRentalByChannel(ctx context.Context, channelID int64) (*LockRentalCampaign, error) {
	var r LockRentalCampaign
	err := s.db.WithContext(ctx).
		Where("target_channel_id = ? AND status = ?", channelID, RentalActive).
		First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &r, err
}

// TryRecordJoinReward تلاش می‌کند یک رکورد پاداش ثبت کند. اگر این کاربر
// قبلاً برای همین کمپین پاداش گرفته باشد، false برمی‌گرداند (idempotency
// در سطح دیتابیس با unique constraint — حتی اگر دو درخواست هم‌زمان از دو
// instance مختلف ads-bot برسند، فقط یکی موفق می‌شود).
// RewardSettlementDelay مهلت انتظار بین join و واریز واقعی — فرصت برای
// تشخیص تقلب پیش از پرداخت نهایی.
const RewardSettlementDelay = 24 * time.Hour

func (s *Store) TryRecordJoinReward(ctx context.Context, rentalID uuid.UUID, telegramID int64, amount float64) (firstTime bool, err error) {
	res := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&RentalJoinReward{
			ID: uuid.New(), RentalID: rentalID, TelegramID: telegramID, AmountTON: amount,
			Status: RewardPending, SettleAt: time.Now().Add(RewardSettlementDelay),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// FindDueRewards پاداش‌های pending که مهلت‌شان رسیده را برمی‌گرداند —
// برای scheduler که هر چند دقیقه این‌ها را تسویه می‌کند.
func (s *Store) FindDueRewards(ctx context.Context, limit int) ([]RentalJoinReward, error) {
	var list []RentalJoinReward
	err := s.db.WithContext(ctx).
		Where("status = ? AND settle_at <= ?", RewardPending, time.Now()).
		Limit(limit).
		Find(&list).Error
	return list, err
}

// SettleReward یک پاداش pending را settled علامت می‌زند (بعد از واریز موفق).
func (s *Store) SettleReward(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&RentalJoinReward{}).
		Where("id = ? AND status = ?", id, RewardPending).
		Updates(map[string]any{"status": RewardSettled, "settled_at": &now}).Error
}

// ReverseReward یک پاداش pending را reversed علامت می‌زند (fraud-engine رد
// کرد) و بودجه‌ی رزروشده را به کمپین برمی‌گرداند.
func (s *Store) ReverseReward(ctx context.Context, id, rentalID uuid.UUID, amount float64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&RentalJoinReward{}).
			Where("id = ? AND status = ?", id, RewardPending).
			Update("status", RewardReversed)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // قبلاً پردازش شده — idempotent
		}
		// بودجه‌ی رزروشده برگردد (spent کم شود)
		return tx.Model(&LockRentalCampaign{}).
			Where("id = ?", rentalID).
			Update("spent", gorm.Expr("spent - ?", amount)).Error
	})
}

// ReversePendingRewardByUser وقتی fraud-engine یک کاربر را برای یک کانال
// خاص fraud تشخیص می‌دهد صدا زده می‌شود — پاداش pending آن کاربر برای هر
// کمپینی که هدفش همان کانال است (صرف‌نظر از وضعیت فعلی rental — کمپین
// ممکن است بین join و تشخیص تقلب منقضی شده باشد) لغو می‌شود. اگر هیچ
// پاداش pending ای پیدا نشود (یا قبلاً settled شده)، کاری انجام نمی‌دهد.
func (s *Store) ReversePendingRewardByUser(ctx context.Context, channelID, telegramID int64) error {
	var rewards []RentalJoinReward
	err := s.db.WithContext(ctx).
		Joins("JOIN lock_rental_campaigns ON lock_rental_campaigns.id = rental_join_rewards.rental_id").
		Where("lock_rental_campaigns.target_channel_id = ? AND rental_join_rewards.telegram_id = ? AND rental_join_rewards.status = ?",
			channelID, telegramID, RewardPending).
		Find(&rewards).Error
	if err != nil {
		return err
	}
	for _, reward := range rewards {
		if err := s.ReverseReward(ctx, reward.ID, reward.RentalID, reward.AmountTON); err != nil {
			return err
		}
	}
	return nil
}

// ── FreeBotOwnerReward (همان escrow per-join، برای owner ربات رایگان) ──

// TryRecordOwnerReward مثل TryRecordJoinReward ولی per-slot به‌جای
// per-user — idempotency در سطح دیتابیس با unique(rental_id, slot_id).
func (s *Store) TryRecordOwnerReward(ctx context.Context, rentalID, slotID uuid.UUID, ownerTelegramID int64, amount float64) (firstTime bool, err error) {
	res := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&FreeBotOwnerReward{
			ID: uuid.New(), RentalID: rentalID, SlotID: slotID,
			OwnerTelegramID: ownerTelegramID, AmountTON: amount,
			Status: RewardPending, SettleAt: time.Now().Add(RewardSettlementDelay),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// FindDueOwnerRewards مشابه FindDueRewards برای owner ربات‌های رایگان.
func (s *Store) FindDueOwnerRewards(ctx context.Context, limit int) ([]FreeBotOwnerReward, error) {
	var list []FreeBotOwnerReward
	err := s.db.WithContext(ctx).
		Where("status = ? AND settle_at <= ?", RewardPending, time.Now()).
		Limit(limit).
		Find(&list).Error
	return list, err
}

// SettleOwnerReward مشابه SettleReward.
func (s *Store) SettleOwnerReward(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&FreeBotOwnerReward{}).
		Where("id = ? AND status = ?", id, RewardPending).
		Updates(map[string]any{"status": RewardSettled, "settled_at": &now}).Error
}

func (s *Store) FindPendingLockRentals(ctx context.Context) ([]LockRentalCampaign, error) {
	var list []LockRentalCampaign
	return list, s.db.WithContext(ctx).
		Where("status = ?", RentalPendingReview).
		Order("created_at ASC").Find(&list).Error
}

// FindExpiredActiveRentals کمپین‌های active که EndAt شان گذشته را برمی‌گرداند
// — برای scheduler که باید این‌ها را به done تغییر دهد حتی اگر دیگر هیچ
// join جدیدی نمی‌آید که این تشخیص را trigger کند.
func (s *Store) FindExpiredActiveRentals(ctx context.Context) ([]LockRentalCampaign, error) {
	var list []LockRentalCampaign
	return list, s.db.WithContext(ctx).
		Where("status = ? AND end_at IS NOT NULL AND end_at <= ?", RentalActive, time.Now()).
		Find(&list).Error
}

// ApproveLockRental فقط باید توسط OWNER_ID پلتفرم صدا زده شود (چک در handler).
// DefaultRentalDuration مدت پیش‌فرض اعتبار یک کمپین اجاره‌ای از لحظه‌ی
// تأیید — اگر بودجه زودتر تمام نشود، کمپین بعد از این مدت خودکار به پایان
// می‌رسد. (در نسخه‌ی فعلی wizard از کاربر پرسیده نمی‌شود؛ این فقط یک سقف
// زمانی منطقی برای جلوگیری از کمپین‌های فعال بی‌پایان است.)
const DefaultRentalDuration = 30 * 24 * time.Hour

func (s *Store) ApproveLockRental(ctx context.Context, id uuid.UUID, reviewerID int64) error {
	now := time.Now()
	endAt := now.Add(DefaultRentalDuration)
	return s.db.WithContext(ctx).Model(&LockRentalCampaign{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      RentalActive,
			"reviewer_id": reviewerID,
			"reviewed_at": &now,
			"start_at":    &now,
			"end_at":      &endAt,
		}).Error
}

func (s *Store) RejectLockRental(ctx context.Context, id uuid.UUID, reviewerID int64, note string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&LockRentalCampaign{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      RentalRejected,
			"reviewer_id": reviewerID,
			"reviewed_at": &now,
			"review_note": note,
		}).Error
}

func (s *Store) AddRentalJoinCount(ctx context.Context, rentalID uuid.UUID, joins int, cost float64) error {
	return s.db.WithContext(ctx).Model(&LockRentalCampaign{}).
		Where("id = ?", rentalID).
		Updates(map[string]any{
			"total_joins": gorm.Expr("total_joins + ?", joins),
			"real_joins":  gorm.Expr("real_joins + ?", joins),
			"spent":       gorm.Expr("spent + ?", cost),
		}).Error
}

// MarkRentalDoneIfFinished کمپین را به "done" تغییر می‌دهد اگر بودجه تمام
// شده یا منقضی شده باشد — اتمیک با WHERE status='active' تا اگر چند join
// هم‌زمان این شرط را true کنند، فقط یکی واقعا transition را انجام دهد
// (RowsAffected>0 یعنی همین فراخوانی بود که کمپین را به پایان رساند، پس
// مسئول فرستادن اعلام اتمام است).
func (s *Store) MarkRentalDoneIfFinished(ctx context.Context, rentalID uuid.UUID) (justFinished bool, err error) {
	res := s.db.WithContext(ctx).Model(&LockRentalCampaign{}).
		Where("id = ? AND status = ? AND (spent >= budget OR (end_at IS NOT NULL AND end_at <= ?))",
			rentalID, RentalActive, time.Now()).
		Update("status", RentalDone)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// IncrementRentalSpentOnly فقط Spent را بالا می‌برد — برای مواردی که
// شمارنده‌ی join معنی ندارد (مثلا سهم owner ربات رایگان).
func (s *Store) IncrementRentalSpentOnly(ctx context.Context, rentalID uuid.UUID, cost float64) error {
	return s.db.WithContext(ctx).Model(&LockRentalCampaign{}).
		Where("id = ?", rentalID).
		Update("spent", gorm.Expr("spent + ?", cost)).Error
}

// ── FreeBotSlot ──────────────────────────────────────────────

func (s *Store) UpsertFreeBotSlot(ctx context.Context, slot *FreeBotSlot) error {
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "bot_id"}},
			DoNothing: true,
		}).Create(slot).Error
}

func (s *Store) FindFreeBotSlotByBotID(ctx context.Context, botID int64) (*FreeBotSlot, error) {
	var slot FreeBotSlot
	err := s.db.WithContext(ctx).First(&slot, "bot_id = ?", botID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &slot, err
}

// ListFreeSlots ربات‌های رایگانی که هنوز به هیچ کمپینی وصل نیستند را برمی‌گرداند.
func (s *Store) ListFreeSlots(ctx context.Context, limit int) ([]FreeBotSlot, error) {
	var list []FreeBotSlot
	q := s.db.WithContext(ctx).Where("rental_id IS NULL")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return list, q.Find(&list).Error
}

// ListSlotsByRental ربات‌های متصل به یک کمپین اجاره‌ای خاص را برمی‌گرداند.
func (s *Store) ListSlotsByRental(ctx context.Context, rentalID uuid.UUID) ([]FreeBotSlot, error) {
	var list []FreeBotSlot
	return list, s.db.WithContext(ctx).Where("rental_id = ?", rentalID).Find(&list).Error
}

// AssignSlotsToRental N تا از ربات‌های آزاد را به یک کمپین تأیید‌شده وصل می‌کند
// و در اختیار خریدار قرار می‌دهد (طبق گفته‌ی کاربر: تحویل بعد از تأیید ادمین).
func (s *Store) AssignSlotsToRental(ctx context.Context, rentalID uuid.UUID, buyerTelegramID int64, count int) ([]FreeBotSlot, error) {
	var assigned []FreeBotSlot
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var free []FreeBotSlot
		if err := tx.Where("rental_id IS NULL").Limit(count).Find(&free).Error; err != nil {
			return err
		}
		for i := range free {
			free[i].RentalID = &rentalID
			free[i].AssignedOwnerTelegramID = buyerTelegramID
			free[i].IsChannelAdminConfirmed = false
			if err := tx.Save(&free[i]).Error; err != nil {
				return err
			}
		}
		assigned = free
		return nil
	})
	return assigned, err
}

// ReleaseSlot یک ربات را از کمپین جدا و به حالت رایگان/در دسترس برمی‌گرداند
// (مثلا وقتی بودجه‌ی کمپین تمام شد یا rejected شد).
func (s *Store) ReleaseSlot(ctx context.Context, slotID uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&FreeBotSlot{}).
		Where("id = ?", slotID).
		Updates(map[string]any{
			"rental_id":                  nil,
			"assigned_owner_telegram_id": 0,
			"is_channel_admin_confirmed": false,
		}).Error
}

// ConfirmChannelAdmin وقتی خریدار ربات را در کانال خودش ادمین کرد صدا زده می‌شود.
// از همین لحظه سرویس باید شروع به قفل‌کردن برای آن تبلیغ بکند.
func (s *Store) ConfirmChannelAdmin(ctx context.Context, slotID uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&FreeBotSlot{}).
		Where("id = ?", slotID).
		Update("is_channel_admin_confirmed", true).Error
}

// ResolveSlotOwnerTelegramID آیدی تلگرام صاحب واقعی یک ربات رایگان را پیدا
// می‌کند — کسی که از botmanager این instance را ساخته (نه خریدار اجاره).
// raw query عمدی است (نه import مدل botmanager) تا coupling نداشته باشیم؛
// همان الگوی ValidateServiceID در botpay.
func (s *Store) ResolveSlotOwnerTelegramID(ctx context.Context, botInstanceID uuid.UUID) (int64, error) {
	var row struct{ TelegramID int64 }
	err := s.db.WithContext(ctx).
		Table("bot_instances").
		Select("users.telegram_id as telegram_id").
		Joins("JOIN users ON users.id = bot_instances.owner_id").
		Where("bot_instances.id = ?", botInstanceID).
		Take(&row).Error
	if err != nil {
		return 0, err
	}
	return row.TelegramID, nil
}
