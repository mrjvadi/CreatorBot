package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/google/uuid"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/models"
	vpnports "github.com/mrjvadi/creatorbot/vpn-bot/internal/vpn"
)

// ════════════════════════════════════════════════════════════
// step های مربوط به پنل (در state.go تعریف می‌شن)
// ════════════════════════════════════════════════════════════
// stepAddPanelType  ← نوع پنل
// stepAddPanelURL   ← آدرس پنل
// stepAddPanelUser  ← یوزرنیم
// stepAddPanelPass  ← پسورد
// stepAddPanelCap   ← ظرفیت (0 = نامحدود)
// ════════════════════════════════════════════════════════════

func (h *Handler) adminPanels(ctx context.Context, c tele.Context) error {
	panels, err := h.store.ListPanels(ctx)
	if err != nil {
		return c.Send("❌ خطا در دریافت پنل‌ها.")
	}

	if len(panels) == 0 {
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data("➕ افزودن پنل", "panel_add")))
		return c.Send(
			"<b>🖥 پنل‌های VPN</b>\n\nهیچ پنلی وجود ندارد.\nبرای افزودن پنل کلیک کنید:",
			tele.ModeHTML, kb,
		)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🖥 پنل‌ها (%d)</b>\n\n", len(panels)))
	for _, p := range panels {
		active := "✅"
		if !p.IsActive {
			active = "❌"
		}
		cap := "نامحدود"
		if p.Capacity > 0 {
			cap = fmt.Sprintf("%d/%d", p.ActiveCount, p.Capacity)
		}
		sb.WriteString(fmt.Sprintf(
			"%s <b>%s</b> [%s]\n   🔗 %s\n   👥 %s\n   🆔 <code>%s</code>\n\n",
			active, p.Name, p.Type, p.BaseURL, cap, p.ID,
		))
	}

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("➕ افزودن پنل", "panel_add")),
		kb.Row(kb.Data("🔄 تست همه پنل‌ها", "panel_test_all")),
	)
	return c.Send(sb.String(), tele.ModeHTML, kb)
}

// ── add panel wizard ──────────────────────────────────────

func (h *Handler) startAddPanel(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	h.setStep(ctx, uid, stepAddPanelType)

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("Marzban", "ptype:marzban")),
		kb.Row(kb.Data("MarzNeshin", "ptype:marzneshin")),
		kb.Row(kb.Data("Hiddify", "ptype:hiddify")),
		kb.Row(kb.Data("X-UI", "ptype:xui")),
		kb.Row(kb.Data(btnCancel, "cancel")),
	)
	return c.Edit("<b>➕ پنل جدید</b>\n\nنوع پنل را انتخاب کنید:", tele.ModeHTML, kb)
}

func (h *Handler) handlePanelType(ctx context.Context, c tele.Context, ptype string) error {
	uid := c.Sender().ID
	defer c.Respond()
	h.setStep(ctx, uid, stepAddPanelURL, "ptype", ptype)
	return c.Edit(
		fmt.Sprintf("نوع: <b>%s</b>\n\nآدرس پنل را وارد کنید:\nمثال: <code>http://1.2.3.4:8080</code>", ptype),
		tele.ModeHTML,
	)
}

func (h *Handler) handlePanelURL(ctx context.Context, c tele.Context, st wizardState, url string) error {
	uid := c.Sender().ID
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	if !strings.HasPrefix(url, "http") {
		return c.Send("❌ آدرس باید با http یا https شروع شود.")
	}
	h.setStep(ctx, uid, stepAddPanelUser,
		"ptype", st.Data["ptype"],
		"url", url,
	)
	return c.Send("نام کاربری ادمین پنل:", kbCancel())
}

func (h *Handler) handlePanelUser(ctx context.Context, c tele.Context, st wizardState, username string) error {
	uid := c.Sender().ID
	h.setStep(ctx, uid, stepAddPanelPass,
		"ptype", st.Data["ptype"],
		"url", st.Data["url"],
		"user", username,
	)
	return c.Send("رمز عبور ادمین پنل:", kbCancel())
}

func (h *Handler) handlePanelPass(ctx context.Context, c tele.Context, st wizardState, pass string) error {
	uid := c.Sender().ID
	h.setStep(ctx, uid, stepAddPanelCap,
		"ptype", st.Data["ptype"],
		"url", st.Data["url"],
		"user", st.Data["user"],
		"pass", pass,
	)
	return c.Send(
		"ظرفیت پنل (تعداد اکانت):\n<code>0</code> = نامحدود",
		tele.ModeHTML, kbCancel(),
	)
}

