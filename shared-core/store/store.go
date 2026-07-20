// Package store provides repository methods used by botmanager and apimanager.
// agentmanager does not use the store directly (it only writes heartbeats via
// the Notifier and reads commands from the stream).
//
// All methods depend only on ports.DB — swap the DB adapter in main.go
// without touching this file.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// خطاهای شناخته‌شده‌ی redeem — کالر (botmanager) از errors.Is برای تشخیص
// دقیقِ دلیلِ رد شدن استفاده می‌کند (به‌جای string matching روی پیام).
var (
	ErrPromoNotRedeemable   = errors.New("promo code not redeemable")
	ErrPromoAlreadyRedeemed = errors.New("promo code already redeemed by this user")
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

// MarkServerOnline updates IsOnline=true and LastSeen=now. OnlineSince only moves forward
// if the server was previously offline (or never had one) — a server that's continuously
// online keeps its original OnlineSince across repeated heartbeats.
// Called by apimanager when it receives a heartbeat from agentmanager.
func (s *Store) MarkServerOnline(ctx context.Context, serverID any) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.Server{}).
		Where("id = ?", serverID).
		Updates(map[string]any{
			"is_online": true,
			"last_seen": gorm.Expr("NOW()"),
			"online_since": gorm.Expr(
				"CASE WHEN is_online = false OR online_since IS NULL THEN NOW() ELSE online_since END"),
		}).Error
}

// MarkServerOffline sets IsOnline=false for servers whose last heartbeat
// is older than the given threshold. Called by a background job in apimanager.
//
// باگ واقعی که این‌جا بود (۲۰۲۶-۰۷-۰۵، از لاگ واقعی پیدا شد): placeholder ی «?» داخل رشته‌ی
// تک‌کوتیشن `INTERVAL '? seconds'` قرار داشت — یعنی از نظر GORM/driver داخل یک literal رشته‌ای
// بود، نه یک جای‌گزینِ واقعی، و باعث خطای «mismatched param and argument count» می‌شد. این کد
// از قبل در فایل بود ولی تا این‌که MarkStaleServersOffline واقعاً صدا زده شود (تازه در همین
// جلسه، با goroutine دوره‌ای در main.go) هیچ‌وقت اجرا نشده بود که این باگ خودش را نشان بدهد.
// راه‌حل: عدد را با ضرب در INTERVAL '1 second' بیرون از هر quote قرار می‌دهیم — یک الگوی
// استاندارد و امنِ پارامتردارکردنِ interval در PostgreSQL.
func (s *Store) MarkStaleServersOffline(ctx context.Context, thresholdSeconds int) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.Server{}).
		Where("is_online = true AND last_seen < NOW() - (? * INTERVAL '1 second')", thresholdSeconds).
		Updates(map[string]any{"is_online": false, "online_since": nil}).Error
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

// UpdateTemplate فیلدهای یک تمپلیت موجود را ذخیره می‌کند (t.ID باید ست شده باشد).
func (s *Store) UpdateTemplate(ctx context.Context, t *models.BotTemplate) error {
	return s.db.Conn().WithContext(ctx).Save(t).Error
}

// UpdateTemplateSchema فقط فیلد config_schema یک قالب را به‌روز می‌کند.
func (s *Store) UpdateTemplateSchema(ctx context.Context, id any, schema string) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.BotTemplate{}).
		Where("id = ?", id).
		Update("config_schema", schema).Error
}

// DeleteTemplate یک تمپلیت را حذف می‌کند (soft delete).
func (s *Store) DeleteTemplate(ctx context.Context, id any) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.BotTemplate{}, "id = ?", id).Error
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

// UpdateInstance رکورد کاملِ instance را ذخیره می‌کند — برای ویرایش دستیِ ادمین (انقضا/پلن/
// lock mode) از پنل «همه‌ی ربات‌ها» که قبلاً هیچ راه ویرایشی نداشت (بازخورد کاربر ۲۰۲۶-۰۷-۰۵).
func (s *Store) UpdateInstance(ctx context.Context, inst *models.BotInstance) error {
	return s.db.Conn().WithContext(ctx).Save(inst).Error
}

