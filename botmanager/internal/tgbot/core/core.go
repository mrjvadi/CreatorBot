// Package core وابستگی‌های مشترک و helperهای پایه‌ی ربات را نگه می‌دارد.
// هندلرهای admin و user این Deps را در بر می‌گیرند (embed) و از متدهای آن
// استفاده می‌کنند. این‌طوری منطق هر بخش در package جدا قرار می‌گیرد.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/licenseclient"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	"github.com/mrjvadi/creatorbot/shared-core/ton"
	natsadapter "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Deps همه‌ی وابستگی‌های مشترک + helperهای پایه.
type Deps struct {
	Bot         *tele.Bot
	Store       *store.Store
	Cache       ports.Cache
	Docker      *sharedocker.Manager
	Log         ports.Logger
	OwnerID     int64
	BotUsername string
	EncryptKey  string
	Ton         *ton.Client
	Pay         *natspayclient.Client
	License     *licenseclient.Client
	Tr          *i18n.Translator
	NC          *natsadapter.Client
}

// New یک Deps می‌سازد.
func New(
	bot *tele.Bot,
	st *store.Store,
	cache ports.Cache,
	docker *sharedocker.Manager,
	log ports.Logger,
	ownerID int64,
	encryptKey string,
	tonClient *ton.Client,
	payClient *natspayclient.Client,
	nc *natsadapter.Client,
	licenseClient *licenseclient.Client,
) *Deps {
	return &Deps{
		Bot:         bot,
		Store:       st,
		Cache:       cache,
		Docker:      docker,
		Log:         log,
		OwnerID:     ownerID,
		BotUsername: bot.Me.Username,
		EncryptKey:  encryptKey,
		Ton:         tonClient,
		Pay:         payClient,
		License:     licenseClient,
		Tr:          i18n.New(cache),
		NC:          nc,
	}
}

// ── i18n ──────────────────────────────────────────────────

func (d *Deps) T(ctx context.Context, uid int64, key i18n.Key, args ...any) string {
	return d.Tr.T(ctx, uid, key, args...)
}

func (d *Deps) Btn(ctx context.Context, uid int64, key i18n.Key) string {
	return d.Tr.Btn(ctx, uid, key)
}

func (d *Deps) F(key string, val any) ports.Field { return ports.F(key, val) }

// TxStatusAlert یک پاسخِ callback (به‌صورت اعلان) برای وضعیتِ یک تراکنش/فاکتور
// می‌سازد. r=nil یعنی استعلام ناموفق بوده.
func (d *Deps) TxStatusAlert(ctx context.Context, uid int64, r *natspayclient.InvoiceStatusResult) *tele.CallbackResponse {
	var text string
	switch {
	case r == nil:
		text = d.T(ctx, uid, i18n.KeyTxCheckFailed)
	case r.Status == protocol.InvoiceStatusPaid:
		text = d.T(ctx, uid, i18n.KeyTxPaid)
	case r.Status == protocol.InvoiceStatusPartial:
		text = d.T(ctx, uid, i18n.KeyTxPartial, r.PaidTON, r.AmountTON)
	case r.Status == protocol.InvoiceStatusExpired:
		text = d.T(ctx, uid, i18n.KeyTxExpired)
	case r.Status == protocol.InvoiceStatusNotFound:
		text = d.T(ctx, uid, i18n.KeyTxNotFound)
	default: // pending یا نامشخص
		text = d.T(ctx, uid, i18n.KeyTxPending)
	}
	return &tele.CallbackResponse{Text: text, ShowAlert: true}
}

// ExtractBotID شناسه ربات را از توکن تلگرام (فرمت <id>:<secret>) استخراج می‌کند.
// مشترک بین wizard (user) و دپلوی تستی (admin).
func ExtractBotID(token string) (int64, error) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, fmt.Errorf("invalid token")
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid token")
	}
	return id, nil
}

var botTypeLabelKeys = map[models.BotType]i18n.Key{
	models.BotTypeUploader: i18n.KeyBotTypeUploader,
	models.BotTypeVPN:      i18n.KeyBotTypeVPN,
	models.BotTypeArchive:  i18n.KeyBotTypeArchive,
	models.BotTypeMember:   i18n.KeyBotTypeMember,
}

