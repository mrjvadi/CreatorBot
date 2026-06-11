// Package store provides repository methods used by botmanager and apimanager.
// agentmanager does not use the store directly (it only writes heartbeats via
// the Notifier and reads commands from the stream).
//
// All methods depend only on ports.DB — swap the DB adapter in main.go
// without touching this file.
package store

import (
	"context"
	"fmt"
	"strings"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Store aggregates all botmanager repositories.
type Store struct{ db ports.DB }

func New(db ports.DB) *Store { return &Store{db: db} }

// ---- User ----

func (s *Store) FindUserByTelegramID(ctx context.Context, id int64) (*models.User, error) {
	var u models.User
	err := s.db.Conn().WithContext(ctx).Where("telegram_id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (s *Store) UpsertUser(ctx context.Context, u *models.User) error {
	return s.db.Conn().WithContext(ctx).
		Where(models.User{TelegramID: u.TelegramID}).
		Assign(*u).
		FirstOrCreate(u).Error
}

// ---- Server ----

func (s *Store) ListServers(ctx context.Context) ([]models.Server, error) {
	var list []models.Server
	return list, s.db.Conn().WithContext(ctx).Find(&list).Error
}

func (s *Store) CreateServer(ctx context.Context, srv *models.Server) error {
	return s.db.Conn().WithContext(ctx).Create(srv).Error
}

func (s *Store) DeleteServer(ctx context.Context, id any) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.Server{}, "id = ?", id).Error
}

// MarkServerOnline updates IsOnline=true and LastSeen=now.
// Called by apimanager when it receives a heartbeat from agentmanager.
func (s *Store) MarkServerOnline(ctx context.Context, serverID any) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.Server{}).
		Where("id = ?", serverID).
		Updates(map[string]any{"is_online": true, "last_seen": gorm.Expr("NOW()")}).Error
}

// MarkServerOffline sets IsOnline=false for servers whose last heartbeat
// is older than the given threshold. Called by a background job in apimanager.
func (s *Store) MarkStaleServersOffline(ctx context.Context, thresholdSeconds int) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.Server{}).
		Where("is_online = true AND last_seen < NOW() - INTERVAL '? seconds'", thresholdSeconds).
		Update("is_online", false).Error
}

// ---- Template ----

func (s *Store) ListTemplates(ctx context.Context) ([]models.BotTemplate, error) {
	var list []models.BotTemplate
	return list, s.db.Conn().WithContext(ctx).Where("is_active = true").Find(&list).Error
}

func (s *Store) FindTemplate(ctx context.Context, id any) (*models.BotTemplate, error) {
	var t models.BotTemplate
	err := s.db.Conn().WithContext(ctx).First(&t, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

func (s *Store) CreateTemplate(ctx context.Context, t *models.BotTemplate) error {
	return s.db.Conn().WithContext(ctx).Create(t).Error
}

// ---- Instance ----

func (s *Store) ListInstancesByOwner(ctx context.Context, ownerID any) ([]models.BotInstance, error) {
	var list []models.BotInstance
	return list, s.db.Conn().WithContext(ctx).Where("owner_id = ?", ownerID).Find(&list).Error
}

func (s *Store) FindInstance(ctx context.Context, id any) (*models.BotInstance, error) {
	var inst models.BotInstance
	err := s.db.Conn().WithContext(ctx).First(&inst, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &inst, err
}

func (s *Store) CreateInstance(ctx context.Context, inst *models.BotInstance) error {
	return s.db.Conn().WithContext(ctx).Create(inst).Error
}

func (s *Store) UpdateInstanceStatus(ctx context.Context, id any, status models.InstanceStatus) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.BotInstance{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (s *Store) DeleteInstance(ctx context.Context, id any) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.BotInstance{}, "id = ?", id).Error
}

// ---- Plan ----

func (s *Store) ListPlans(ctx context.Context) ([]models.Plan, error) {
	var list []models.Plan
	return list, s.db.Conn().WithContext(ctx).Where("is_active = true").Find(&list).Error
}

// ---- Plan (write) ----

func (s *Store) CreatePlan(ctx context.Context, p *models.Plan) error {
	return s.db.Conn().WithContext(ctx).Create(p).Error
}

// ---- Instance (list all) ----

func (s *Store) ListAllInstances(ctx context.Context) ([]models.BotInstance, error) {
	var list []models.BotInstance
	return list, s.db.Conn().WithContext(ctx).Find(&list).Error
}

// ---- User management ----

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	var list []models.User
	return list, s.db.Conn().WithContext(ctx).Find(&list).Error
}

func (s *Store) SetUserBlocked(ctx context.Context, id any, blocked bool) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Update("is_blocked", blocked).Error
}