// UpdateInstanceServer یک instance را به سرور دیگری منتقل می‌کند (فقط رکورد DB — خودِ
// stop/deploy روی سرورهای مبدا/مقصد جداگانه توسط handler انجام می‌شود). بازخورد کاربر
// ۲۰۲۶-۰۷-۰۵: «بتونم کانتینر رو از یه سرور به سرور دیگه منتقل کنم».
func (s *Store) UpdateInstanceServer(ctx context.Context, id any, serverID any) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.BotInstance{}).
		Where("id = ?", id).
		Update("server_id", serverID).Error
}

func (s *Store) DeleteInstance(ctx context.Context, id any) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.BotInstance{}, "id = ?", id).Error
}

// UpdateInstanceEnvOverrides مقادیرِ تنظیماتِ کاربرمحورِ instance را ذخیره می‌کند (رشته‌ی
// JSON، طبق فیلد EnvOverrides). این مقادیر تا restart بعدی روی کانتینر واقعی اعمال نمی‌شوند.
func (s *Store) UpdateInstanceEnvOverrides(ctx context.Context, id any, envJSON string) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.BotInstance{}).
		Where("id = ?", id).
		Update("env_overrides", envJSON).Error
}

// UpdateInstanceExpiry تاریخ انقضای یک instance را به‌روزرسانی می‌کند.
// expiresAt = nil یعنی ابدی (بدون انقضا).
func (s *Store) UpdateInstanceExpiry(ctx context.Context, id any, expiresAt *time.Time) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.BotInstance{}).
		Where("id = ?", id).
		Update("expires_at", expiresAt).Error
}

// ListInstancesExpiringBetween instanceهایی که انقضایشان در بازه‌ی [from, to] است
// و حذف‌نشده‌اند را برمی‌گرداند (برای یادآور انقضا).
func (s *Store) ListInstancesExpiringBetween(ctx context.Context, from, to time.Time) ([]models.BotInstance, error) {
	var list []models.BotInstance
	err := s.db.Conn().WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at BETWEEN ? AND ?", from, to).
		Where("status <> ?", models.StatusDeleted).
		Find(&list).Error
	return list, err
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

// ListAllPlans همه‌ی پلن‌ها را برمی‌گرداند (فعال و غیرفعال) — برای پنل مدیریت،
// برخلاف ListPlans/ListActivePlans که فقط فعال‌ها را برای ویترین خرید نشان می‌دهند.
func (s *Store) ListAllPlans(ctx context.Context) ([]models.Plan, error) {
	var list []models.Plan
	return list, s.db.Conn().WithContext(ctx).Preload("Limits").Order("price ASC").Find(&list).Error
}

// UpdatePlan فیلدهای یک پلن موجود را ذخیره می‌کند (p.ID باید ست شده باشد).
func (s *Store) UpdatePlan(ctx context.Context, p *models.Plan) error {
	return s.db.Conn().WithContext(ctx).Save(p).Error
}

// DeletePlan یک پلن را حذف می‌کند (soft delete، چون Plan از Base ارث می‌برد).
func (s *Store) DeletePlan(ctx context.Context, id any) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.Plan{}, "id = ?", id).Error
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

// ListServiceTypes انواع سرویسِ فعال (distinct type) را برمی‌گرداند.
// منبعِ حقیقتِ «سرویس‌های پویا» — هیچ نوعی در کد hardcode نیست.
func (s *Store) ListServiceTypes(ctx context.Context) ([]string, error) {
	var types []string
	err := s.db.Conn().WithContext(ctx).
		Model(&models.BotTemplate{}).
		Where("is_active = true").
		Distinct().
		Order("type").
		Pluck("type", &types).Error
	return types, err
}

