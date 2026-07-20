package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
)

// AdminPaymentsList همان تاریخچه پرداخت پنل وب را در تلگرام نشان می‌دهد.
func (h *Admin) AdminPaymentsList(ctx context.Context, c tele.Context) error {
	payments, err := h.Store.ListAllPayments(ctx)
	if err != nil {
		return c.Send("❌ payment history failed", h.KbAdmin(ctx, c.Sender().ID))
	}
	lines := []string{"💳 <b>آخرین پرداخت‌ها</b>", ""}
	start := 0
	if len(payments) > 20 {
		start = len(payments) - 20
	}
	for i := len(payments) - 1; i >= start; i-- {
		p := payments[i]
		lines = append(lines, fmt.Sprintf("<code>%s</code> · %.4f TON · %s", p.ID.String()[:8], p.Amount, p.Status))
	}
	if len(payments) == 0 {
		lines = append(lines, "—")
	}
	return c.Send(strings.Join(lines, "\n"), tele.ModeHTML, h.KbAdmin(ctx, c.Sender().ID))
}

// AdminAuditList همان audit trail پنل وب را در تلگرام نشان می‌دهد.
func (h *Admin) AdminAuditList(ctx context.Context, c tele.Context) error {
	logs, err := h.Store.ListAdminAuditLogs(ctx, "", "", 30)
	if err != nil {
		return c.Send("❌ audit log failed", h.KbAdmin(ctx, c.Sender().ID))
	}
	lines := []string{"🧾 <b>تاریخچه ممیزی</b>", ""}
	for _, item := range logs {
		lines = append(lines, fmt.Sprintf("<code>%s</code> · %s · %s", item.Action, item.TargetType, item.Description))
	}
	if len(logs) == 0 {
		lines = append(lines, "—")
	}
	return c.Send(strings.Join(lines, "\n"), tele.ModeHTML, h.KbAdmin(ctx, c.Sender().ID))
}

// AdminBotMigrateMenu سرورهای آنلاین مقصد را نشان می‌دهد. Instance ID در state
// نگه داشته می‌شود تا callback از محدودیت ۶۴ بایت تلگرام عبور نکند.
func (h *Admin) AdminBotMigrateMenu(ctx context.Context, c tele.Context, instanceID string) error {
	inst, err := h.Store.FindInstance(ctx, instanceID)
	if err != nil || inst == nil {
		return c.Respond(&tele.CallbackResponse{Text: "instance not found", ShowAlert: true})
	}
	servers, _ := h.Store.ListServers(ctx)
	h.SetStep(ctx, c.Sender().ID, state.StepIdle, "migrate_instance", instanceID)
	kb := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)
	for _, server := range servers {
		if server.IsOnline && server.ID != inst.ServerID {
			rows = append(rows, kb.Row(kb.Data("🖥 "+server.Name, "admin_bot_migrate_do:"+server.ID.String())))
		}
	}
	if len(rows) == 0 {
		return c.Respond(&tele.CallbackResponse{Text: "no online target server", ShowAlert: true})
	}
	rows = append(rows, kb.Row(kb.Data("❌ لغو", "cancel")))
	kb.Inline(rows...)
	_ = c.Respond()
	return c.Send("🔄 <b>سرور مقصد را انتخاب کنید</b>\n"+inst.ContainerName, tele.ModeHTML, kb)
}

