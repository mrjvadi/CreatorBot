// Package scheduler — زمان‌بندی jobهای admanager-bot.
//
// Scheduler هر N ثانیه دو کار انجام می‌دهد:
//  1. jobهای due را اجرا می‌کند (ارسال تبلیغ، ریپلی، حذف، آمار، یادآوری، پایان کمپین).
//  2. کمپین‌های در حال اجرا را بررسی و در صورت نیاز رزرو/ job ارسال جدید می‌سازد،
//     و بازه‌ی روزانه‌ی هرکدام را رعایت می‌کند (شروع تا پایان).
package scheduler

import (
	"context"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
	"github.com/mrjvadi/creatorbot/admanager-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Scheduler مدیر jobهای پس‌زمینه.
type Scheduler struct {
	store    *store.Store
	bot      *tele.Bot
	log      ports.Logger
	ownerID  int64 // برای ارسال پیام یادآوری در پیام خصوصی
	interval time.Duration
	quit     chan struct{}
}

// New یک Scheduler جدید می‌سازد.
func New(st *store.Store, bot *tele.Bot, log ports.Logger, ownerID int64) *Scheduler {
	return &Scheduler{
		store:    st,
		bot:      bot,
		log:      log,
		ownerID:  ownerID,
		interval: 30 * time.Second,
		quit:     make(chan struct{}),
	}
}

// Start حلقه‌ی اصلی scheduler را در یک goroutine شروع می‌کند.
func (s *Scheduler) Start() {
	go s.loop()
	s.log.Info("scheduler started", ports.F("interval", s.interval))
}

// Stop scheduler را متوقف می‌کند.
func (s *Scheduler) Stop() {
	close(s.quit)
}

func (s *Scheduler) loop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-s.quit:
			return
		}
	}
}

func (s *Scheduler) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// ۱) اجرای jobهای due
	jobs, err := s.store.FetchDueJobs(ctx)
	if err != nil {
		s.log.Error("fetch jobs error", ports.F("err", err))
	} else {
		for _, job := range jobs {
			if err := s.store.MarkJobRunning(ctx, job.ID); err != nil {
				continue
			}
			go s.runJob(job)
		}
	}

	// ۲) تولید رزرو برای کمپین‌های در حال اجرا (با رعایت بازه‌ی روزانه)
	s.planCampaigns(ctx)
}

// ── ساخت رزرو برای کمپین‌های فعال ────────────────────────────────