// ListTemplatesByType همه‌ی تمپلیت‌های فعالِ یک نوع سرویس (هر کدام یک تگ).
// مرتب بر اساس جدیدترین (created_at desc) تا اولین مورد = جدیدترین تگ باشد.
func (s *Store) ListTemplatesByType(ctx context.Context, serviceType string) ([]models.BotTemplate, error) {
	var list []models.BotTemplate
	err := s.db.Conn().WithContext(ctx).
		Where("type = ? AND is_active = true", serviceType).
		Order("created_at DESC").
		Find(&list).Error
	return list, err
}

// FindTemplateByTypeAndTag تمپلیتِ یک نوع سرویس با تگِ (ImageTag) مشخص.
func (s *Store) FindTemplateByTypeAndTag(ctx context.Context, serviceType, tag string) (*models.BotTemplate, error) {
	var t models.BotTemplate
	err := s.db.Conn().WithContext(ctx).
		Where("type = ? AND image_tag = ? AND is_active = true", serviceType, tag).
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
		Where("id = ?::uuid", serverID).
		Updates(map[string]any{
			"is_online": true,
			"last_seen": gorm.Expr("NOW()"),
			"online_since": gorm.Expr(
				"CASE WHEN is_online = false OR online_since IS NULL THEN NOW() ELSE online_since END"),
		}).Error
}

// RecordHeartbeat مثل MarkServerOnlineByServerID عمل می‌کند (is_online/last_seen/online_since)
// و علاوه بر آن آمار اختیاریِ CPU/RAM (فعلاً همیشه nil — رجوع به کامنت HeartbeatMsg در
// shared-core/protocol) و آخرین اسنپ‌شات containers را هم ذخیره می‌کند. این دومی از قبل در
// هر heartbeat می‌آمد ولی هیچ‌جا ذخیره نمی‌شد (بازخورد کاربر ۲۰۲۶-۰۷-۰۳: نمایش کامل سرورها).
func (s *Store) RecordHeartbeat(
	ctx context.Context,
	serverID string,
	cpuPercent *float64,
	memUsedMB, memTotalMB *int64,
	containersJSON string,
) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.Server{}).
		Where("id = ?::uuid", serverID).
		Updates(map[string]any{
			"is_online": true,
			"last_seen": gorm.Expr("NOW()"),
			"online_since": gorm.Expr(
				"CASE WHEN is_online = false OR online_since IS NULL THEN NOW() ELSE online_since END"),
			"cpu_percent":     cpuPercent,
			"memory_used_mb":  memUsedMB,
			"memory_total_mb": memTotalMB,
			"last_containers": containersJSON,
		}).Error
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

func (s *Store) ListPaymentsByUser(ctx context.Context, userID any) ([]models.Payment, error) {
	var list []models.Payment
	return list, s.db.Conn().WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&list).Error
}

func (s *Store) ListAllPayments(ctx context.Context) ([]models.Payment, error) {
	var list []models.Payment
	return list, s.db.Conn().WithContext(ctx).Order("created_at DESC").Find(&list).Error
}

// ---- Subscription ----

func (s *Store) CreateSubscription(ctx context.Context, sub *models.Subscription) error {
	return s.db.Conn().WithContext(ctx).Create(sub).Error
}

// ActivateSubscription تعویض پلن را اتمیک می‌کند: اشتراک‌های فعال قبلی همان
// کاربر غیرفعال و رکورد جدید در همان transaction ساخته می‌شود.
func (s *Store) ActivateSubscription(ctx context.Context, sub *models.Subscription) error {
	return s.db.Conn().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Subscription{}).
			Where("user_id = ? AND is_active = ?", sub.UserID, true).
			Update("is_active", false).Error; err != nil {
			return err
		}
		return tx.Create(sub).Error
	})
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
	if isBlankID(id) {
		return nil, nil // از کوئریِ بی‌فایده با UUID صفر/خالی جلوگیری کن
	}
	var plan models.Plan
	err := s.db.Conn().WithContext(ctx).Where("id = ?", id).First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &plan, err
}