func (d *Deps) BotTypeLabel(ctx context.Context, uid int64, t models.BotType) string {
	if k, ok := botTypeLabelKeys[t]; ok {
		return d.T(ctx, uid, k)
	}
	return string(t)
}

// ── Auth ──────────────────────────────────────────────────

const ctxKeyUser = "bm:user"

func (d *Deps) LoadUser(ctx context.Context, c tele.Context) (*models.User, error) {
	if v := c.Get(ctxKeyUser); v != nil {
		u, _ := v.(*models.User)
		return u, nil
	}
	u, err := d.Store.FindUserByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		return nil, err
	}
	c.Set(ctxKeyUser, u)
	return u, nil
}

func (d *Deps) IsOwner(c tele.Context) bool { return c.Sender().ID == d.OwnerID }

func (d *Deps) IsAdmin(c tele.Context) bool {
	if c.Sender().ID == d.OwnerID {
		return true
	}
	u, _ := d.LoadUser(context.Background(), c)
	return u != nil && (u.Role == models.RoleAdmin || u.Role == models.RoleOwner)
}

func adminModeKey(uid int64) string { return fmt.Sprintf("bm:adminmode:%d", uid) }

func (d *Deps) IsInAdminMode(c tele.Context) bool {
	if !d.IsAdmin(c) {
		return false
	}
	val, err := d.Cache.Get(context.Background(), adminModeKey(c.Sender().ID))
	if err != nil || val == "" {
		return true
	}
	return val == "1"
}

func (d *Deps) SetAdminMode(ctx context.Context, uid int64, on bool) {
	val := "0"
	if on {
		val = "1"
	}
	_ = d.Cache.Set(ctx, adminModeKey(uid), val, 30*24*time.Hour)
}

func (d *Deps) GetOrCreateUser(ctx context.Context, c tele.Context) (*models.User, error) {
	u, err := d.LoadUser(ctx, c)
	if err != nil {
		return nil, err
	}
	if u != nil {
		return u, nil
	}
	role := models.RoleUser
	if c.Sender().ID == d.OwnerID {
		role = models.RoleOwner
	}
	u = &models.User{
		TelegramID: c.Sender().ID,
		Username:   c.Sender().Username,
		FirstName:  c.Sender().FirstName,
		Role:       role,
	}
	if err := d.Store.UpsertUser(ctx, u); err != nil {
		return u, err
	}
	c.Set(ctxKeyUser, u)
	return u, nil
}

// ── Audit ─────────────────────────────────────────────────

func (d *Deps) AuditLog(ctx context.Context, actorID uuid.UUID, actorRole,
	targetID, targetType string, action models.AuditAction, meta string) {
	log := &models.AuditLog{
		ActorID:     actorID,
		ActorRole:   actorRole,
		TargetID:    targetID,
		TargetType:  targetType,
		Action:      action,
		Description: meta,
	}
	if err := d.Store.CreateAuditLog(ctx, log); err != nil {
		d.Log.Error("audit log failed", d.F("err", err))
	}
}

// ── State machine (Redis) ─────────────────────────────────

const stateTTL = 15 * time.Minute

func stateKey(uid int64) string { return fmt.Sprintf("bm:s:%d", uid) }

func (d *Deps) GetState(ctx context.Context, uid int64) state.UserState {
	raw, err := d.Cache.Get(ctx, stateKey(uid))
	if err != nil || raw == "" {
		return state.UserState{Step: state.StepIdle, Data: map[string]string{}}
	}
	var s state.UserState
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return state.UserState{Step: state.StepIdle, Data: map[string]string{}}
	}
	if s.Data == nil {
		s.Data = map[string]string{}
	}
	return s
}

func (d *Deps) SetState(ctx context.Context, uid int64, s state.UserState) {
	data, _ := json.Marshal(s)
	_ = d.Cache.Set(ctx, stateKey(uid), string(data), stateTTL)
}

func (d *Deps) ClearState(ctx context.Context, uid int64) {
	_ = d.Cache.Del(ctx, stateKey(uid))
}

func (d *Deps) SetStep(ctx context.Context, uid int64, st state.Step, kv ...string) {
	s := d.GetState(ctx, uid)
	s.Step = st
	if s.Data == nil {
		s.Data = map[string]string{}
	}
	for i := 0; i+1 < len(kv); i += 2 {
		s.Data[kv[i]] = kv[i+1]
	}
	d.SetState(ctx, uid, s)
}