func (s *Scheduler) planCampaigns(ctx context.Context) {
	campaigns, err := s.store.ListActiveCampaigns(ctx)
	if err != nil {
		s.log.Error("list active campaigns", ports.F("err", err))
		return
	}
	now := time.Now()

	reminderLead := models.DefaultReminderMinutesBefore
	if st, err := s.store.GetSettings(ctx); err == nil && st != nil {
		reminderLead = st.ReminderMinutesBefore
	}

	for _, cm := range campaigns {
		if cm.EndAt != nil && now.After(*cm.EndAt) {
			_ = s.store.UpdateCampaignStatus(ctx, cm.ID, models.CampaignCompleted)
			continue
		}
		if cm.IntervalMinutes < 1 {
			continue // پیکربندی ناقص
		}

		// بازه‌ی روزانه‌ی شروع–پایان (با پشتیبانی از عبور از نیمه‌شب).
		// اگر الان بیرون از بازه‌ایم، هر چرخه‌ی در حال اجرا را فوراً قطع
		// می‌کنیم — بدون توقف خودِ ربات/سایر کمپین‌ها.
		if !models.InDailyWindow(now, cm.StartHour, cm.StartMinute, cm.EndHour, cm.EndMinute) {
			s.cutOffCampaign(ctx, cm)
			continue
		}

		ads, _ := s.store.ListAdsByCampaign(ctx, cm.ID)
		if len(ads) == 0 {
			continue
		}
		channels := s.resolveChannels(ctx, cm)
		if len(channels) == 0 {
			continue
		}

		windowStart := models.CurrentWindowStart(now, cm.StartHour, cm.StartMinute, cm.EndHour, cm.EndMinute)

		// انتخاب تبلیغ جاری بر اساس چرخش زمانی.
		elapsed := int(now.Sub(windowStart).Minutes())
		idx := models.RotationIndex(elapsed, cm.RotationMinutes, len(ads))
		ad := ads[idx]

		// فاصله‌ی شروع یک دور جدید = فاصله‌ی بین پست‌ها × (تعداد پست‌های یک دور).
		// یک دور = پست اصلی + ریپلی‌ها؛ بنابراین پست‌ها پیوسته و بدون هم‌پوشانی
		// تکرار می‌شوند. (چرخش فقط تعیین می‌کند کدام تبلیغ نمایش داده شود.)
		postsPerCycle := len(ad.Replies) + 1
		cycleGap := time.Duration(cm.IntervalMinutes*postsPerCycle) * time.Minute
		leadDur := time.Duration(reminderLead) * time.Minute

		for _, ch := range channels {
			// جلوگیری از ارسال هم‌زمانِ تکراری (رزروی که هنوز باز است،
			// چه در انتظار چه از قبل رزرو‌شده برای یادآوری).
			if open, _ := s.store.CountReservations(ctx, cm.ID, ch.ID,
				models.ReservationPending, models.ReservationConfirmed); open > 0 {
				continue
			}

			// زمانِ سررسیدِ اسلاتِ بعدی این کانال را حساب کن.
			nextDue := windowStart
			if last, ok := s.store.LastReservationAt(ctx, cm.ID, ch.ID); ok {
				nextDue = last.Add(cycleGap)
			}
			// اسلاتی که بیرون از بازه‌ی امروز می‌افتد را رد کن (فردا دوباره
			// محاسبه می‌شود).
			if !models.InDailyWindow(nextDue, cm.StartHour, cm.StartMinute, cm.EndHour, cm.EndMinute) {
				continue
			}
			// اگر یادآوری فعال است، از لحظه‌ی «سررسید منهای مهلت یادآوری»
			// به بعد رزرو را از قبل می‌سازیم؛ اگر خاموش است، دقیقاً در لحظه‌ی
			// سررسید (رفتار قبلی).
			if reminderLead > 0 {
				if now.Before(nextDue.Add(-leadDur)) {
					continue
				}
			} else if now.Before(nextDue) {
				continue
			}

			res := &models.Reservation{
				CampaignID:      cm.ID,
				AdvertisementID: ad.ID,
				ChannelID:       ch.ID,
				ScheduledAt:     nextDue,
			}
			if err := s.store.CreateReservation(ctx, res); err != nil {
				s.log.Error("create reservation", ports.F("err", err))
				continue
			}
			job := &models.ScheduledJob{
				Type:            models.JobTypeSendAd,
				CampaignID:      cm.ID,
				AdvertisementID: ad.ID,
				ChannelID:       ch.ID,
				RunAt:           nextDue,
				Payload:         res.ID,
			}
			if err := s.store.CreateJob(ctx, job); err != nil {
				s.log.Error("create send job", ports.F("err", err))
				continue
			}
			if reminderLead > 0 {
				reminderAt := nextDue.Add(-leadDur)
				if reminderAt.Before(now) {
					reminderAt = now
				}
				_ = s.store.CreateJob(ctx, &models.ScheduledJob{
					Type:            models.JobTypeReminder,
					CampaignID:      cm.ID,
					AdvertisementID: ad.ID,
					ChannelID:       ch.ID,
					RunAt:           reminderAt,
					Payload:         res.ID,
				})
			}
		}
	}
}

// resolveChannels کانال‌های هدف یک کمپین را از روی برچسب‌ها و کانال‌های خاص جمع می‌کند.
func (s *Scheduler) resolveChannels(ctx context.Context, cm models.Campaign) []models.Channel {
	seen := map[string]bool{}
	var out []models.Channel

	for _, tagID := range cm.TargetTagIDs {
		chs, _ := s.store.ListChannelsByTag(ctx, tagID, cm.MinMemberCount)
		for _, ch := range chs {
			if !seen[ch.ID] {
				seen[ch.ID] = true
				out = append(out, ch)
			}
		}
	}
	for _, chID := range cm.TargetChannelIDs {
		ch, err := s.store.FindChannel(ctx, chID)
		if err == nil && ch != nil && ch.Status == models.ChannelActive && !seen[ch.ID] {
			seen[ch.ID] = true
			out = append(out, *ch)
		}
	}
	return out
}

