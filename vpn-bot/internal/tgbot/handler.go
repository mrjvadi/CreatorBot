// Package tgbot قابلیت‌های ربات VPN را پیاده‌سازی می‌کند.
//
// فایل‌ها:
//
//	handler.go   ← Handler، Register، /start، onText، onCallback
//	user.go      ← خرید، اشتراک من، کیف پول
//	admin.go     ← پنل ادمین
//	state.go     ← state machine در Redis
//	keyboards.go ← keyboard ها
//	helpers.go   ← توابع کمکی
package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/memberclient"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/joinevents"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/store"
)

// Handler handler اصلی vpn-bot.
type Handler struct {
	bot        *tele.Bot
	store      *store.Store
	panel      ports.VPNPanel
	gateway    ports.PaymentGateway
	cache      ports.Cache
	log        ports.Logger
	channelID  int64
	ownerID    int64
	botID      int64
	encryptKey string
	nats       *natsclient.Client

	// rentalStatus/joinPublisher فقط وقتی این instance رایگان به یک کمپینِ
	// اجاره‌ی قفلِ فعال در ads-bot وصل است معنی دارند — nil یعنی راه‌اندازی
	// نشده (مثلاً NATS در دسترس نبوده، رجوع main.go).
	rentalStatus  *memberclient.RentalStatus
	joinPublisher *joinevents.Publisher
}

func NewHandler(
	bot *tele.Bot,
	st *store.Store,
	panel ports.VPNPanel,
	gateway ports.PaymentGateway,
	cache ports.Cache,
	log ports.Logger,
	channelID int64,
	ownerID int64,
	encryptKey string,
	botID int64,
	nc *natsclient.Client,
	rentalStatus *memberclient.RentalStatus,
	joinPublisher *joinevents.Publisher,
) *Handler {
	return &Handler{
		bot: bot, store: st, panel: panel,
		gateway: gateway, cache: cache, log: log,
		channelID: channelID, ownerID: ownerID,
		encryptKey: encryptKey, botID: botID, nats: nc,
		rentalStatus: rentalStatus, joinPublisher: joinPublisher,
	}
}

func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start", h.onStart)
	b.Handle("/help", h.onHelp)
	b.Handle("/buy", h.onBuy)
	b.Handle("/myvpn", h.onMyVPN)
	b.Handle("/wallet", h.onWallet)
	b.Handle("/admin", h.onAdmin)
	b.Handle("/cancel", h.onCancel)

	b.Handle(tele.OnText, h.onText)
	b.Handle(tele.OnPhoto, h.onPhoto)
	b.Handle(tele.OnCallback, h.onCallback)
	b.Handle(tele.OnMyChatMember, h.onMyChatMember)
	if h.joinPublisher != nil {
		b.Handle(tele.OnChatMember, h.joinPublisher.HandleChatMember)
		b.Handle(tele.OnUserJoined, h.joinPublisher.HandleUserJoined)
		b.Handle(tele.OnUserLeft, h.joinPublisher.HandleUserLeft)
	}
}

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()

	u, err := h.getOrCreate(ctx, c)
	if err != nil {
		return c.Send("❌ خطا. دوباره امتحان کنید.")
	}
	if u.IsBlocked {
		return c.Send("⛔️ دسترسی شما محدود شده است.")
	}
	if err := h.checkMembership(ctx, c); err != nil {
		return err
	}
	if h.isAdmin(c) {
		return c.Send(
			fmt.Sprintf("سلام <b>%s</b> 👑\nپنل ادمین:", c.Sender().FirstName),
			tele.ModeHTML, kbAdminMain(),
		)
	}
	return c.Send(
		fmt.Sprintf(
			"👋 سلام <b>%s</b>!\n\nبه ربات VPN خوش آمدید.\n\n💳 موجودی: <b>%.0f تومان</b>",
			c.Sender().FirstName, u.Balance,
		),
		tele.ModeHTML, kbMain(),
	)
}

func (h *Handler) onHelp(c tele.Context) error {
	return c.Send(
		"<b>❓ راهنما</b>\n\n"+
			"🛒 <b>خرید:</b> پلن انتخاب کن و پرداخت کن\n"+
			"🔑 <b>اشتراک من:</b> لینک و QR Code\n"+
			"💳 <b>کیف پول:</b> موجودی\n\n"+
			"📱 لینک را در اپ VPN خود وارد کنید.",
		tele.ModeHTML, kbMain(),
	)
}

func (h *Handler) onCancel(c tele.Context) error {
	h.clearState(context.Background(), c.Sender().ID)
	return c.Send("لغو شد.", kbMain())
}