// isBlankID بررسی می‌کند شناسه خالی یا UUID صفر است.
func isBlankID(id string) bool {
	return id == "" || id == uuid.Nil.String()
}

// ---- Capacity Engine ----

// CountInstancesByOwnerAndType تعداد ربات‌های فعال یک کاربر از یک نوع.
func (s *Store) CountInstancesByOwnerAndType(ctx context.Context, ownerID uuid.UUID, botType string) (int, error) {
	// پیدا کردن template_id های این نوع سرویس
	var templateIDs []string
	if err := s.db.Conn().WithContext(ctx).
		Model(&models.BotTemplate{}).
		Where("type = ? AND is_active = true", botType).
		Pluck("id::text", &templateIDs).Error; err != nil || len(templateIDs) == 0 {
		return 0, err
	}

	// شمارش instance های این کاربر با این template ها
	var count int64
	err := s.db.Conn().WithContext(ctx).
		Model(&models.BotInstance{}).
		Where("owner_id = ?", ownerID).
		Where("template_id::text IN ?", templateIDs).
		Count(&count).Error
	return int(count), err
}

// FindPlanWithLimits پلن را با limit هایش بارگذاری می‌کند.
func (s *Store) FindPlanWithLimits(ctx context.Context, id string) (*models.Plan, error) {
	if isBlankID(id) {
		return nil, nil // از کوئریِ بی‌فایده با UUID صفر/خالی جلوگیری کن
	}
	var plan models.Plan
	err := s.db.Conn().WithContext(ctx).
		Preload("Limits").
		Where("id = ?", id).First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &plan, err
}

// SetPlanLimit محدودیت یک نوع ربات را برای پلن تنظیم می‌کند (upsert).
func (s *Store) SetPlanLimit(ctx context.Context, planID uuid.UUID, botType string, maxBots int) error {
	var existing models.PlanBotLimit
	err := s.db.Conn().WithContext(ctx).
		Where("plan_id = ? AND bot_type = ?", planID, botType).
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.db.Conn().WithContext(ctx).Create(&models.PlanBotLimit{
			PlanID:  planID,
			BotType: botType,
			MaxBots: maxBots,
		}).Error
	}
	if err != nil {
		return err
	}
	return s.db.Conn().WithContext(ctx).
		Model(&existing).Update("max_bots", maxBots).Error
}

// UpdatePlanMaxBots سقف کلی ربات‌های یک پلن را آپدیت می‌کند.
func (s *Store) UpdatePlanMaxBots(ctx context.Context, planID uuid.UUID, maxBots int) error {
	if maxBots < 0 {
		maxBots = 0
	}
	return s.db.Conn().WithContext(ctx).
		Model(&models.Plan{}).
		Where("id = ?", planID).
		Update("max_bots", maxBots).Error
}

// CanCreateInstance بررسی کامل ظرفیت — قلب Capacity Engine.
// خروجی: (مجاز است؟, تعداد فعلی, حداکثر مجاز, خطا)
func (s *Store) CanCreateInstance(ctx context.Context, userID uuid.UUID, botType string) (bool, int, int, error) {
	sub, err := s.GetActiveSubscription(ctx, userID)
	if err != nil {
		return false, 0, 0, err
	}
	if sub == nil {
		return false, 0, 0, nil // اشتراکی ندارد
	}
	// انقضا
	if sub.ExpiresAt != nil && timeNow().After(*sub.ExpiresAt) {
		return false, 0, 0, nil
	}

	plan, err := s.FindPlanWithLimits(ctx, sub.PlanID.String())
	if err != nil || plan == nil {
		return false, 0, 0, err
	}

	limit := plan.LimitFor(botType)
	if limit <= 0 {
		return false, 0, limit, nil // این نوع ربات در پلن مجاز نیست
	}

	current, err := s.CountInstancesByOwnerAndType(ctx, userID, botType)
	if err != nil {
		return false, current, limit, err
	}

	return current < limit, current, limit, nil
}

