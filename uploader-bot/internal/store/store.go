// Package store — uploader-bot repository.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

type Store struct{ db ports.DB }

func New(db ports.DB) *Store { return &Store{db: db} }

// ── User ─────────────────────────────────────────────────────

func (s *Store) GetOrCreateUser(ctx context.Context, tgID int64, username, firstName string) (*models.User, error) {
	var u models.User
	err := s.db.Conn().WithContext(ctx).
		Where(models.User{TelegramID: tgID}).
		Attrs(models.User{Username: username, FirstName: firstName}).
		FirstOrCreate(&u).Error
	return &u, err
}

func (s *Store) GetUser(ctx context.Context, tgID int64) (*models.User, error) {
	var u models.User
	err := s.db.Conn().WithContext(ctx).Where("telegram_id = ?", tgID).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (s *Store) UpdateUser(ctx context.Context, u *models.User) error {
	return s.db.Conn().WithContext(ctx).Save(u).Error
}

func (s *Store) BlockUser(ctx context.Context, tgID int64, block bool) error {
	return s.db.Conn().WithContext(ctx).Model(&models.User{}).
		Where("telegram_id = ?", tgID).
		Update("is_blocked", block).Error
}

func (s *Store) SetUserSub(ctx context.Context, tgID int64, planID uuid.UUID, days int) error {
	exp := time.Now().AddDate(0, 0, days)
	return s.db.Conn().WithContext(ctx).Model(&models.User{}).
		Where("telegram_id = ?", tgID).
		Updates(map[string]any{"sub_plan_id": planID, "sub_expires_at": exp}).Error
}

func (s *Store) ResetDownloadCounts(ctx context.Context) error {
	return s.db.Conn().WithContext(ctx).
		Model(&models.User{}).Where("1=1").
		Update("free_downloads", 0).Error
}

func (s *Store) SearchUser(ctx context.Context, query string) (*models.User, error) {
	var u models.User
	err := s.db.Conn().WithContext(ctx).
		Where("telegram_id::text = ? OR username ILIKE ?", query, "%"+query+"%").
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (s *Store) ListUsers(ctx context.Context, page, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	s.db.Conn().WithContext(ctx).Model(&models.User{}).Count(&total)
	err := s.db.Conn().WithContext(ctx).
		Order("created_at DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&users).Error
	return users, total, err
}

// ── Code ─────────────────────────────────────────────────────

func (s *Store) CreateCode(ctx context.Context, c *models.Code) error {
	return s.db.Conn().WithContext(ctx).Create(c).Error
}

func (s *Store) FindCode(ctx context.Context, code string) (*models.Code, error) {
	var c models.Code
	err := s.db.Conn().WithContext(ctx).
		Preload("Files").
		Where("code = ?", code).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (s *Store) FindCodeByID(ctx context.Context, id uuid.UUID) (*models.Code, error) {
	var c models.Code
	err := s.db.Conn().WithContext(ctx).
		Preload("Files").First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (s *Store) UpdateCode(ctx context.Context, c *models.Code) error {
	return s.db.Conn().WithContext(ctx).Save(c).Error
}

func (s *Store) DeleteCode(ctx context.Context, id uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.Code{}, id).Error
}

func (s *Store) IncrementCodeUse(ctx context.Context, id uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).Model(&models.Code{}).
		Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

func (s *Store) ListCodes(ctx context.Context, folderID *uuid.UUID, page, limit int) ([]models.Code, int64, error) {
	var codes []models.Code
	var total int64
	q := s.db.Conn().WithContext(ctx).Model(&models.Code{})
	if folderID != nil {
		q = q.Where("folder_id = ?", *folderID)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&codes).Error
	return codes, total, err
}

func (s *Store) SearchCodes(ctx context.Context, query string) ([]models.Code, error) {
	var codes []models.Code
	err := s.db.Conn().WithContext(ctx).
		Where("code ILIKE ? OR caption ILIKE ?", "%"+query+"%", "%"+query+"%").
		Limit(20).Find(&codes).Error
	return codes, err
}

func (s *Store) CodeExists(ctx context.Context, code string) bool {
	var count int64
	s.db.Conn().WithContext(ctx).Model(&models.Code{}).Where("code = ?", code).Count(&count)
	return count > 0
}

// ── File ─────────────────────────────────────────────────────

func (s *Store) CreateFile(ctx context.Context, f *models.File) error {
	return s.db.Conn().WithContext(ctx).Create(f).Error
}

func (s *Store) AddFileToCode(ctx context.Context, codeID, fileID uuid.UUID, order int) error {
	return s.db.Conn().WithContext(ctx).Create(&models.CodeFile{
		CodeID: codeID, FileID: fileID, Order: order,
	}).Error
}

func (s *Store) RemoveFileFromCode(ctx context.Context, codeID, fileID uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).
		Where("code_id = ? AND file_id = ?", codeID, fileID).
		Delete(&models.CodeFile{}).Error
}

func (s *Store) GetFilesForCode(ctx context.Context, codeID uuid.UUID) ([]models.File, error) {
	var files []models.File
	err := s.db.Conn().WithContext(ctx).
		Joins("JOIN code_files ON code_files.file_id = files.id").
		Where("code_files.code_id = ?", codeID).
		Order("code_files.order ASC").
		Find(&files).Error
	return files, err
}

// ── Folder ────────────────────────────────────────────────────

func (s *Store) CreateFolder(ctx context.Context, f *models.Folder) error {
	return s.db.Conn().WithContext(ctx).Create(f).Error
}

func (s *Store) ListFolders(ctx context.Context, parentID *uuid.UUID) ([]models.Folder, error) {
	var folders []models.Folder
	q := s.db.Conn().WithContext(ctx).Where("is_active = true")
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}
	err := q.Order("sort_order ASC, name ASC").Find(&folders).Error
	return folders, err
}

func (s *Store) DeleteFolder(ctx context.Context, id uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.Folder{}, id).Error
}

// ── Force Join Channel ────────────────────────────────────────

func (s *Store) AddForceJoinChannel(ctx context.Context, ch *models.ForceJoinChannel) error {
	return s.db.Conn().WithContext(ctx).Create(ch).Error
}

func (s *Store) ListForceJoinChannels(ctx context.Context) ([]models.ForceJoinChannel, error) {
	var chs []models.ForceJoinChannel
	err := s.db.Conn().WithContext(ctx).
		Where("is_active = true").Order("sort_order ASC").Find(&chs).Error
	return chs, err
}

func (s *Store) RemoveForceJoinChannel(ctx context.Context, id uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.ForceJoinChannel{}, id).Error
}

