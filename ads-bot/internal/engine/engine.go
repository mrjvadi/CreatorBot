// Package engine منطق توزیع تبلیغات و محاسبه درآمد ناشران را پیاده‌سازی می‌کند.
package engine

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/google/uuid"
	"github.com/mrjvadi/creatorbot/ads-bot/internal/store"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/fraudclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Broadcaster interface ارسال پیام به کانال.
type Broadcaster interface {
	SendToChannel(ctx context.Context, channelID int64, campaign *store.Campaign) (int, error)
}

// Engine مدیریت توزیع کمپین‌ها.
type Engine struct {
	store       *store.Store
	nc          *natsclient.Client
	broadcaster Broadcaster
	fraud       *fraudclient.Client
	log         ports.Logger
}

func New(st *store.Store, nc *natsclient.Client, b Broadcaster, fraud *fraudclient.Client, log ports.Logger) *Engine {
	return &Engine{store: st, nc: nc, broadcaster: b, fraud: fraud, log: log}
}

// ── Distribute ─────────────────────────────────────────────

// DistributeCampaign یک کمپین را به کانال‌های مناسب ارسال می‌کند.
func (e *Engine) DistributeCampaign(ctx context.Context, campaign *store.Campaign) error {
	if campaign.Status != store.CampaignActive {
		return fmt.Errorf("campaign not active")
	}
	if campaign.IsExpired() {
		e.store.UpdateCampaign(ctx, &store.Campaign{
			ID: campaign.ID, Status: store.CampaignDone,
		})
		return nil
	}

	// پیدا کردن کانال‌های مناسب (CPJ کانال <= CPJ کمپین)
	channels, err := e.store.ListActiveChannels(ctx, campaign.CPJ)
	if err != nil || len(channels) == 0 {
		e.log.Info("no channels for campaign",
			ports.F("campaign", campaign.ID),
			ports.F("cpj", campaign.CPJ))
		return nil
	}

	remaining := campaign.RemainingBudget()
	if remaining <= 0 {
		e.store.UpdateCampaign(ctx, &store.Campaign{
			ID: campaign.ID, Status: store.CampaignDone,
		})
		return nil
	}

	sent := 0
	for _, ch := range channels {
		// بررسی بودجه
		if remaining <= 0 {
			break
		}

		// ارسال به کانال
		msgID, err := e.broadcaster.SendToChannel(ctx, ch.ChannelID, campaign)
		if err != nil {
			e.log.Error("send to channel",
				ports.F("channel", ch.ChannelID),
				ports.F("err", err))
			continue
		}

		// ثبت impression
		imp := &store.Impression{
			CampaignID: campaign.ID,
			ChannelID:  ch.ChannelID,
			MessageID:  msgID,
		}
		e.store.CreateImpression(ctx, imp)
		sent++

		e.log.Info("campaign sent to channel",
			ports.F("campaign", campaign.ID),
			ports.F("channel", ch.ChannelID))
	}

	e.log.Info("campaign distributed",
		ports.F("campaign", campaign.ID),
		ports.F("channels", sent))
	return nil
}

// RecordJoin ثبت عضو جدید از تبلیغ.
// user_telegram_id برای بررسی fake بودن از fraud-engine استفاده می‌شود.
func (e *Engine) RecordJoin(ctx context.Context, campaignID uuid.UUID, channelTelegramID int64, userTelegramID int64) error {
	campaign, err := e.store.FindCampaign(ctx, campaignID)
	if err != nil || campaign == nil {
		return fmt.Errorf("campaign not found")
	}

	// ── بررسی fake بودن کاربر از fraud-engine ─────────────
	trustMultiplier := 1.0
	if e.fraud != nil && userTelegramID > 0 {
		score, _ := e.fraud.GetUserScore(ctx, userTelegramID)
		if score != nil {
			trustMultiplier = score.TrustMultiplier()
			if score.IsFake() {
				e.log.Info("fake user detected — join skipped",
					ports.F("user", userTelegramID),
					ports.F("score", score.Score))
				return nil // join fake را نادیده بگیر
			}
		}
	}

	// هزینه واقعی = CPJ × trustMultiplier
	// کاربران با امتیاز پایین‌تر → CPJ کمتر → revenue کمتر
	effectiveCPJ := campaign.CPJ * trustMultiplier
	cost := effectiveCPJ
	if cost > campaign.RemainingBudget() {
		cost = campaign.RemainingBudget()
	}

	e.store.AddJoinCount(ctx, campaignID, 1, cost)

	// پرداخت به صاحب کانال
	e.nc.PublishCore("earning.created", map[string]any{
		"type":              "ad_income",
		"owner_telegram_id": channelTelegramID,
		"total_nano":        int64(cost * 1e9),
		"bot_id":            "ads-bot",
		"ref_id":            campaignID.String(),
		"description":       fmt.Sprintf("درآمد تبلیغ — trust:%.0f%%", trustMultiplier*100),
	})

	return nil
}

// RunScheduler کمپین‌های فعال را در فاصله‌های زمانی پخش می‌کند.
func (e *Engine) RunScheduler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	e.log.Info("ads scheduler started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.runBatch(ctx)
		}
	}
}

func (e *Engine) runBatch(ctx context.Context) {
	campaigns, err := e.store.FindActiveCampaigns(ctx)
	if err != nil {
		e.log.Error("ads batch", ports.F("err", err))
		return
	}
	for _, c := range campaigns {
		cp := c
		if err := e.DistributeCampaign(ctx, &cp); err != nil {
			e.log.Error("distribute campaign",
				ports.F("id", c.ID), ports.F("err", err))
		}
	}
}

// ── TelegramBroadcaster ────────────────────────────────────

// TelegramBroadcaster ارسال کمپین به کانال از طریق تلگرام.
type TelegramBroadcaster struct {
	bot *tele.Bot
}

func NewBroadcaster(bot *tele.Bot) *TelegramBroadcaster {
	return &TelegramBroadcaster{bot: bot}
}

func (b *TelegramBroadcaster) SendToChannel(ctx context.Context, channelID int64, campaign *store.Campaign) (int, error) {
	chat := &tele.Chat{ID: channelID}

	var kb *tele.ReplyMarkup
	if campaign.ButtonURL != "" && campaign.ButtonText != "" {
		kb = &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.URL(campaign.ButtonText, campaign.ButtonURL)))
	}

	var msg *tele.Message
	var err error

	switch campaign.MediaType {
	case "photo":
		photo := &tele.Photo{
			File:    tele.FromFileID(campaign.MediaFileID),
			Caption: campaign.Caption,
		}
		if kb != nil {
			msg, err = b.bot.Send(chat, photo, tele.ModeHTML, kb)
		} else {
			msg, err = b.bot.Send(chat, photo, tele.ModeHTML)
		}
	case "video":
		video := &tele.Video{
			File:    tele.FromFileID(campaign.MediaFileID),
			Caption: campaign.Caption,
		}
		if kb != nil {
			msg, err = b.bot.Send(chat, video, tele.ModeHTML, kb)
		} else {
			msg, err = b.bot.Send(chat, video, tele.ModeHTML)
		}
	default:
		if kb != nil {
			msg, err = b.bot.Send(chat, campaign.Caption, tele.ModeHTML, kb)
		} else {
			msg, err = b.bot.Send(chat, campaign.Caption, tele.ModeHTML)
		}
	}

	if err != nil {
		return 0, err
	}
	return msg.ID, nil
}