func timeNow() time.Time { return time.Now() }

// FindUsersByIDs چند کاربر را یک‌جا می‌گیرد — برای جاهایی که قبلاً در یک
// حلقه FindUserByID صدا زده می‌شد (N+1)، مثل RunExpiryScan.
func (s *Store) FindUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]models.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []models.User
	err := s.db.Conn().WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	return users, err
}

func (s *Store) FindUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var u models.User
	err := s.db.Conn().WithContext(ctx).Where("id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (s *Store) FindServerByID(ctx context.Context, id string) (*models.Server, error) {
	var s2 models.Server
	err := s.db.Conn().WithContext(ctx).Where("id = ?", id).First(&s2).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &s2, err
}

func (s *Store) FindInstanceByContainerName(ctx context.Context, name string) (*models.BotInstance, error) {
	var inst models.BotInstance
	err := s.db.Conn().WithContext(ctx).Where("container_name = ?", name).First(&inst).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &inst, err
}

func (s *Store) ListPlansByType(ctx context.Context, serviceType string) ([]models.Plan, error) {
	// پیدا کردن template های این نوع
	var tmplIDs []string
	if err := s.db.Conn().WithContext(ctx).
		Model(&models.BotTemplate{}).
		Where("type = ? AND is_active = true", serviceType).
		Pluck("id::text", &tmplIDs).Error; err != nil || len(tmplIDs) == 0 {
		return nil, err
	}

	// پلن های این template ها
	var plans []models.Plan
	err := s.db.Conn().WithContext(ctx).
		Where("is_active = true AND template_id::text IN ?", tmplIDs).
		Order("price ASC").
		Find(&plans).Error
	return plans, err
}

// SelectLeastLoadedServer سرورِ آنلاینِ با کمترین تعداد instance فعال را برمی‌گرداند —
// برخلاف FindBestOnlineServer (که فقط بر اساس تازگی heartbeat انتخاب می‌کند و قبلاً این‌جا
// این متد اصلاً صدا زده نمی‌شد)، این یکی واقعاً بار را بین سرورها پخش می‌کند.
//
// requiredTag اگر خالی نباشد، فقط سرورهایی که این تگ را در Tags شان دارند در نظر گرفته
// می‌شوند (بازخورد کاربر ۲۰۲۶-۰۷-۰۵: «فقط پنل‌های فری به سرور با تگ فری بیاد» — caller باید
// روی fallback به "" تصمیم بگیرد اگر سروری با آن تگ پیدا نشد).
// سرورهایی که به MaxContainers شان رسیده‌اند (۰ = نامحدود) از انتخاب کنار گذاشته می‌شوند.
func (s *Store) SelectLeastLoadedServer(ctx context.Context, requiredTag string) (*models.Server, error) {
	var server models.Server
	err := s.db.Conn().WithContext(ctx).
		Raw(`
		SELECT s.*
		FROM servers s
		WHERE s.is_online = true
		  AND s.deleted_at IS NULL
		  AND (s.max_containers = 0 OR s.max_containers > (
			SELECT COUNT(*) FROM bot_instances bi
			WHERE bi.server_id::text = s.id::text
			  AND bi.status != 'deleted'
			  AND bi.deleted_at IS NULL
		  ))
		  AND (? = '' OR (',' || COALESCE(s.tags, '') || ',') LIKE '%,' || ? || ',%')
		ORDER BY (
			SELECT COUNT(*) FROM bot_instances bi
			WHERE bi.server_id::text = s.id::text
			  AND bi.status != 'deleted'
			  AND bi.deleted_at IS NULL
		) ASC
		LIMIT 1
	`, requiredTag, requiredTag).First(&server).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &server, err
}

