package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/models"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/store"
)

type Scheduler struct {
	store  *store.Store
	panel  ports.VPNPanel
	sender ports.BotSender
	log    ports.Logger
}

func New(st *store.Store, panel ports.VPNPanel, sender ports.BotSender, log ports.Logger) *Scheduler {
	return &Scheduler{store: st, panel: panel, sender: sender, log: log}
}

func (s *Scheduler) Start(ctx context.Context) {
	go runEvery(ctx, 1*time.Hour, func() { s.notifyExpiring(ctx) })
	go runEvery(ctx, 30*time.Minute, func() { s.disableExpired(ctx) })
	go runEvery(ctx, 15*time.Minute, func() { s.syncUsage(ctx) })
}

func runEvery(ctx context.Context, d time.Duration, fn func()) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			fn()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) notifyExpiring(ctx context.Context) {
	subs, err := s.store.FindSubscriptionsExpiringIn(ctx, 3*24*time.Hour)
	if err != nil {
		s.log.Error("notifyExpiring: query failed", ports.F("err", err))
		return
	}
	for _, sub := range subs {
		remaining := time.Until(sub.ExpiresAt).Round(time.Hour)
		msg := fmt.Sprintf("⚠️ اشتراک شما در <b>%s</b> منقضی می‌شود.\n\n/renew برای تمدید", remaining)
		// FIX 10: sub.User is populated by Preload
		if err := s.sender.Send(ctx, sub.User.TelegramID, msg, ports.WithHTML()); err != nil {
			s.log.Error("notifyExpiring: send failed", ports.F("err", err))
		}
	}
}

func (s *Scheduler) disableExpired(ctx context.Context) {
	subs, err := s.store.FindExpiredSubscriptions(ctx)
	if err != nil {
		s.log.Error("disableExpired: query failed", ports.F("err", err))
		return
	}
	for _, sub := range subs {
		if err := s.panel.DisableUser(ctx, sub.Username); err != nil {
			s.log.Error("disableExpired: panel error",
				ports.F("username", sub.Username), ports.F("err", err))
			continue
		}
		s.store.UpdateSubscriptionStatus(ctx, sub.ID, models.SubDisabled)
		s.log.Info("subscription disabled", ports.F("username", sub.Username))
	}
}

func (s *Scheduler) syncUsage(ctx context.Context) {
	subs, err := s.store.FindActiveSubscriptions(ctx)
	if err != nil {
		s.log.Error("syncUsage: query failed", ports.F("err", err))
		return
	}
	for _, sub := range subs {
		user, err := s.panel.GetUser(ctx, sub.Username)
		if err != nil {
			continue
		}
		s.store.UpdateSubscriptionUsage(ctx, sub.ID, user.UsedData)
	}
}