// ── Shared keyboards ──────────────────────────────────────

func (d *Deps) B(ctx context.Context, uid int64, k i18n.Key) tele.Btn {
	kb := &tele.ReplyMarkup{}
	return kb.Text(d.Btn(ctx, uid, k))
}

func (d *Deps) KbUser(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(d.B(ctx, uid, i18n.KeyMenuMyBots), d.B(ctx, uid, i18n.KeyMenuCreateBot)),
		kb.Row(d.B(ctx, uid, i18n.KeyMenuWallet), d.B(ctx, uid, i18n.KeyMenuPlans)),
		kb.Row(d.B(ctx, uid, i18n.KeyMenuSettings), d.B(ctx, uid, i18n.KeyMenuSupport)),
		kb.Row(d.B(ctx, uid, i18n.KeyMenuLanguage)),
	)
	return kb
}

func (d *Deps) KbUserFull(ctx context.Context, uid int64, _ *models.Subscription) *tele.ReplyMarkup {
	return d.KbUser(ctx, uid)
}

func (d *Deps) KbAdmin(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(d.B(ctx, uid, i18n.KeyMenuUsers), d.B(ctx, uid, i18n.KeyMenuBots)),
		kb.Row(d.B(ctx, uid, i18n.KeyMenuPlans), d.B(ctx, uid, i18n.KeyMenuTemplates)),
		kb.Row(d.B(ctx, uid, i18n.KeyMenuServers), d.B(ctx, uid, i18n.KeyMenuStats)),
		kb.Row(d.B(ctx, uid, i18n.KeyMenuBroadcast), d.B(ctx, uid, i18n.KeyMenuSystem)),
		kb.Row(d.B(ctx, uid, i18n.KeyMenuExitAdmin)),
	)
	return kb
}

func (d *Deps) KbUserActions(ctx context.Context, uid int64, targetID int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data(d.Btn(ctx, uid, i18n.KeyBtnBlock), fmt.Sprintf("block_user:%d", targetID)),
			kb.Data(d.Btn(ctx, uid, i18n.KeyBtnUnblock), fmt.Sprintf("unblock_user:%d", targetID)),
		),
		kb.Row(
			kb.Data(d.Btn(ctx, uid, i18n.KeyBtnMakeAdmin), fmt.Sprintf("make_admin:%d", targetID)),
			kb.Data(d.Btn(ctx, uid, i18n.KeyBtnMakeUser), fmt.Sprintf("make_user:%d", targetID)),
		),
		kb.Row(kb.Data(d.Btn(ctx, uid, i18n.KeyBtnAddCredit), fmt.Sprintf("add_credit:%d", targetID))),
		kb.Row(kb.Data(d.Btn(ctx, uid, i18n.KeyBtnBackToList), "admin_users")),
	)
	return kb
}

func (d *Deps) KbBack(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data(d.Btn(ctx, uid, i18n.KeyBtnBack), "cancel")))
	return kb
}

func (d *Deps) KbBackCancel(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data(d.Btn(ctx, uid, i18n.KeyBtnBack), "cancel"),
			kb.Data(d.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel"),
		),
	)
	return kb
}

func (d *Deps) KbCancel(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data(d.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel")))
	return kb
}

// KbLanguage کیبورد انتخاب زبان (نام زبان‌ها endonym و ترجمه‌ناپذیر).
func KbLanguage() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("🇮🇷 فارسی", "lang:fa"),
			kb.Data("🇬🇧 English", "lang:en"),
		),
	)
	return kb
}

// SendMain منوی اصلیِ مناسبِ نقش را می‌فرستد.
func (d *Deps) SendMain(c tele.Context, text string) error {
	ctx := context.Background()
	uid := c.Sender().ID
	if d.IsInAdminMode(c) {
		return c.Send(text, d.KbAdmin(ctx, uid))
	}
	return c.Send(text, d.KbUser(ctx, uid))
}

// IsCancel بررسی می‌کند متن دکمه‌ی لغو/بازگشت یا /cancel است.
func (d *Deps) IsCancel(ctx context.Context, uid int64, text string) bool {
	return text == d.Btn(ctx, uid, i18n.KeyBtnCancel) ||
		text == d.Btn(ctx, uid, i18n.KeyBtnBack) ||
		text == "/cancel"
}
