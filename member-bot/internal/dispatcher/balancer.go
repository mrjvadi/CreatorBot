// Package dispatcher - balancer.go
// تقسیم کانال‌ها بین check bot ها به صورت هوشمند.
//
// هر کانال به حداقل ۲ bot اختصاص داده می‌شود (redundancy).
// اگه یه bot down شد، bot دیگه‌ای جواب می‌دهد.
// بار بین bot ها با round-robin تقسیم می‌شود.
package dispatcher

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/member-bot/internal/models"
	"github.com/mrjvadi/creatorbot/member-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

const (
	// هر کانال حداقل به چند bot اختصاص داده شود
	minBotsPerChannel = 2
)

// Balancer تقسیم کانال‌ها بین bot ها.
type Balancer struct {
	store *store.Store
	cache ports.Cache
	log   ports.Logger
}

func NewBalancer(st *store.Store, cache ports.Cache, log ports.Logger) *Balancer {
	return &Balancer{store: st, cache: cache, log: log}
}

// Assign کانال‌ها را بین bot های فعال تقسیم می‌کند.
// هر کانال به min(minBotsPerChannel, len(bots)) bot اختصاص می‌یابد.
func (b *Balancer) Assign(ctx context.Context) error {
	bots, err := b.store.FindActiveBots(ctx)
	if err != nil {
		return fmt.Errorf("balancer: load bots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	locks, err := b.store.ListAllLocks(ctx)
	if err != nil {
		return fmt.Errorf("balancer: load locks: %w", err)
	}
	if len(locks) == 0 {
		return nil
	}

	// تعداد کانال به ازای هر bot (برای load balance)
	botLoad := make(map[string]int, len(bots))
	for _, bot := range bots {
		botLoad[bot.ID.String()] = len(bot.Memberships)
	}

	// تعداد redundancy — نمی‌تواند بیشتر از bot های موجود باشد
	redundancy := minBotsPerChannel
	if len(bots) < redundancy {
		redundancy = len(bots)
	}

	assigned := 0
	for _, lock := range locks {
		if lock.Status != models.LockActive {
			continue
		}

		// bot های کمتر-لود را انتخاب کن
		selected := b.selectLeastLoaded(bots, botLoad, redundancy)

		for _, botID := range selected {
			if err := b.store.AddBotMembership(ctx, &models.BotChannelMembership{
				BotID: botID,
				ChannelID:  lock.ChannelID,
			}); err != nil {
				b.log.Error("balancer: add membership",
					ports.F("bot", botID),
					ports.F("channel", lock.ChannelID),
					ports.F("err", err))
				continue
			}
			botLoad[botID.String()]++
		}
		assigned++
	}

	b.log.Info("channel assignment done",
		ports.F("channels", assigned),
		ports.F("bots", len(bots)),
		ports.F("redundancy", redundancy))

	return nil
}

// Rebalance وقتی bot جدید اضافه می‌شود، کانال‌ها را دوباره تقسیم می‌کند.
func (b *Balancer) Rebalance(ctx context.Context) error {
	// پاک کردن assignment فعلی
	if err := b.store.ClearBotMemberships(ctx); err != nil {
		return fmt.Errorf("rebalance: clear: %w", err)
	}
	return b.Assign(ctx)
}

// OnBotDown وقتی یه bot down می‌شود، کانال‌هایش را به bot های دیگر منتقل می‌کند.
func (b *Balancer) OnBotDown(ctx context.Context, downBotID string) error {
	b.log.Info("bot down — reassigning channels", ports.F("bot", downBotID))

	// غیرفعال کردن bot
	if err := b.store.DeactivateBotByID(ctx, downBotID); err != nil {
		b.log.Error("deactivate bot failed", ports.F("err", err))
	}

	// rebalance بقیه
	return b.Rebalance(ctx)
}

// Stats آمار assignment فعلی.
func (b *Balancer) Stats(ctx context.Context) map[string]int {
	bots, _ := b.store.FindActiveBots(ctx)
	result := make(map[string]int, len(bots))
	for _, bot := range bots {
		result[bot.ID.String()] = len(bot.Memberships)
	}
	return result
}

// ── helpers ──────────────────────────────────────────────

// selectLeastLoaded n تا bot با کمترین load انتخاب می‌کند.
func (b *Balancer) selectLeastLoaded(bots []models.CheckBot, loads map[string]int, n int) []uuid.UUID {
	type botWithLoad struct {
		bot  models.CheckBot
		load int
	}

	ranked := make([]botWithLoad, 0, len(bots))
	for _, bot := range bots {
		ranked = append(ranked, botWithLoad{bot, loads[bot.ID.String()]})
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].load < ranked[j].load
	})

	if n > len(ranked) {
		n = len(ranked)
	}

	selected := make([]uuid.UUID, n)
	for i := 0; i < n; i++ {
		selected[i] = ranked[i].bot.ID
	}
	return selected
}

// HealthCheck بررسی می‌کند کدام bot ها واقعاً آنلاین هستند.
// اگه bot ای از Redis پاسخ ندهد، OnBotDown صدا زده می‌شود.
func (b *Balancer) HealthCheck(ctx context.Context) error {
	bots, err := b.store.FindActiveBots(ctx)
	if err != nil {
		return err
	}

	for _, bot := range bots {
		heartbeatKey := "bot:heartbeat:" + bot.ID.String()
		_, err := b.cache.Get(ctx, heartbeatKey)
		if err != nil {
			// bot پاسخ نداده — offline
			b.log.Warn("bot appears offline",
				ports.F("bot", bot.ID),
				ports.F("username", bot.Username))
			_ = b.OnBotDown(ctx, bot.ID.String())
		}
	}
	return nil
}

// ScaleUp یک check-bot جدید اضافه می‌کند و rebalance می‌کند.
func (b *Balancer) ScaleUp(ctx context.Context, token, encryptedToken string) error {
	// ثبت bot جدید
	newBot := &models.CheckBot{
		Token:     encryptedToken,
		IsActive:  true,
		RateLimit: 20,
	}
	if err := b.store.CreateCheckBot(ctx, newBot); err != nil {
		return fmt.Errorf("scale up: create bot: %w", err)
	}

	b.log.Info("new check-bot registered, rebalancing",
		ports.F("bot", newBot.ID))

	// rebalance با bot جدید
	return b.Rebalance(ctx)
}

// ScaleDown یک check-bot را حذف می‌کند.
func (b *Balancer) ScaleDown(ctx context.Context, botID string) error {
	if err := b.OnBotDown(ctx, botID); err != nil {
		return err
	}
	return b.store.DeleteBotByID(ctx, botID)
}
