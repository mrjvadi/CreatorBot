package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Handler) adminUsersList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	users, _ := h.store.ListUsers(ctx)

	if len(users) == 0 {
		return c.Send(h.t(ctx, uid, i18n.KeyUsersEmpty), h.kbAdmin(ctx, uid))
	}

	owners, admins, regular, blocked := 0, 0, 0, 0
	for _, u := range users {
		if u.IsBlocked { blocked++ }
		switch u.Role {
		case models.RoleOwner: owners++
		case models.RoleAdmin: admins++
		default:               regular++
		}
	}

	lines := []string{
		h.t(ctx, uid, i18n.KeyUsersTitle, len(users)), "",
		h.t(ctx, uid, i18n.KeyAdminUserSummary, owners, admins, regular, blocked), "",
	}
	for _, u := range users {
		lines = append(lines, fmtUser(u))
	}
	lines = append(lines, "", "TelegramID:")

	return c.Send(joinLines(lines), tele.ModeHTML, h.kbBack(ctx, uid))
}

func (h *Handler) adminUserDetail(ctx context.Context, c tele.Context, telegramID int64) error {
	uid := c.Sender().ID
	u, err := h.store.FindUserByTelegramID(ctx, telegramID)
	if err != nil || u == nil {
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyNotFound))
	}

	instances, _ := h.store.ListInstancesByOwner(ctx, u.ID)
	uname := ""
	if u.Username != "" { uname = " (@" + u.Username + ")" }
	blocked := "—"
	if u.IsBlocked { blocked = h.t(ctx, uid, i18n.KeyAdminUserBlocked) }

	text := fmt.Sprintf(
		h.t(ctx, uid, i18n.KeyAdminUserDetail),
		u.FirstName, uname, u.TelegramID, u.Role, blocked, len(instances),
	)

	h.setStep(ctx, uid, stepUserAction, "user_id", u.ID.String())
	return c.Send(text, tele.ModeHTML, h.kbUserActions(ctx, uid, u.TelegramID))
}

func (h *Handler) adminUserHandleAction(ctx context.Context, c tele.Context, userID, action string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	blockBtn   := h.btn(ctx, uid, i18n.KeyBtnBlock)
	unblockBtn := h.btn(ctx, uid, i18n.KeyBtnUnblock)
	adminBtn   := h.btn(ctx, uid, i18n.KeyBtnMakeAdmin)
	userBtn    := h.btn(ctx, uid, i18n.KeyBtnMakeUser)
	backBtn    := h.btn(ctx, uid, i18n.KeyBtnBack)

	switch action {
	case blockBtn:
		h.store.SetUserBlocked(ctx, userID, true)
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyUserBlocked))
	case unblockBtn:
		h.store.SetUserBlocked(ctx, userID, false)
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyUserUnblocked))
	case adminBtn:
		h.store.SetUserRole(ctx, userID, models.RoleAdmin)
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyUserMadeAdmin))
	case userBtn:
		h.store.SetUserRole(ctx, userID, models.RoleUser)
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyUserMadeUser))
	case backBtn:
		return h.adminUsersList(ctx, c)
	}

	// TelegramID وارد شده
	var tid int64
	if _, err := fmt.Sscanf(strings.TrimSpace(action), "%d", &tid); err == nil && tid > 0 {
		return h.adminUserDetail(ctx, c, tid)
	}

	return nil
}