// cutOffCampaign بازه‌ی روزانه‌ی این کمپین تمام شده: هر رزروِ زنده‌ای که
// هنوز پیام در کانال دارد را فوراً پاک می‌کند، رزروهای در انتظار (که
// جلوی ساختِ رزروِ بعدی را می‌گیرند) را لغو می‌کند، و همه‌ی jobهای بازِ
// کمپین را لغو می‌کند. فقط همین کمپین متأثر می‌شود؛ ربات/scheduler/سایر
// کمپین‌ها ادامه‌ی کار می‌دهند.
func (s *Scheduler) cutOffCampaign(ctx context.Context, cm models.Campaign) {
	live, err := s.store.ListLiveReservationsByCampaign(ctx, cm.ID)
	if err != nil {
		s.log.Error("list live reservations for cutoff", ports.F("err", err))
	}
	for _, res := range live {
		ch, err := s.store.FindChannel(ctx, res.ChannelID)
		if err == nil && ch != nil {
			for _, mid := range res.LiveMessageIDs {
				_ = s.bot.Delete(&tele.Message{ID: mid, Chat: &tele.Chat{ID: ch.TelegramID}})
			}
		}
		_ = s.store.MarkReservationExpired(ctx, res.ID)
	}
	// رزروهای pending/confirmed (ساخته‌شده از قبل برای یادآوری، ولی هنوز
	// واقعاً ارسال نشده) هم باید لغو شوند — وگرنه برای همیشه جلوی
	// CountReservations را می‌گیرند و آن کانال دیگر هرگز رزروِ تازه نمی‌گیرد.
	_ = s.store.CancelCampaignReservations(ctx, cm.ID)
	_ = s.store.CancelCampaignJobs(ctx, cm.ID)
}

// ── اجرای jobها ──────────────────────────────────────────────────

func (s *Scheduler) runJob(job models.ScheduledJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var err error
	switch job.Type {
	case models.JobTypeSendAd:
		err = s.sendAd(ctx, job)
	case models.JobTypeSendReply:
		err = s.sendReply(ctx, job)
	case models.JobTypeDeletePost:
		err = s.deletePost(ctx, job)
	case models.JobTypeUpdateStats:
		err = s.updateStats(ctx, job)
	case models.JobTypeEndCampaign:
		err = s.endCampaign(ctx, job)
	case models.JobTypeReminder:
		err = s.sendReminder(ctx, job)
	default:
		s.log.Warn("unknown job type", ports.F("type", string(job.Type)))
		_ = s.store.MarkJobDone(ctx, job.ID)
		return
	}

	if err != nil {
		s.log.Error("job failed",
			ports.F("id", job.ID),
			ports.F("type", string(job.Type)),
			ports.F("err", err),
		)
		if job.Attempts < job.MaxAttempts {
			next := time.Now().Add(time.Duration(job.Attempts+1) * 5 * time.Minute)
			_ = s.store.RetryJob(ctx, job.ID, next)
		} else {
			_ = s.store.MarkJobFailed(ctx, job.ID, err.Error())
			if job.Payload != "" {
				_ = s.store.MarkReservationFailed(ctx, job.Payload, err.Error())
			}
		}
		return
	}

	_ = s.store.MarkJobDone(ctx, job.ID)
}

// ── Job handlers ──────────────────────────────────────────────────

