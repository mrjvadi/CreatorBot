package admin

import (
	"context"

	tele "gopkg.in/telebot.v4"
)

// AdminUserAction اعمال action روی کاربر (از طریق callback).
func (h *Admin) AdminUserAction(ctx context.Context, c tele.Context, idStr, action string) error {
	defer func() { _ = c.Respond() }()
	return h.AdminUserHandleAction(ctx, c, idStr, action)
}