func (h *Admin) AdminBotMigrate(ctx context.Context, c tele.Context, targetServerID string) error {
	st := h.GetState(ctx, c.Sender().ID)
	instanceID := st.Data["migrate_instance"]
	h.ClearState(ctx, c.Sender().ID)
	inst, err := h.Store.FindInstance(ctx, instanceID)
	if err != nil || inst == nil {
		return c.Respond(&tele.CallbackResponse{Text: "instance not found", ShowAlert: true})
	}
	target, err := h.Store.FindServerByID(ctx, targetServerID)
	if err != nil || target == nil || !target.IsOnline || target.ID == inst.ServerID {
		return c.Respond(&tele.CallbackResponse{Text: "invalid target server", ShowAlert: true})
	}
	tmpl, err := h.Store.FindTemplate(ctx, inst.TemplateID.String())
	if err != nil || tmpl == nil {
		return c.Respond(&tele.CallbackResponse{Text: "template not found", ShowAlert: true})
	}
	plainToken, err := auth.Decrypt(inst.BotToken, h.EncryptKey)
	if err != nil {
		// سازگاری مهاجرتی برای رکوردهای قدیمی botmanager که plaintext بوده‌اند.
		if _, tokenErr := models.BotIDFromToken(inst.BotToken); tokenErr != nil {
			return c.Respond(&tele.CallbackResponse{Text: "token decrypt failed", ShowAlert: true})
		}
		plainToken = inst.BotToken
		if encrypted, encErr := auth.Encrypt(plainToken, h.EncryptKey); encErr == nil {
			_ = h.Store.UpdateBotToken(ctx, inst.BotID, encrypted)
		}
	}
	containerID := inst.ContainerID
	if containerID == "" {
		containerID = inst.ContainerName
	}
	if err := h.Docker.Stop(ctx, inst.ServerID.String(), containerID); err != nil {
		h.Log.Warn("telegram migrate: stop old container failed", h.F("err", err), h.F("instance", inst.ID))
	}
	owner, _ := h.Store.FindUserByID(ctx, inst.OwnerID)
	ownerTelegram := int64(0)
	ownerRole := "user"
	if owner != nil {
		ownerTelegram, ownerRole = owner.TelegramID, string(owner.Role)
	}
	planID := ""
	if inst.PlanID != nil {
		planID = inst.PlanID.String()
	}
	jwtToken, _ := auth.GenerateAccessToken(inst.OwnerID.String(), ownerRole, auth.JWTConfig{AccessSecret: h.EncryptKey})
	licenseToken := ""
	if h.License != nil {
		licenseToken, _ = h.License.Issue(ctx, inst.BotID, "bot_"+strconv.FormatInt(inst.BotID, 10), inst.OwnerID.String(), target.ID.String(), planID)
	}
	env := map[string]string{
		"BOT_TOKEN": plainToken, "INSTANCE_ID": "bot_" + strconv.FormatInt(inst.BotID, 10),
		"OWNER_TELEGRAM": strconv.FormatInt(ownerTelegram, 10), "OWNER_ID": strconv.FormatInt(ownerTelegram, 10),
		"PLAN_ID": planID, "JWT_TOKEN": jwtToken, "LICENSE_TOKEN": licenseToken, "SERVER_ID": target.ID.String(),
	}
	if inst.EnvOverrides != "" {
		var overrides map[string]string
		if json.Unmarshal([]byte(inst.EnvOverrides), &overrides) == nil {
			for key, value := range overrides {
				env[key] = value
			}
		}
	}
	cmd := protocol.DeployCommand{Type: protocol.MsgDeploy, ContainerName: inst.ContainerName, ImageName: tmpl.ImageName, ImageTag: tmpl.ImageTag, EnvVars: env}
	if err := h.Docker.Deploy(ctx, target.ID.String(), cmd); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "deploy on target failed", ShowAlert: true})
	}
	if err := h.Store.UpdateInstanceServer(ctx, inst.ID, target.ID); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "database update failed", ShowAlert: true})
	}
	_ = h.Store.UpdateInstanceStatus(ctx, inst.ID, models.StatusPending)
	if actor, _ := h.LoadUser(ctx, c); actor != nil {
		h.AuditLog(ctx, actor.ID, string(actor.Role), inst.ID.String(), "instance", models.AuditAdminAction, "migrate to "+target.ID.String())
	}
	_ = c.Respond(&tele.CallbackResponse{Text: "migration queued"})
	return h.AdminBotsList(ctx, c)
}