// ── Sub Plans ─────────────────────────────────────────────────

func (s *Store) CreateSubPlan(ctx context.Context, p *models.SubPlan) error {
	return s.db.Conn().WithContext(ctx).Create(p).Error
}

func (s *Store) ListSubPlans(ctx context.Context) ([]models.SubPlan, error) {
	var plans []models.SubPlan
	err := s.db.Conn().WithContext(ctx).
		Where("is_active = true").Order("sort_order ASC, price ASC").Find(&plans).Error
	return plans, err
}

func (s *Store) DeleteSubPlan(ctx context.Context, id uuid.UUID) error {
	return s.db.Conn().WithContext(ctx).Delete(&models.SubPlan{}, id).Error
}

// ── Payment ───────────────────────────────────────────────────

func (s *Store) CreatePayment(ctx context.Context, p *models.Payment) error {
	return s.db.Conn().WithContext(ctx).Create(p).Error
}

func (s *Store) ConfirmPayment(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return s.db.Conn().WithContext(ctx).Model(&models.Payment{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": models.PaymentConfirmed, "confirmed_at": now}).Error
}

func (s *Store) FindPaymentByAuthority(ctx context.Context, authority string) (*models.Payment, error) {
	var p models.Payment
	err := s.db.Conn().WithContext(ctx).Where("authority = ?", authority).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

// ── Download Log ──────────────────────────────────────────────

func (s *Store) LogDownload(ctx context.Context, userID, codeID uuid.UUID) error {
	var log models.DownloadLog
	err := s.db.Conn().WithContext(ctx).
		Where("user_id = ? AND code_id = ?", userID, codeID).First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.db.Conn().WithContext(ctx).Create(&models.DownloadLog{
			UserID: userID, CodeID: codeID, Count: 1,
		}).Error
	}
	return s.db.Conn().WithContext(ctx).Model(&log).
		UpdateColumn("count", gorm.Expr("count + 1")).Error
}

func (s *Store) GetDownloadCount(ctx context.Context, userID, codeID uuid.UUID) int {
	var log models.DownloadLog
	s.db.Conn().WithContext(ctx).
		Where("user_id = ? AND code_id = ?", userID, codeID).First(&log)
	return log.Count
}

// ── Settings ──────────────────────────────────────────────────

func (s *Store) GetSetting(ctx context.Context, key string) string {
	var st models.Setting
	s.db.Conn().WithContext(ctx).Where("key = ?", key).First(&st)
	return st.Value
}

func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	return s.db.Conn().WithContext(ctx).
		Where(models.Setting{Key: key}).
		Assign(models.Setting{Value: value}).
		FirstOrCreate(&models.Setting{}).Error
}

func (s *Store) GetAllSettings(ctx context.Context) map[string]string {
	var settings []models.Setting
	s.db.Conn().WithContext(ctx).Find(&settings)
	result := make(map[string]string, len(settings))
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result
}

// ── Backup ────────────────────────────────────────────────────

func (s *Store) CreateBackup(ctx context.Context, b *models.Backup) error {
	return s.db.Conn().WithContext(ctx).Create(b).Error
}

func (s *Store) ListBackups(ctx context.Context, limit int) ([]models.Backup, error) {
	var backups []models.Backup
	err := s.db.Conn().WithContext(ctx).
		Order("created_at DESC").Limit(limit).Find(&backups).Error
	return backups, err
}

// ── Stats ─────────────────────────────────────────────────────

type Stats struct {
	TotalUsers int64
	TotalCodes int64
	TotalFiles int64
	TodayUsers int64
	ActiveSubs int64
}

func (s *Store) GetStats(ctx context.Context) Stats {
	var st Stats
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	s.db.Conn().WithContext(ctx).Model(&models.User{}).Count(&st.TotalUsers)
	s.db.Conn().WithContext(ctx).Model(&models.Code{}).Count(&st.TotalCodes)
	s.db.Conn().WithContext(ctx).Model(&models.File{}).Count(&st.TotalFiles)
	s.db.Conn().WithContext(ctx).Model(&models.User{}).
		Where("created_at >= ?", today).Count(&st.TodayUsers)
	s.db.Conn().WithContext(ctx).Model(&models.User{}).
		Where("sub_expires_at > ?", now).Count(&st.ActiveSubs)
	return st
}

// ── Admin ─────────────────────────────────────────────────────

func (s *Store) IsAdmin(ctx context.Context, tgID int64) bool {
	var count int64
	s.db.Conn().WithContext(ctx).Model(&models.Admin{}).
		Where("telegram_id = ?", tgID).Count(&count)
	return count > 0
}

func (s *Store) AddAdmin(ctx context.Context, tgID int64, username string) error {
	return s.db.Conn().WithContext(ctx).Create(&models.Admin{
		TelegramID: tgID, Username: username,
	}).Error
}

func (s *Store) RemoveAdmin(ctx context.Context, tgID int64) error {
	return s.db.Conn().WithContext(ctx).
		Where("telegram_id = ? AND is_owner = false", tgID).
		Delete(&models.Admin{}).Error
}

func (s *Store) ListAdmins(ctx context.Context) ([]models.Admin, error) {
	var admins []models.Admin
	err := s.db.Conn().WithContext(ctx).Find(&admins).Error
	return admins, err
}

// ── Preview Channel ───────────────────────────────────────────

func (s *Store) AddPreviewChannel(ctx context.Context, ch *models.PreviewChannel) error {
	return s.db.Conn().WithContext(ctx).Create(ch).Error
}

func (s *Store) ListPreviewChannels(ctx context.Context) ([]models.PreviewChannel, error) {
	var chs []models.PreviewChannel
	err := s.db.Conn().WithContext(ctx).Where("is_active = true").Find(&chs).Error
	return chs, err
}

// ── Helpers ───────────────────────────────────────────────────

// GenerateUniqueCode یک کد یکتای ۸ کاراکتری تولید می‌کند.
func (s *Store) GenerateUniqueCode(ctx context.Context) string {
	for {
		code := fmt.Sprintf("%08d", time.Now().UnixNano()%100_000_000)
		if !s.CodeExists(ctx, code) {
			return code
		}
	}
}