func (h *Handler) handlePanelCap(ctx context.Context, c tele.Context, st wizardState, capStr string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	cap := 0
	fmt.Sscanf(strings.TrimSpace(capStr), "%d", &cap)

	// رمزنگاری پسورد
	encPass, err := auth.Encrypt(st.Data["pass"], h.encryptKey)
	if err != nil {
		return c.Send("❌ خطا در رمزنگاری.")
	}

	panel := &models.Panel{
		Name:      fmt.Sprintf("%s-%s", st.Data["ptype"], st.Data["url"][7:14]),
		Type:      st.Data["ptype"],
		BaseURL:   st.Data["url"],
		Username:  st.Data["user"],
		Password:  encPass,
		Capacity:  cap,
		IsActive:  true,
	}

	// تست اتصال قبل از ذخیره
	testPanel, err := vpnports.NewPanel(st.Data["ptype"], st.Data["url"], st.Data["user"], st.Data["pass"], "")
	if err != nil {
		return c.Send("❌ نوع پنل پشتیبانی نمی‌شود.")
	}

	testCtx, cancel := context.WithTimeout(ctx, 10*1e9)
	defer cancel()
	if err := testPanel.Login(testCtx); err != nil {
		return c.Send(fmt.Sprintf(
			"❌ اتصال به پنل ناموفق:\n<code>%s</code>\n\nبررسی کنید:\n• آدرس درست است؟\n• پنل روشن است؟\n• یوزر/پس درست است؟",
			err.Error(),
		), tele.ModeHTML, kbAdminMain())
	}

	if err := h.store.CreatePanel(ctx, panel); err != nil {
		h.log.Error("createPanel", ports.F("err", err))
		return c.Send("❌ خطا در ذخیره پنل.")
	}

	capText := "نامحدود"
	if cap > 0 {
		capText = fmt.Sprintf("%d اکانت", cap)
	}

	return c.Send(
		fmt.Sprintf(
			"✅ <b>پنل اضافه شد</b>\n\n"+
				"🔗 %s\n"+
				"📦 نوع: %s\n"+
				"👥 ظرفیت: %s\n"+
				"🆔 <code>%s</code>",
			panel.BaseURL, panel.Type, capText, panel.ID,
		),
		tele.ModeHTML, kbAdminMain(),
	)
}

// ── panel actions ─────────────────────────────────────────

func (h *Handler) testAllPanels(ctx context.Context, c tele.Context) error {
	defer c.Respond()
	panels, _ := h.store.ListPanels(ctx)
	if len(panels) == 0 {
		return c.Edit("هیچ پنلی وجود ندارد.")
	}

	var sb strings.Builder
	sb.WriteString("<b>🔄 نتیجه تست پنل‌ها</b>\n\n")

	for _, p := range panels {
		pass, _ := auth.Decrypt(p.Password, h.encryptKey)
		testPanel, err := vpnports.NewPanel(p.Type, p.BaseURL, p.Username, pass, "")
		status := "✅"
		detail := ""
		if err != nil {
			status = "❌"
			detail = err.Error()
		} else {
			testCtx, cancel := context.WithTimeout(ctx, 5*1e9)
			if err := testPanel.Login(testCtx); err != nil {
				status = "❌"
				detail = err.Error()
			}
			cancel()
		}
		sb.WriteString(fmt.Sprintf("%s <b>%s</b>", status, p.Name))
		if detail != "" {
			sb.WriteString(fmt.Sprintf("\n   <i>%s</i>", detail[:min(50, len(detail))]))
		}
		sb.WriteString("\n")
	}

	return c.Edit(sb.String(), tele.ModeHTML)
}

func (h *Handler) togglePanel(ctx context.Context, c tele.Context, panelIDStr string) error {
	defer c.Respond()
	panelID, _ := uuid.Parse(panelIDStr)
	panel, err := h.store.FindPanelByID(ctx, panelID)
	if err != nil || panel == nil {
		return c.Edit("❌ پنل یافت نشد.")
	}
	panel.IsActive = !panel.IsActive
	h.store.UpdatePanel(ctx, panel)
	status := "فعال"
	if !panel.IsActive {
		status = "غیرفعال"
	}
	return c.Edit(fmt.Sprintf("✅ پنل <b>%s</b> %s شد.", panel.Name, status), tele.ModeHTML)
}

func (h *Handler) deletePanel(ctx context.Context, c tele.Context, panelIDStr string) error {
	defer c.Respond()
	panelID, _ := uuid.Parse(panelIDStr)
	h.store.DeletePanel(ctx, panelID)
	return c.Edit("🗑 پنل حذف شد.")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