// sendAd پست اصلی را می‌فرستد، سقفِ کلِ چرخه (DeleteAfterMinutes) را به‌عنوان
// شبکه‌ی ایمنی زمان‌بندی می‌کند، و اگر ریپلی داشته باشد اولین ریپلی را بعد
// از IntervalMinutes زمان‌بندی می‌کند.
func (s *Scheduler) sendAd(ctx context.Context, job models.ScheduledJob) error {
	ad, err := s.store.FindAd(ctx, job.AdvertisementID)
	if err != nil || ad == nil {
		return err
	}
	ch, err := s.store.FindChannel(ctx, job.ChannelID)
	if err != nil || ch == nil {
		return err
	}
	cm, _ := s.store.FindCampaign(ctx, job.CampaignID)
	if cm == nil {
		return nil
	}

	main, err := s.bot.Copy(&tele.Chat{ID: ch.TelegramID},
		&tele.Message{ID: ad.MainMessageID, Chat: &tele.Chat{ID: ad.SourceChatID}})
	if err != nil {
		return err
	}
	if job.Payload != "" {
		_ = s.store.SetReservationMainPosted(ctx, job.Payload, main.ID)
	}
	_ = s.store.IncrCampaignImpressions(ctx, job.CampaignID)
	s.afterPost(ctx, ch, main.ID)

	now := time.Now()

	// شبکه‌ی ایمنیِ سقفِ کل چرخه: هرچه پیش بیاید (ریپلی‌ها کِی تمام شوند)،
	// این job در بدترین حالت همه‌چیز را در همین لحظه‌ی مشخص پاک می‌کند.
	if cm.DeleteAfterMinutes > 0 && job.Payload != "" {
		deleteAt := now.Add(time.Duration(cm.DeleteAfterMinutes) * time.Minute)
		_ = s.store.CreateJob(ctx, &models.ScheduledJob{
			Type:       models.JobTypeDeletePost,
			CampaignID: job.CampaignID,
			ChannelID:  job.ChannelID,
			RunAt:      deleteAt,
			Payload:    job.Payload,
		})
		_ = s.store.SetReservationDeleteAt(ctx, job.Payload, deleteAt)
	}

	if len(ad.Replies) > 0 {
		_ = s.store.CreateJob(ctx, &models.ScheduledJob{
			Type:            models.JobTypeSendReply,
			CampaignID:      job.CampaignID,
			AdvertisementID: ad.ID,
			ChannelID:       job.ChannelID,
			RunAt:           now.Add(time.Duration(cm.IntervalMinutes) * time.Minute),
			Payload:         job.Payload + "|0",
		})
	}

	s.log.Info("ad main sent", ports.F("campaign", job.CampaignID), ports.F("channel", ch.Title))
	return nil
}

// sendReply یک ریپلی را جای ریپلیِ قبلی (در صورت وجود) می‌گذارد: ریپلیِ
// قبلی پاک می‌شود، ریپلیِ تازه با مدت‌زمانِ خودش می‌آید. اگر این آخرین
// ریپلیِ لیست بود، بعد از مدت‌زمانش کل چرخه (پست اصلی + این ریپلی) پاک
// می‌شود. Payload = "<resID>|<index>".
func (s *Scheduler) sendReply(ctx context.Context, job models.ScheduledJob) error {
	resID, idx, ok := splitReplyPayload(job.Payload)
	if !ok {
		return nil
	}
	res, err := s.store.FindReservation(ctx, resID)
	if err != nil || res == nil || len(res.LiveMessageIDs) == 0 {
		return nil // چرخه قبلاً قطع/تمام شده
	}
	ad, err := s.store.FindAd(ctx, job.AdvertisementID)
	if err != nil || ad == nil || idx >= len(ad.Replies) {
		return nil
	}
	ch, err := s.store.FindChannel(ctx, job.ChannelID)
	if err != nil || ch == nil {
		return err
	}

	mainID := res.LiveMessageIDs[0]
	// ریپلیِ قبلی (اگر بود) را پاک کن — همیشه فقط یک ریپلی هم‌زمان زنده است.
	if len(res.LiveMessageIDs) > 1 {
		_ = s.bot.Delete(&tele.Message{ID: res.LiveMessageIDs[1], Chat: &tele.Chat{ID: ch.TelegramID}})
	}

	reply := ad.Replies[idx]
	m, err := s.bot.Copy(&tele.Chat{ID: ch.TelegramID},
		&tele.Message{ID: reply.MessageID, Chat: &tele.Chat{ID: ad.SourceChatID}},
		&tele.SendOptions{ReplyTo: &tele.Message{ID: mainID}})
	if err != nil {
		return err
	}
	_ = s.store.SetReservationReplyPosted(ctx, resID, m.ID, idx)
	s.afterPost(ctx, ch, m.ID)

	now := time.Now()
	dur := time.Duration(reply.DurationMinutes) * time.Minute
	if idx+1 < len(ad.Replies) {
		_ = s.store.CreateJob(ctx, &models.ScheduledJob{
			Type:            models.JobTypeSendReply,
			CampaignID:      job.CampaignID,
			AdvertisementID: ad.ID,
			ChannelID:       job.ChannelID,
			RunAt:           now.Add(dur),
			Payload:         resID + "|" + strconv.Itoa(idx+1),
		})
	} else {
		// آخرین ریپلی بود: وقتی مدتش تمام شد کل چرخه پاک شود.
		deleteAt := now.Add(dur)
		_ = s.store.CreateJob(ctx, &models.ScheduledJob{
			Type:       models.JobTypeDeletePost,
			CampaignID: job.CampaignID,
			ChannelID:  job.ChannelID,
			RunAt:      deleteAt,
			Payload:    resID,
		})
		_ = s.store.SetReservationDeleteAt(ctx, resID, deleteAt)
	}
	return nil
}

