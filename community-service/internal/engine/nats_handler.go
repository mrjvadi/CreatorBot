package engine

import (
	"context"
	"encoding/json"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/community-service/internal/store"
)

// RegisterNATSListeners همه NATS subscriptions را ثبت می‌کند.
func (e *Engine) RegisterNATSListeners(nc *natsclient.Client) {
	// ── membership events از member-bot ───────────────────
	nc.Subscribe("membership.joined", func(data []byte) {
		var event struct {
			TelegramID  int64  `json:"telegram_id"`
			ChatID      int64  `json:"community_id"`
			CampaignID  string `json:"campaign_id"`
			InviteLink  string `json:"invite_link"`
		}
		if err := json.Unmarshal(data, &event); err != nil {
			return
		}
		ctx := context.Background()
		e.HandleJoin(ctx, event.TelegramID, event.ChatID, event.CampaignID, event.InviteLink)
	})

	nc.Subscribe("membership.left", func(data []byte) {
		var event struct {
			TelegramID  int64 `json:"telegram_id"`
			ChatID      int64 `json:"community_id"`
		}
		if err := json.Unmarshal(data, &event); err != nil {
			return
		}
		ctx := context.Background()
		e.HandleLeave(ctx, event.TelegramID, event.ChatID)
	})

	// ── activity events از member-bot ────────────────────
	nc.Subscribe("community.activity.updated", func(data []byte) {
		var event struct {
			TelegramID  int64 `json:"telegram_id"`
			ChatID      int64 `json:"community_id"`
			Messages    int   `json:"messages"`
			Replies     int   `json:"replies"`
			Reactions   int   `json:"reactions"`
		}
		if err := json.Unmarshal(data, &event); err != nil {
			return
		}
		ctx := context.Background()
		community, _ := e.store.FindCommunityByChatID(ctx, event.ChatID)
		if community == nil || community.Type != store.TypeGroup {
			return
		}
		e.store.IncrementMemberActivity(ctx, community.ID, event.TelegramID,
			event.Messages, event.Replies, event.Reactions)
	})

	// ── revenue event از ads-bot ──────────────────────────
	nc.Subscribe("campaign.revenue.generated", func(data []byte) {
		var event struct {
			CommunityID string  `json:"community_id"`
			CampaignID  string  `json:"campaign_id"`
			RevenueTON  float64 `json:"revenue_ton"`
			ValidJoins  int     `json:"valid_joins"`
		}
		if err := json.Unmarshal(data, &event); err != nil {
			return
		}
		ctx := context.Background()

		// community_id می‌تواند chat_id عددی باشد
		var communityID interface{}
		var chatID int64
		var err error

		// اگه عددی بود → chat_id، اگه UUID بود → community UUID
		if err = json.Unmarshal([]byte(`"`+event.CommunityID+`"`), &chatID); err == nil {
			community, _ := e.store.FindCommunityByChatID(ctx, chatID)
			if community != nil {
				communityID = community.ID
			}
		} else {
			communityID = event.CommunityID
		}
		_ = communityID
		_ = err

		community, _ := e.store.FindCommunityByChatID(ctx, chatID)
		if community == nil {
			return
		}
		e.DistributeRevenue(ctx, community.ID, event.CampaignID, event.RevenueTON, event.ValidJoins)
	})

	// ── fraud score updates ────────────────────────────────
	nc.Subscribe("community.score.updated", func(data []byte) {
		var event struct {
			CommunityID int64   `json:"community_id"` // chat_id
			Score       int     `json:"score"`
		}
		if err := json.Unmarshal(data, &event); err != nil {
			return
		}
		ctx := context.Background()
		e.store.UpdateCommunityScore(ctx, event.CommunityID, event.Score)
	})

	e.log.Info("community-service NATS listeners registered")
}

// ── helper ─────────────────────────────────────────────────

func logF(key string, val any) ports.Field { return ports.F(key, val) }
var _ = logF // suppress
