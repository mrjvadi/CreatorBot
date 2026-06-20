// Package scheduler runs periodic background jobs for member-bot.
package scheduler

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/member-bot/internal/models"
	"github.com/mrjvadi/creatorbot/member-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Scheduler struct {
	store *store.Store
	bot   *tele.Bot
	log   ports.Logger
}

func New(st *store.Store, bot *tele.Bot, log ports.Logger) *Scheduler {
	return &Scheduler{store: st, bot: bot, log: log}
}

func (s *Scheduler) Start(ctx context.Context) {
	go runEvery(ctx, 1*time.Minute, func() { s.expireLocks(ctx) })
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

// expireLocks finds locks that have timed out or hit member capacity,
// marks them expired, and notifies the owner.
func (s *Scheduler) expireLocks(ctx context.Context) {
	locks, err := s.store.FindExpiredLocks(ctx)
	if err != nil {
		s.log.Error("expireLocks: query failed", ports.F("err", err))
		return
	}
	for _, lock := range locks {
		if err := s.store.ExpireLock(ctx, lock.ID); err != nil {
			s.log.Error("expireLocks: update failed", ports.F("lock", lock.ID), ports.F("err", err))
			continue
		}
		msg := fmt.Sprintf("⚠️ قفل کانال <b>%s</b> منقضی شد.\nدلیل: %s",
			lock.ChannelTitle, expireReason(lock))
		// Owner رو از store بگیر
		owner, err := s.store.FindOwnerByID(ctx, lock.OwnerID)
		if err != nil || owner == nil {
			continue
		}
		if _, err := s.bot.Send(&tele.User{ID: owner.TelegramID}, msg, tele.ModeHTML); err != nil {
			s.log.Error("expireLocks: notify failed", ports.F("err", err))
		}
		s.log.Info("lock expired", ports.F("channel", lock.ChannelID))
	}
}

func expireReason(l models.Lock) string {
	if l.MaxMembers > 0 && l.CurrentCount >= l.MaxMembers {
		return fmt.Sprintf("ظرفیت تکمیل شد (%d/%d)", l.CurrentCount, l.MaxMembers)
	}
	return fmt.Sprintf("زمان پایان یافت (%s)", l.ExpiresAt.Format("2006-01-02 15:04"))
}
