// Package licensing منطق صدور/بررسی/ابطال لایسنس هر instance را پیاده‌سازی می‌کند.
package licensing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/mrjvadi/creatorbot/license-service/internal/store"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Service منطق کسب‌وکار لایسنس.
type Service struct {
	store *store.Store
	nc    *natsclient.Client
	log   ports.Logger

	// signingSecret راز امضای JWT توکن لایسنس — مستقل از ENCRYPTION_KEY و
	// SERVICE_HMAC_SECRET پلتفرم تا نشتِ هرکدام باعثِ جعلِ لایسنس نشود.
	signingSecret string
	tokenTTL      time.Duration
}

func New(st *store.Store, nc *natsclient.Client, log ports.Logger, signingSecret string, tokenTTL time.Duration) *Service {
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour * 365 * 10 // عملاً «بدون انقضا»؛ ابطال واقعی با license.revoke
	}
	return &Service{store: st, nc: nc, log: log, signingSecret: signingSecret, tokenTTL: tokenTTL}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Issue یک لایسنس تازه برای instance صادر می‌کند (idempotent — اگر از قبل
// برای همین BotID لایسنس فعالی باشد، همان را بازتولید/بازگرداند به‌جای
// ساختن رکورد تکراری، چون AutoMigrate یک uniqueIndex روی bot_id دارد).
func (s *Service) Issue(ctx context.Context, req protocol.LicenseIssueRequest) (string, error) {
	subject := fmt.Sprintf("bot_%d", req.BotID)
	token, err := auth.GenerateAccessToken(subject, "license", auth.JWTConfig{
		AccessSecret: s.signingSecret,
		AccessTTL:    s.tokenTTL,
	})
	if err != nil {
		return "", fmt.Errorf("sign license token: %w", err)
	}

	existing, err := s.store.FindByBotID(ctx, req.BotID)
	if err != nil {
		return "", err
	}

	var expiresAt *time.Time
	if req.ExpiresAt > 0 {
		t := time.Unix(req.ExpiresAt, 0)
		expiresAt = &t
	}

	if existing != nil {
		existing.InstanceID = req.InstanceID
		existing.OwnerID = req.OwnerID
		existing.PlanID = req.PlanID
		existing.TokenHash = hashToken(token)
		existing.KnownServerID = req.ServerID
		existing.Status = "active"
		existing.RevokedReason = ""
		existing.ExpiresAt = expiresAt
		existing.CloneFlagCount = 0
		if err := s.store.Save(ctx, existing); err != nil {
			return "", err
		}
		s.log.Info("license re-issued", ports.F("bot_id", req.BotID), ports.F("server_id", req.ServerID))
		return token, nil
	}

	lic := &store.License{
		BotID:         req.BotID,
		InstanceID:    req.InstanceID,
		OwnerID:       req.OwnerID,
		PlanID:        req.PlanID,
		TokenHash:     hashToken(token),
		KnownServerID: req.ServerID,
		Status:        "active",
		ExpiresAt:     expiresAt,
	}
	if err := s.store.Create(ctx, lic); err != nil {
		return "", err
	}
	s.log.Info("license issued", ports.F("bot_id", req.BotID), ports.F("server_id", req.ServerID))
	return token, nil
}

// Verify بررسی می‌کند لایسنس یک instance معتبر است، و اگر check-in از
// ServerID غیرمنتظره باشد یک clone-warning برمی‌گرداند (fail-open — لایسنس
// باطل نمی‌شود، فقط پرچم می‌خورد و رویداد license.clone_detected منتشر
// می‌شود تا botmanager بتواند به ادمین/مالک اطلاع دهد).
func (s *Service) Verify(ctx context.Context, req protocol.LicenseVerifyRequest) (bool, string, bool, error) {
	lic, err := s.store.FindByBotID(ctx, req.BotID)
	if err != nil {
		return false, "", false, err
	}
	if lic == nil {
		return false, string(protocol.LicenseExpired), false, errors.New("license not found")
	}
	// اعتبار امضای توکن — جلوی probing با فقط دانستن bot_id را می‌گیرد؛
	// این توکن باید دقیقاً همان چیزی باشد که در Issue صادر شده.
	if hashToken(req.Token) != lic.TokenHash {
		return false, lic.Status, false, errors.New("token mismatch")
	}

	now := time.Now()
	lic.LastCheckinAt = &now
	lic.LastServerSeen = req.ServerID

	cloneWarning := false
	if lic.KnownServerID != "" && req.ServerID != "" && lic.KnownServerID != req.ServerID {
		cloneWarning = true
		lic.CloneFlagCount++
		s.log.Warn("license clone check-in detected",
			ports.F("bot_id", req.BotID),
			ports.F("known_server", lic.KnownServerID),
			ports.F("checkin_server", req.ServerID),
			ports.F("flag_count", lic.CloneFlagCount))
		if s.nc != nil {
			_ = s.nc.PublishCore(protocol.SubjLicenseCloneDetected, protocol.LicenseCloneDetectedEvent{
				BotID:            req.BotID,
				InstanceID:       lic.InstanceID,
				KnownServerID:    lic.KnownServerID,
				UnexpectedServer: req.ServerID,
				DetectedAt:       now.Unix(),
			})
		}
	}
	_ = s.store.Save(ctx, lic)

	valid := lic.IsActive()
	status := lic.Status
	if lic.ExpiresAt != nil && now.After(*lic.ExpiresAt) {
		status = string(protocol.LicenseExpired)
	}
	return valid, status, cloneWarning, nil
}

// Revoke لایسنس یک instance را باطل می‌کند.
func (s *Service) Revoke(ctx context.Context, botID int64, reason string) error {
	return s.store.Revoke(ctx, botID, reason)
}