func (s *Store) SetUserRole(ctx context.Context, id any, role models.UserRole) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Update("role", role).Error
}

// ---- InviteLink ----

func (s *Store) CreateInviteLink(ctx context.Context, link *models.InviteLink) error {
	return s.db.Conn().WithContext(ctx).Create(link).Error
}

func (s *Store) FindInviteLinkByToken(ctx context.Context, token string) (*models.InviteLink, error) {
	var link models.InviteLink
	err := s.db.Conn().WithContext(ctx).Where("token = ?", token).First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &link, err
}

func (s *Store) ListInviteLinks(ctx context.Context, createdBy int64) ([]models.InviteLink, error) {
	var list []models.InviteLink
	return list, s.db.Conn().WithContext(ctx).
		Where("created_by = ?", createdBy).
		Order("created_at DESC").
		Find(&list).Error
}

func (s *Store) IncrementInviteUse(ctx context.Context, token string, instanceID *models.InviteLink) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.InviteLink{}).
		Where("token = ?", token).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

func (s *Store) DeleteInviteLink(ctx context.Context, token string) error {
	return s.db.Conn().WithContext(ctx).
		Where("token = ?", token).
		Delete(&models.InviteLink{}).Error
}

func (s *Store) FindTemplateByType(ctx context.Context, botType string) (*models.BotTemplate, error) {
	var t models.BotTemplate
	err := s.db.Conn().WithContext(ctx).
		Where("type = ? AND is_active = true", botType).
		First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

// UpdateInstanceStatusByContainerName وضعیت instance را از طریق نام container به‌روز می‌کند.
// agentlistener از این متد برای sync وضعیت از heartbeat استفاده می‌کند.
func (s *Store) UpdateInstanceStatusByContainerName(ctx context.Context, containerName string, status models.InstanceStatus) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.BotInstance{}).
		Where("container_name = ?", containerName).
		Update("status", status).Error
}

// MarkServerOnlineByServerID سرور را بر اساس Server.ID (نه uuid بلکه string) آنلاین علامت می‌زند.
// agentlistener این را با payload.ServerID (که UUID سرور است) صدا می‌زند.
func (s *Store) MarkServerOnlineByServerID(ctx context.Context, serverID string) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.Server{}).
		Where("id::text = ?", serverID).
		Updates(map[string]any{"is_online": true, "last_seen": gorm.Expr("NOW()")}).Error
}