// splitReplyPayload "<resID>|<index>" را تجزیه می‌کند.
func splitReplyPayload(p string) (string, int, bool) {
	i := strings.LastIndex(p, "|")
	if i < 0 {
		return "", 0, false
	}
	idx, err := strconv.Atoi(p[i+1:])
	if err != nil {
		return "", 0, false
	}
	return p[:i], idx, true
}

// deletePost پایانِ یک چرخه است — چه به‌خاطر رسیدنِ سقفِ کل چرخه، چه
// به‌خاطر تمام‌شدنِ آخرین ریپلی. Payload = resID.
func (s *Scheduler) deletePost(ctx context.Context, job models.ScheduledJob) error {
	if job.Payload == "" {
		return nil
	}
	return s.endCycle(ctx, job.Payload, job.ChannelID)
}

// endCycle پیام‌های زنده‌ی یک رزرو را پاک می‌کند، آن را «تمام‌شده» علامت
// می‌زند، و هر jobِ بازمانده‌ی همان رزرو (ریپلیِ بعدی یا حذفِ دیگر) را لغو
// می‌کند تا دوباره اجرا نشود.
func (s *Scheduler) endCycle(ctx context.Context, resID, channelID string) error {
	res, err := s.store.FindReservation(ctx, resID)
	if err != nil || res == nil {
		return nil
	}
	if ch, err := s.store.FindChannel(ctx, channelID); err == nil && ch != nil {
		for _, mid := range res.LiveMessageIDs {
			_ = s.bot.Delete(&tele.Message{ID: mid, Chat: &tele.Chat{ID: ch.TelegramID}})
		}
	}
	_ = s.store.MarkReservationExpired(ctx, resID)
	_ = s.store.CancelReservationJobs(ctx, resID)
	return nil
}

