package engine

import (
	"context"
	"fmt"
	"math"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/ads-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Analyzer تحلیل ممبرهای کانال و محاسبه امتیاز.
type Analyzer struct {
	bot   *tele.Bot
	store *store.Store
	log   ports.Logger
}

func NewAnalyzer(bot *tele.Bot, st *store.Store, log ports.Logger) *Analyzer {
	return &Analyzer{bot: bot, store: st, log: log}
}

// AnalyzeChannel کانال را تحلیل می‌کند و امتیاز آن را محاسبه می‌کند.
func (a *Analyzer) AnalyzeChannel(ctx context.Context, ch *store.AdChannel) (*ChannelAnalysisResult, error) {
	a.log.Info("analyzing channel",
		ports.F("channel_id", ch.ChannelID),
		ports.F("name", ch.ChannelName))

	chat := &tele.Chat{ID: ch.ChannelID}

	// دریافت اطلاعات کانال
	chatInfo, err := a.bot.ChatByID(ch.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("get chat info: %w", err)
	}

	memberCount := chatInfo.MembersCount
	_ = chat

	// ── تحلیل sample از ممبرها ──────────────────────────────
	// نمی‌توانیم همه ممبرها را بگیریم — sample می‌گیریم
	// از طریق bot.ChatMembers (محدود به admin ها در telegram API)
	// به جایش از heuristic استفاده می‌کنیم

	result := &ChannelAnalysisResult{
		ChannelID:   ch.ChannelID,
		MemberCount: memberCount,
	}

	// ── Heuristic scoring ────────────────────────────────────
	score := 50 // امتیاز پایه

	// فاکتور ۱: تعداد ممبر
	switch {
	case memberCount < 100:
		score -= 20 // کانال خیلی کوچک
	case memberCount < 1000:
		score += 0
	case memberCount < 10000:
		score += 10
	case memberCount < 100000:
		score += 20
	default:
		score += 25
	}

	// فاکتور ۲: نسبت View به ممبر (اگه داریم)
	// در Telegram API عمومی نیست — skip

	// فاکتور ۳: username داشتن
	if chatInfo.Username != "" {
		score += 10
	}

	// فاکتور ۴: توضیحات داشتن
	if chatInfo.Description != "" {
		score += 5
	}

	// فاکتور ۵: linked group داشتن
	if chatInfo.LinkedChatID != 0 {
		score += 10
	}

	// نرمال کردن به 0-100
	score = clamp(score, 0, 100)

	// ── تخمین fake percent ───────────────────────────────────
	// بر اساس pattern های شناخته‌شده
	fakePercent := estimateFakePercent(memberCount, chatInfo.Username != "")

	// ── محاسبه ممبر real ─────────────────────────────────────
	realMembers := int(float64(memberCount) * (1.0 - fakePercent/100.0))

	result.Score = score
	result.FakePercent = fakePercent
	result.RealMembers = realMembers
	result.AnalyzedAt = time.Now()

	a.log.Info("channel analyzed",
		ports.F("channel", ch.ChannelID),
		ports.F("score", score),
		ports.F("fake_pct", fakePercent),
		ports.F("real_members", realMembers))

	return result, nil
}

// ChannelAnalysisResult نتیجه تحلیل.
type ChannelAnalysisResult struct {
	ChannelID   int64
	MemberCount int
	Score       int
	FakePercent float64
	RealMembers int
	AnalyzedAt  time.Time
}

// AnalyzeMember یک ممبر را از نظر fake بودن بررسی می‌کند.
func (a *Analyzer) AnalyzeMember(ctx context.Context, channelID, userID int64) *store.MemberAnalysis {
	analysis := &store.MemberAnalysis{
		ChannelID:  channelID,
		TelegramID: userID,
		AnalyzedAt: time.Now(),
	}

	// دریافت اطلاعات کاربر
	user, err := a.bot.ChatByID(userID)
	if err != nil {
		// نمی‌توانیم اطلاعات بگیریم → احتمال fake بالا
		analysis.RealScore = 20
		analysis.IsFake = true
		return analysis
	}

	score := 0

	// فاکتور ۱: username داشتن (+20)
	if user.Username != "" {
		analysis.HasUsername = true
		score += 20
	}

	// فاکتور ۲: عکس پروفایل (+20)
	// نمی‌توانیم مستقیم بگیریم ولی اگه PhotoID داشت
	if user.Photo != nil {
		analysis.HasProfilePhoto = true
		score += 20
	}

	// فاکتور ۳: bot نبودن (+30)
	if !user.IsBot() {
		score += 30
	} else {
		analysis.IsBot = true
	}

	// فاکتور ۴: نام داشتن (+15)
	if user.FirstName != "" && len(user.FirstName) > 1 {
		score += 15
	}

	// فاکتور ۵: نام خانوادگی (+15)
	if user.LastName != "" {
		score += 15
	}

	analysis.RealScore = clamp(score, 0, 100)
	analysis.IsFake = analysis.RealScore < 40

	return analysis
}

// ComputeEffectiveCPJ CPJ موثر کانال را محاسبه می‌کند.
//
// فرمول:
//
//	EffectiveCPJ = BaseCPJ × CategoryMultiplier × (1 - fake_ratio) × score_factor
func ComputeEffectiveCPJ(baseCPJ, categoryMultiplier, fakePercent float64, score int) float64 {
	fakeRatio := fakePercent / 100.0
	scoreFactor := 0.5 + (float64(score)/100.0)*0.5 // 0.5 تا 1.0
	return baseCPJ * categoryMultiplier * (1.0 - fakeRatio) * scoreFactor
}

// ── helpers ────────────────────────────────────────────────

func clamp(v, min, max int) int {
	if v < min { return min }
	if v > max { return max }
	return v
}

// estimateFakePercent تخمین درصد fake از روی heuristic.
func estimateFakePercent(memberCount int, hasUsername bool) float64 {
	// کانال‌های بدون username معمولاً fake بیشتری دارند
	base := 15.0
	if !hasUsername {
		base = 35.0
	}

	// کانال‌های خیلی کوچک یا خیلی بزرگ با jump ناگهانی
	// (نمی‌توانیم تاریخچه رشد را ببینیم — از اندازه تخمین می‌زنیم)
	switch {
	case memberCount < 500:
		base += 10
	case memberCount > 100000:
		// کانال‌های بزرگ معمولاً legitimate ترند
		base = math.Max(base-10, 5)
	}

	return math.Min(base, 80) // حداکثر ۸۰٪
}
