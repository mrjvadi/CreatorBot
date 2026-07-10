package admin

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/format"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Admin) AdminUsersList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	users, _ := h.Store.ListUsers(ctx)

	if len(users) == 0 {
		return c.Send(h.T(ctx, uid, i18n.KeyUsersEmpty), h.KbAdmin(ctx, uid))
	}

	owners, admins, regular, blocked := 0, 0, 0, 0
	for _, u := range users {
		if u.IsBlocked {
			blocked++
		}
		switch u.Role {
		case models.RoleOwner:
			owners++
		case models.RoleAdmin:
			admins++
		default:
			regular++
		}
	}

	lines := []string{
		h.T(ctx, uid, i18n.KeyUsersTitle, len(users)), "",
		h.T(ctx, uid, i18n.KeyAdminUserSummary, owners, admins, regular, blocked), "",
	}
	for _, u := range users {
		lines = append(lines, format.FmtUser(u))
	}
	lines = append(lines, "", h.T(ctx, uid, i18n.KeyUsersSearchPrompt))

	// قبلاً اینجا هیچ state‌ای فعال نمی‌شد، پس تایپِ یک TelegramID بعد از این
	// پیام به هیچ‌جا نمی‌رفت (state idle می‌ماند). حالا واقعاً منتظرِ ورودی می‌مانیم.
	h.SetStep(ctx, uid, state.StepAdminUserSearch)
	return c.Send(format.JoinLines(lines), tele.ModeHTML, h.KbBack(ctx, uid))
}

func (h *Admin) AdminUserDetail(ctx context.Context, c tele.Context, telegramID int64) error {
	uid := c.Sender().ID
	u, err := h.Store.FindUserByTelegramID(ctx, telegramID)
	if err != nil || u == nil {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyNotFound))
	}

	instances, _ := h.Store.ListInstancesByOwner(ctx, u.ID)
	uname := ""
	if u.Username != "" {
		uname = " (@" + u.Username + ")"
	}
	blocked := "—"
	if u.IsBlocked {
		blocked = h.T(ctx, uid, i18n.KeyAdminUserBlocked)
	}

	text := fmt.Sprintf(
		h.T(ctx, uid, i18n.KeyAdminUserDetail),
		u.FirstName, uname, u.TelegramID, u.Role, blocked, len(instances),
	)

	h.SetStep(ctx, uid, state.StepUserAction, "user_id", u.ID.String())
	return c.Send(text, tele.ModeHTML, h.KbUserActions(ctx, uid, u.TelegramID))
}

// AdminUserSearchSubmit ورودیِ متنیِ TelegramID را که بعد از AdminUsersList
// تایپ می‌شود پردازش می‌کند.
func (h *Admin) AdminUserSearchSubmit(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	var tid int64
	if _, err := fmt.Sscanf(strings.TrimSpace(text), "%d", &tid); err != nil || tid <= 0 {
		// دوباره منتظرِ ورودیِ درست می‌مانیم به‌جای پاک‌کردنِ بی‌سروصدای state.
		h.SetStep(ctx, uid, state.StepAdminUserSearch)
		return c.Send(h.T(ctx, uid, i18n.KeyUsersSearchInvalid), h.KbBackCancel(ctx, uid))
	}
	h.ClearState(ctx, uid)
	return h.AdminUserDetail(ctx, c, tid)
}

func (h *Admin) AdminUserHandleAction(ctx context.Context, c tele.Context, userID, action string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)

	blockBtn := h.Btn(ctx, uid, i18n.KeyBtnBlock)
	unblockBtn := h.Btn(ctx, uid, i18n.KeyBtnUnblock)
	adminBtn := h.Btn(ctx, uid, i18n.KeyBtnMakeAdmin)
	userBtn := h.Btn(ctx, uid, i18n.KeyBtnMakeUser)
	backBtn := h.Btn(ctx, uid, i18n.KeyBtnBack)

	switch action {
	case blockBtn:
		_ = h.Store.SetUserBlocked(ctx, userID, true)
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyUserBlocked))
	case unblockBtn:
		_ = h.Store.SetUserBlocked(ctx, userID, false)
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyUserUnblocked))
	case adminBtn:
		_ = h.Store.SetUserRole(ctx, userID, models.RoleAdmin)
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyUserMadeAdmin))
	case userBtn:
		_ = h.Store.SetUserRole(ctx, userID, models.RoleUser)
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyUserMadeUser))
	case backBtn:
		return h.AdminUsersList(ctx, c)
	}

	// TelegramID وارد شده
	var tid int64
	if _, err := fmt.Sscanf(strings.TrimSpace(action), "%d", &tid); err == nil && tid > 0 {
		return h.AdminUserDetail(ctx, c, tid)
	}

	return nil
}