// FindBestOnlineServer اولین سرور آنلاین را با کمترین بار برمی‌گرداند.
func (s *Store) FindBestOnlineServer(ctx context.Context) (*models.Server, error) {
	var srv models.Server
	// سرورهایی که آنلاین هستند و heartbeat آن‌ها اخیراً دریافت شده
	err := s.db.Conn().WithContext(ctx).
		Where("is_online = true AND last_seen > NOW() - INTERVAL '30 seconds'").
		Order("last_seen DESC").
		First(&srv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &srv, err
}

// ---- Schema-aware Instance operations ----

// CreateInstanceWithSchema یک BotInstance ساخته و schema مربوطه را در DB ایجاد می‌کند.
func (s *Store) CreateInstanceWithSchema(ctx context.Context, inst *models.BotInstance) error {
	// ساخت نام schema
	if inst.DBSchema == "" {
		inst.DBSchema = "inst_" + inst.ID.String()[:8]
	}

	// ابتدا instance را در DB ذخیره کن
	if err := s.db.Conn().WithContext(ctx).Create(inst).Error; err != nil {
		return err
	}

	// سپس schema را بساز
	if err := s.db.Conn().WithContext(ctx).Exec(
		"CREATE SCHEMA IF NOT EXISTS " + inst.DBSchema,
	).Error; err != nil {
		// اگه schema ساخته نشد، instance را هم rollback کن
		s.db.Conn().WithContext(ctx).Delete(inst)
		return fmt.Errorf("create schema %s: %w", inst.DBSchema, err)
	}

	return nil
}

// DropInstanceSchema schema یک instance را حذف می‌کند.
// معمولاً قبل از DeleteInstance صدا زده می‌شود.
func (s *Store) DropInstanceSchema(ctx context.Context, instanceID any) error {
	inst, err := s.FindInstance(ctx, instanceID)
	if err != nil || inst == nil {
		return err
	}
	if inst.DBSchema == "" {
		return nil
	}
	return s.db.Conn().WithContext(ctx).Exec(
		"DROP SCHEMA IF EXISTS " + inst.DBSchema + " CASCADE",
	).Error
}

// GetInstanceDSN یک DSN با search_path برای instance مشخص می‌سازد.
// ربات‌های deploy شده از این DSN برای اتصال به DB استفاده می‌کنند.
func (s *Store) GetInstanceDSN(ctx context.Context, instanceID any, baseDSN string) (string, error) {
	inst, err := s.FindInstance(ctx, instanceID)
	if err != nil || inst == nil {
		return "", fmt.Errorf("instance not found")
	}
	if inst.DBSchema == "" {
		return "", fmt.Errorf("instance has no schema")
	}
	sep := "?"
	if strings.Contains(baseDSN, "?") {
		sep = "&"
	}
	return baseDSN + sep + "search_path=" + inst.DBSchema, nil
}

// FindInstanceByBotID یک instance را با Bot ID پیدا می‌کند.
// برای جلوگیری از ثبت مجدد یک ربات با توکن جدید استفاده می‌شود.
func (s *Store) FindInstanceByBotID(ctx context.Context, botID int64) (*models.BotInstance, error) {
	var inst models.BotInstance
	err := s.db.Conn().WithContext(ctx).
		Where("bot_id = ?", botID).
		First(&inst).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &inst, err
}

// UpdateBotToken توکن رمزنگاری‌شده یک instance را آپدیت می‌کند.
// Bot ID تغییر نمی‌کند — فقط توکن جدید ذخیره می‌شود.
func (s *Store) UpdateBotToken(ctx context.Context, botID int64, encryptedToken string) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.BotInstance{}).
		Where("bot_id = ?", botID).
		Update("bot_token", encryptedToken).Error
}

// ---- Payment ----

func (s *Store) CreatePayment(ctx context.Context, p *models.Payment) error {
	return s.db.Conn().WithContext(ctx).Create(p).Error
}

func (s *Store) FindPayment(ctx context.Context, id string) (*models.Payment, error) {
	var p models.Payment
	err := s.db.Conn().WithContext(ctx).Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (s *Store) UpdatePayment(ctx context.Context, p *models.Payment) error {
	return s.db.Conn().WithContext(ctx).Save(p).Error
}

// ---- Subscription ----

func (s *Store) CreateSubscription(ctx context.Context, sub *models.Subscription) error {
	return s.db.Conn().WithContext(ctx).Create(sub).Error
}

func (s *Store) GetActiveSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	var subs []models.Subscription
	if err := s.db.Conn().WithContext(ctx).
		Where("user_id = ? AND is_active = true", userID).
		Limit(1).Find(&subs).Error; err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, nil
	}
	return &subs[0], nil
}

func (s *Store) GetFreePlan(ctx context.Context) (*models.Plan, error) {
	var plan models.Plan
	err := s.db.Conn().WithContext(ctx).
		Where("is_free = true AND is_active = true").
		First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &plan, err
}

func (s *Store) ListActivePlans(ctx context.Context) ([]models.Plan, error) {
	var plans []models.Plan
	err := s.db.Conn().WithContext(ctx).
		Where("is_active = true").
		Order("price ASC").
		Find(&plans).Error
	return plans, err
}

func (s *Store) FindPlan(ctx context.Context, id string) (*models.Plan, error) {
	var plan models.Plan
	err := s.db.Conn().WithContext(ctx).Where("id = ?", id).First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &plan, err
}