// UpdateServer اطلاعات یک سرور را کامل ذخیره می‌کند (نام/آی‌پی/تگ‌ها/سقف container).
func (s *Store) UpdateServer(ctx context.Context, srv *models.Server) error {
	return s.db.Conn().WithContext(ctx).Save(srv).Error
}

// ── Audit Log ─────────────────────────────────────────────

// CreateAuditLog یک رکورد audit ایجاد می‌کند.
func (s *Store) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	return s.db.Conn().WithContext(ctx).Create(log).Error
}

// ListAuditLogs لیست audit log های یک کاربر.
func (s *Store) ListAuditLogs(ctx context.Context, actorID string, limit int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := s.db.Conn().WithContext(ctx).
		Where("actor_id = ?", actorID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// ListAdminAuditLogs همه audit log ها برای ادمین.
func (s *Store) ListAdminAuditLogs(ctx context.Context, action, targetType string, limit int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	q := s.db.Conn().WithContext(ctx).Order("created_at DESC").Limit(limit)
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if targetType != "" {
		q = q.Where("target_type = ?", targetType)
	}
	return logs, q.Find(&logs).Error
}

// ---- SourceWorkerConfig ----
// پیاده‌سازی سمتِ ذخیره‌سازی برای قرارداد source.worker.* (شرح کامل در
// shared-core/protocol/source_worker.go و مدل در shared-core/models).

func (s *Store) CreateSourceWorkerConfig(ctx context.Context, cfg *models.SourceWorkerConfig) error {
	return s.db.Conn().WithContext(ctx).Create(cfg).Error
}

func (s *Store) ListSourceWorkerConfigs(ctx context.Context) ([]models.SourceWorkerConfig, error) {
	var list []models.SourceWorkerConfig
	err := s.db.Conn().WithContext(ctx).Order("created_at DESC").Find(&list).Error
	return list, err
}

func (s *Store) FindSourceWorkerConfigByLicenseKey(ctx context.Context, licenseKey string) (*models.SourceWorkerConfig, error) {
	if licenseKey == "" {
		return nil, nil
	}
	var cfg models.SourceWorkerConfig
	err := s.db.Conn().WithContext(ctx).Where("license_key = ?", licenseKey).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &cfg, err
}

func (s *Store) FindSourceWorkerConfig(ctx context.Context, id any) (*models.SourceWorkerConfig, error) {
	var cfg models.SourceWorkerConfig
	err := s.db.Conn().WithContext(ctx).First(&cfg, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &cfg, err
}

func (s *Store) DeleteSourceWorkerConfig(ctx context.Context, id any) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.SourceWorkerConfig{}, "id = ?", id).Error
}

func (s *Store) SetSourceWorkerConfigActive(ctx context.Context, id any, active bool) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.SourceWorkerConfig{}).
		Where("id = ?", id).
		Update("is_active", active).Error
}

// ListUsersByActivePlan کاربرانی که همین الان یک اشتراکِ فعال از پلنِ
// داده‌شده دارند — برای broadcastِ فیلترشده به تفکیکِ پلن.
func (s *Store) ListUsersByActivePlan(ctx context.Context, planID string) ([]models.User, error) {
	if isBlankID(planID) {
		return nil, nil
	}
	var users []models.User
	err := s.db.Conn().WithContext(ctx).
		Joins("JOIN subscriptions ON subscriptions.user_id = users.id").
		Where("subscriptions.plan_id = ? AND subscriptions.is_active = ?", planID, true).
		Find(&users).Error
	return users, err
}

