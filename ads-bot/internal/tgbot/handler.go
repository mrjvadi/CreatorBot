package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/ads-bot/internal/engine"
	"github.com/mrjvadi/creatorbot/ads-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Handler struct {
	bot     *tele.Bot
	store   *store.Store
	engine  *engine.Engine
	cache   ports.Cache
	log     ports.Logger
	ownerID int64
	pay     *natspayclient.Client // برای کسر بودجه‌ی اجاره‌ی قفل از کیف پول خریدار
}

func NewHandler(
	bot *tele.Bot,
	st *store.Store,
	eng *engine.Engine,
	cache ports.Cache,
	log ports.Logger,
	ownerID int64,
	pay *natspayclient.Client,
) *Handler {
	return &Handler{bot: bot, store: st, engine: eng, cache: cache, log: log, ownerID: ownerID, pay: pay}
}

func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start", h.onStart)
	b.Handle("/help", h.onHelp)
	b.Handle("/newcampaign", h.onNewCampaign)
	b.Handle("/campaigns", h.onMyCampaigns)
	b.Handle("/channels", h.onMyChannels)
	b.Handle("/addchannel", h.onAddChannel)
	b.Handle("/balance", h.onBalance)
	b.Handle("/rentlock", h.onRentLock)
	b.Handle("/admin", h.onAdmin)
	b.Handle("/cancel", h.onCancel)

	b.Handle(tele.OnText, h.onText)
	b.Handle(tele.OnPhoto, h.onMedia)
	b.Handle(tele.OnVideo, h.onMedia)
	b.Handle(tele.OnChannelPost, h.onChannelPost)
	b.Handle(tele.OnCallback, h.onCallback)
}

func (h *Handler) isAdmin(c tele.Context) bool { return c.Sender().ID == h.ownerID }

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()
	pub, _ := h.store.UpsertPublisher(ctx,
		c.Sender().ID, c.Sender().Username, c.Sender().FirstName)

	if h.isAdmin(c) {
		return c.Send("👑 پنل ادمین:", kbAdminMain())
	}
	return c.Send(
		"<b>📣 Ads Bot</b>\n\n"+
			"سلام <b>"+c.Sender().FirstName+"</b>!\n\n"+
			"💰 موجودی: <b>"+fmtFloat(pub.Balance)+" TON</b>\n\n"+
			"با این ربات می‌توانید کمپین تبلیغاتی بسازید\nیا از کانال خود درآمد کسب کنید.",
		tele.ModeHTML, kbMain(),
	)
}

func (h *Handler) onHelp(c tele.Context) error {
	return c.Send(
		"<b>❓ راهنما</b>\n\n"+
			"<b>📣 ناشر تبلیغ:</b>\n"+
			"کمپین بسازید → بودجه تعیین کنید → تبلیغ در کانال‌ها پخش می‌شود\n\n"+
			"<b>📢 صاحب کانال:</b>\n"+
			"کانال خود را ثبت کنید → حداقل CPJ را تعیین کنید\n→ تبلیغات دریافت کنید → درآمد کسب کنید\n\n"+
			"<b>CPJ</b> = Cost Per Join — هزینه جذب هر عضو جدید",
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
	case btnNewCampaign:
		return h.onNewCampaign(c)
	case btnMyCampaigns:
		return h.onMyCampaigns(c)
	case btnMyChannels:
		return h.onMyChannels(c)
	case btnAddChannel:
		return h.onAddChannel(c)
	case btnBalance:
		return h.onBalance(c)
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

func (h *Handler) onMedia(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	st := h.getState(ctx, uid)
	if st.Step == stepCampMedia {
		return h.handleCampaignMedia(ctx, c, st)
	}
	return nil
}

func (h *Handler) onChannelPost(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	st := h.getState(ctx, uid)
	if st.Step == stepChannelFwd {
		return h.handleChannelForward(ctx, c, st)
	}
	return nil
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
	case "approve":
		if len(parts) == 2 {
			if !h.isAdmin(c) {
				return c.Edit("⛔️ این عملیات فقط برای ادمین اصلی است.")
			}
			return h.approveCampaign(ctx, c, parts[1])
		}
	case "reject":
		if len(parts) == 2 {
			if !h.isAdmin(c) {
				return c.Edit("⛔️ این عملیات فقط برای ادمین اصلی است.")
			}
			return h.startReject(ctx, c, parts[1])
		}
	case "camp_pause":
		if len(parts) == 2 {
			return h.pauseCampaign(ctx, c, parts[1])
		}
	case "camp_del":
		if len(parts) == 2 {
			return h.deleteCampaign(ctx, c, parts[1])
		}
	case "verify_ch":
		if len(parts) == 2 {
			if !h.isAdmin(c) {
				return c.Edit("⛔️ این عملیات فقط برای ادمین اصلی است.")
			}
			return h.verifyChannel(ctx, c, parts[1])
		}
	case "reject_ch":
		if len(parts) == 2 {
			if !h.isAdmin(c) {
				return c.Edit("⛔️ این عملیات فقط برای ادمین اصلی است.")
			}
			return h.rejectChannel(ctx, c, parts[1])
		}
	case "rent_approve":
		if len(parts) == 2 {
			if !h.isAdmin(c) {
				return c.Edit("⛔️ تأیید اجاره‌ی قفل فقط با ادمین اصلی پلتفرم است.")
			}
			return h.approveLockRental(ctx, c, parts[1])
		}
	case "rent_reject":
		if len(parts) == 2 {
			if !h.isAdmin(c) {
				return c.Edit("⛔️ این عملیات فقط برای ادمین اصلی است.")
			}
			return h.rejectLockRental(ctx, c, parts[1])
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
	case stepCampName:
		return h.handleCampName(ctx, c, st, text)
	case stepCampCaption:
		return h.handleCampCaption(ctx, c, st, text)
	case stepCampButton:
		return h.handleCampButton(ctx, c, st, text)
	case stepCampURL:
		return h.handleCampURL(ctx, c, st, text)
	case stepCampBudget:
		return h.handleCampBudget(ctx, c, st, text)
	case stepCampCPJ:
		return h.handleCampCPJ(ctx, c, st, text)
	case stepChannelCPJ:
		return h.handleChannelCPJ(ctx, c, st, text)
	case stepAdminBroadcast:
		return h.doBroadcast(ctx, c, text)
	case stepRejectNote:
		return h.doReject(ctx, c, st, text)
	case stepRentChannel:
		return h.handleRentChannel(ctx, c, st, text)
	case stepRentBudget:
		return h.handleRentBudget(ctx, c, st, text)
	case stepRentReward:
		return h.handleRentReward(ctx, c, st, text)
	}
	return nil
}

func (h *Handler) onBalance(c tele.Context) error {
	ctx := context.Background()
	pub, _ := h.store.FindPublisher(ctx, c.Sender().ID)
	bal := 0.0
	if pub != nil {
		bal = pub.Balance
	}
	return c.Send(
		"<b>💰 موجودی</b>\n\n<b>"+fmtFloat(bal)+" TON</b>",
		tele.ModeHTML, kbMain(),
	)
}

func fmtFloat(f float64) string {
	return strings.TrimRight(strings.TrimRight(
		strings.Replace(fmt.Sprintf("%.4f", f), ".", ".", 1),
		"0"), ".")
}

// suppress unused import
var _ = ports.F