func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	st := h.getState(ctx, uid)
	if st.Step != stepIdle {
		return h.handleStep(ctx, c, st, text)
	}

	switch text {
	case btnBuy:
		return h.onBuy(c)
	case btnMyVPN:
		return h.onMyVPN(c)
	case btnWallet:
		return h.onWallet(c)
	case btnSupport:
		return c.Send("📞 پشتیبانی: @support")
	case btnHelp:
		return h.onHelp(c)
	case btnCancel, btnBack:
		h.clearState(ctx, uid)
		return c.Send("لغو شد.", kbMain())
	}

	if h.isAdmin(c) {
		return h.handleAdminText(ctx, c, text)
	}

	return nil
}

func (h *Handler) onPhoto(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	st := h.getState(ctx, uid)
	if st.Step != stepBuyReceipt && st.Step != stepRenewPayment {
		return nil
	}
	if c.Message().Photo == nil {
		return c.Send("لطفاً عکس رسید را ارسال کنید.")
	}
	photo := c.Message().Photo
	return h.handleReceiptPhoto(ctx, c, st, photo.FileID)
}

func (h *Handler) onCallback(c tele.Context) error {
	ctx := context.Background()
	data := c.Callback().Data
	if len(data) > 0 && data[0] == '\f' {
		data = data[1:]
	}
	defer c.Respond()

	parts := strings.SplitN(data, ":", 2)
	switch parts[0] {
	case "plan":
		if len(parts) == 2 {
			return h.onPlanSelected(ctx, c, parts[1])
		}
	case "gw":
		if len(parts) == 2 {
			return h.onGatewaySelected(ctx, c, parts[1])
		}
	case "link":
		if len(parts) == 2 {
			return h.sendSubscriptionLink(ctx, c, parts[1])
		}
	case "qr":
		if len(parts) == 2 {
			return h.sendSubscriptionQR(ctx, c, parts[1])
		}
	case "renew":
		if len(parts) == 2 {
			return h.onRenewSelected(ctx, c, parts[1])
		}
	case "panel_add":
		if !h.isAdmin(c) {
			return nil
		}
		return h.startAddPanel(ctx, c)
	case "panel_test_all":
		if !h.isAdmin(c) {
			return nil
		}
		return h.testAllPanels(ctx, c)
	case "ptype":
		if !h.isAdmin(c) {
			return nil
		}
		if len(parts) == 2 {
			return h.handlePanelType(ctx, c, parts[1])
		}
	case "panel_toggle":
		if !h.isAdmin(c) {
			return nil
		}
		if len(parts) == 2 {
			return h.togglePanel(ctx, c, parts[1])
		}
	case "panel_del":
		if !h.isAdmin(c) {
			return nil
		}
		if len(parts) == 2 {
			return h.deletePanel(ctx, c, parts[1])
		}
	case "buy":
		return h.onBuy(c)
	case "confirm_buy":
		if len(parts) == 2 {
			return h.confirmBuyWithBalance(ctx, c, parts[1])
		}
	case "verify_payment":
		if len(parts) == 2 {
			return h.verifyOnlinePayment(ctx, c, parts[1])
		}
	case "approve_pay":
		if !h.isAdmin(c) {
			return nil
		}
		if len(parts) == 2 {
			return h.approvePayment(ctx, c, parts[1])
		}
	case "reject_pay":
		if !h.isAdmin(c) {
			return nil
		}
		if len(parts) == 2 {
			return h.rejectPayment(ctx, c, parts[1])
		}
	case "cancel":
		h.clearState(ctx, c.Sender().ID)
		return c.Edit("لغو شد.")
	}
	return nil
}

func (h *Handler) handleStep(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	if text == btnCancel || text == btnBack {
		h.clearState(ctx, uid)
		return c.Send("لغو شد.", kbMain())
	}
	switch st.Step {
	case stepAddPanelURL:
		return h.handlePanelURL(ctx, c, st, text)
	case stepAddPanelUser:
		return h.handlePanelUser(ctx, c, st, text)
	case stepAddPanelPass:
		return h.handlePanelPass(ctx, c, st, text)
	case stepAddPanelCap:
		return h.handlePanelCap(ctx, c, st, text)
	case stepBuyPayment, stepRenewPayment:
		return h.handlePaymentInput(ctx, c, st, text)
	case stepAdminBroadcast:
		return h.doBroadcast(ctx, c, text)
	case stepAdminDiscount:
		return h.handleDiscountInput(ctx, c, st, text)
	}
	return nil
}

func (h *Handler) l() ports.Logger { return h.log }