// ListUsersWithoutActivePlan کاربرانی که هیچ اشتراکِ فعالی ندارند — برای
// broadcastِ فیلترشده (مثلاً برای پیشنهادِ خرید پلن).
func (s *Store) ListUsersWithoutActivePlan(ctx context.Context) ([]models.User, error) {
	var users []models.User
	err := s.db.Conn().WithContext(ctx).
		Where("NOT EXISTS (SELECT 1 FROM subscriptions WHERE subscriptions.user_id = users.id AND subscriptions.is_active = true)").
		Find(&users).Error
	return users, err
}

// ---- PromoCode ----

func (s *Store) CreatePromoCode(ctx context.Context, p *models.PromoCode) error {
	return s.db.Conn().WithContext(ctx).Create(p).Error
}

func (s *Store) ListPromoCodes(ctx context.Context) ([]models.PromoCode, error) {
	var list []models.PromoCode
	err := s.db.Conn().WithContext(ctx).Order("created_at DESC").Find(&list).Error
	return list, err
}

func (s *Store) FindPromoCodeByCode(ctx context.Context, code string) (*models.PromoCode, error) {
	if code == "" {
		return nil, nil
	}
	var p models.PromoCode
	err := s.db.Conn().WithContext(ctx).Where("code = ?", code).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (s *Store) FindPromoCode(ctx context.Context, id any) (*models.PromoCode, error) {
	var p models.PromoCode
	err := s.db.Conn().WithContext(ctx).First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (s *Store) DeletePromoCode(ctx context.Context, id any) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.PromoCode{}, "id = ?", id).Error
}

func (s *Store) SetPromoCodeActive(ctx context.Context, id any, active bool) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.PromoCode{}).
		Where("id = ?", id).
		Update("is_active", active).Error
}

// RedeemPromoCode «claim» اتمیکِ یک redeem است: با قفلِ ردیف (SELECT ... FOR
// UPDATE) از race بین دو تلاشِ هم‌زمان (که هردو چکِ «هنوز تمام نشده» را رد
// کنند) جلوگیری می‌کند. این تابع فقط بخشِ DB (شمارنده + رکوردِ مصرف) را انجام
// می‌دهد؛ اعتبار واقعی در کیفِ پول (botpay/NATS) جداگانه توسط کالر اعطا
// می‌شود — عمداً *بعد* از claim موفق، چون در بدترین حالت (claim موفق ولی
// Credit ناموفق) کاربر پولی از دست نمی‌دهد، فقط باید دوباره تلاش/پیگیری کند؛
// ترتیبِ برعکس می‌توانست به دو بار اعتباردهی برای یک claim منجر شود.
func (s *Store) RedeemPromoCode(ctx context.Context, promoID, userID uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p models.PromoCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, "id = ?", promoID).Error; err != nil {
			return err
		}
		if !p.IsRedeemable() {
			return ErrPromoNotRedeemable
		}
		var count int64
		if err := tx.Model(&models.PromoRedemption{}).
			Where("promo_id = ? AND user_id = ?", promoID, userID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrPromoAlreadyRedeemed
		}
		if err := tx.Model(&models.PromoCode{}).
			Where("id = ?", promoID).
			Update("used_count", gorm.Expr("used_count + 1")).Error; err != nil {
			return err
		}
		return tx.Create(&models.PromoRedemption{PromoID: promoID, UserID: userID}).Error
	})
}

// UpdateSourceWorkerHeartbeat آخرین heartbeat یک worker را (با WorkerID، نه
// LicenseKey — worker بعد از register دیگر LicenseKey را با خودش حمل
// نمی‌کند) ثبت می‌کند. fire-and-forget است، پس نبودِ رکورد خطای مهمی نیست.
func (s *Store) UpdateSourceWorkerHeartbeat(ctx context.Context, workerID, status string) error {
	if workerID == "" {
		return nil
	}
	return s.db.Conn().WithContext(ctx).
		Model(&models.SourceWorkerConfig{}).
		Where("worker_id = ?", workerID).
		Updates(map[string]any{
			"last_heartbeat_at": timeNow(),
			"last_status":       status,
		}).Error
}