// afterPost بعد از هر ارسال موفق (پست اصلی یا ریپلی) در یک کانال صدا زده
// می‌شود: اگر تبلیغِ «ثابت»ی (KeepAsLastMessage) در همان کانال هست که
// دیگر آخرین پیام نیست، دوباره فرستاده می‌شود (و طبق تنظیمش پین/حذفِ
// نسخه‌ی قبلی انجام می‌شود).
//
// این منطق فرض می‌کند فقط همین ربات در کانال پست می‌گذارد؛ اگر چند تبلیغِ
// ثابت هم‌زمان باشند، اولویت/ترتیبشان از طریق نمای زمان‌بندی (بخش ۲.۷ در
// FEATURES_V2.md) به‌صورت دستی تنظیم می‌شود، نه با قاعده‌ی خودکار اینجا.
func (s *Scheduler) afterPost(ctx context.Context, ch *models.Channel, justPostedMsgID int) {
	live, err := s.store.ListLiveReservationsByChannel(ctx, ch.ID)
	if err != nil {
		return
	}
	for _, res := range live {
		if len(res.LiveMessageIDs) == 0 {
			continue
		}
		currentTop := res.LiveMessageIDs[len(res.LiveMessageIDs)-1]
		if currentTop == justPostedMsgID {
			continue // خودش همین الان تازه‌ترین پیام است
		}
		ad, err := s.store.FindAd(ctx, res.AdvertisementID)
		if err != nil || ad == nil || !ad.KeepAsLastMessage {
			continue
		}
		srcMsgID := ad.MainMessageID
		if res.CurrentReplyIndex >= 0 && res.CurrentReplyIndex < len(ad.Replies) {
			srcMsgID = ad.Replies[res.CurrentReplyIndex].MessageID
		}
		newMsg, err := s.bot.Copy(&tele.Chat{ID: ch.TelegramID},
			&tele.Message{ID: srcMsgID, Chat: &tele.Chat{ID: ad.SourceChatID}})
		if err != nil {
			continue
		}
		if ad.DeletePreviousOnRepost {
			for _, mid := range res.LiveMessageIDs {
				_ = s.bot.Delete(&tele.Message{ID: mid, Chat: &tele.Chat{ID: ch.TelegramID}})
			}
		}
		if ad.PinMessage {
			_ = s.bot.Pin(&tele.Message{ID: newMsg.ID, Chat: &tele.Chat{ID: ch.TelegramID}})
		}
		_ = s.store.SetReservationLiveMessage(ctx, res.ID, []int{newMsg.ID})
	}
}

func (s *Scheduler) updateStats(ctx context.Context, _ models.ScheduledJob) error {
	channels, err := s.store.ListChannels(ctx, models.ChannelActive, 1, 200)
	if err != nil {
		return err
	}
	for _, ch := range channels {
		n, err := s.bot.Len(&tele.Chat{ID: ch.TelegramID})
		if err != nil {
			continue
		}
		_ = s.store.UpdateChannelStats(ctx, ch.ID, n, ch.AvgViews, ch.EngageRate)
	}
	return nil
}

func (s *Scheduler) endCampaign(ctx context.Context, job models.ScheduledJob) error {
	if job.CampaignID == "" {
		return nil
	}
	if err := s.store.UpdateCampaignStatus(ctx, job.CampaignID, models.CampaignCompleted); err != nil {
		return err
	}
	s.log.Info("campaign ended", ports.F("campaign_id", job.CampaignID))
	return nil
}

// sendReminder پیش از ارسال واقعیِ یک تبلیغ (طبق BotSettings.ReminderMinutesBefore)
// به ادمین در پیام خصوصی اطلاع می‌دهد. Payload = resID.
func (s *Scheduler) sendReminder(ctx context.Context, job models.ScheduledJob) error {
	if s.ownerID == 0 {
		return nil
	}
	ad, err := s.store.FindAd(ctx, job.AdvertisementID)
	if err != nil || ad == nil {
		return nil
	}
	ch, err := s.store.FindChannel(ctx, job.ChannelID)
	if err != nil || ch == nil {
		return nil
	}
	res, err := s.store.FindReservation(ctx, job.Payload)
	if err != nil || res == nil {
		return nil
	}
	text := "⏰ یادآوری ارسال تبلیغ\n\n" +
		"ساعت " + res.ScheduledAt.Format("15:04") + " تبلیغ «" + ad.Name + "» در کانال «" + ch.Title + "» ارسال می‌شود."
	_, err = s.bot.Send(&tele.User{ID: s.ownerID}, text)
	return err
}

// ── helpers ──────────────────────────────────────────────────────
//
// محاسبات خالصِ زمان‌بندی (بازه‌ی روزانه، چرخش) در internal/models/schedule.go
// هستند تا هم اینجا و هم در نمای زمان‌بندی (internal/tgbot/handler_schedule.go)
// یک‌جا استفاده و تست شوند — نگاه کنید به internal/models/schedule_test.go.
