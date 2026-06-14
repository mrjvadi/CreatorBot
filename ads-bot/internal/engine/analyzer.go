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

type Analyzer struct {
	bot   *tele.Bot
	store *store.Store
	log   ports.Logger
}

func NewAnalyzer(bot *tele.Bot, st *store.Store, log ports.Logger) *Analyzer {
	return &Analyzer{bot: bot, store: st, log: log}
}

type ChannelAnalysisResult struct {
	ChannelID   int64
	MemberCount int
	Score       int
	FakePercent float64
	RealMembers int
	AnalyzedAt  time.Time
}

// AnalyzeChannel کانال را تحلیل و امتیاز آن را محاسبه می‌کند.
func (a *Analyzer) AnalyzeChannel(ctx context.Context, ch *store.AdChannel) (*ChannelAnalysisResult, error) {
	a.log.Info("analyzing channel",
		ports.F("channel_id", ch.ChannelID),
		ports.F("name", ch.ChannelName))

	chatInfo, err := a.bot.ChatByID(ch.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("get chat info: %w", err)
	}

	// دریافت تعداد اعضا
	memberCount, err := a.bot.Len(chatInfo)
	if err != nil {
		memberCount = ch.MemberCount
	}

	score := 50

	switch {
	case memberCount < 100:
		score -= 20
	case memberCount < 1000:
		score += 0
	case memberCount < 10000:
		score += 10
	case memberCount < 100000:
		score += 20
	default:
		score += 25
	}

	if chatInfo.Username != "" {
		score += 10
	}
	if chatInfo.Description != "" {
		score += 5
	}
	if chatInfo.LinkedChatID != 0 {
		score += 10
	}

	score = clamp(score, 0, 100)
	fakePercent := estimateFakePercent(memberCount, chatInfo.Username != "")
	realMembers := int(float64(memberCount) * (1.0 - fakePercent/100.0))

	result := &ChannelAnalysisResult{
		ChannelID:   ch.ChannelID,
		MemberCount: memberCount,
		Score:       score,
		FakePercent: fakePercent,
		RealMembers: realMembers,
		AnalyzedAt:  time.Now(),
	}

	a.log.Info("channel analyzed",
		ports.F("channel", ch.ChannelID),
		ports.F("score", score),
		ports.F("fake_pct", fakePercent))

	return result, nil
}

// AnalyzeMember یک کاربر را از نظر fake بودن بررسی می‌کند.
func (a *Analyzer) AnalyzeMember(ctx context.Context, channelID, userID int64) *store.MemberAnalysis {
	analysis := &store.MemberAnalysis{
		ChannelID:  channelID,
		TelegramID: userID,
		AnalyzedAt: time.Now(),
	}

	user, err := a.bot.ChatByID(userID)
	if err != nil {
		analysis.RealScore = 20
		analysis.IsFake = true
		return analysis
	}

	score := 0

	if user.Username != "" {
		analysis.HasUsername = true
		score += 20
	}
	if user.Photo != nil {
		analysis.HasProfilePhoto = true
		score += 20
	}
	// بررسی نوع چت — user معمولی private هست
	switch user.Type {
	case "private":
		score += 30
	case "bot", "group", "supergroup", "channel":
		analysis.IsBot = true
	default:
		score += 15
	}
	if user.FirstName != "" && len(user.FirstName) > 1 {
		score += 15
	}
	if user.LastName != "" {
		score += 15
	}

	analysis.RealScore = clamp(score, 0, 100)
	analysis.IsFake = analysis.RealScore < 40
	return analysis
}

// ComputeEffectiveCPJ CPJ موثر کانال را محاسبه می‌کند.
func ComputeEffectiveCPJ(baseCPJ, categoryMultiplier, fakePercent float64, score int) float64 {
	fakeRatio := fakePercent / 100.0
	scoreFactor := 0.5 + (float64(score)/100.0)*0.5
	return baseCPJ * categoryMultiplier * (1.0 - fakeRatio) * scoreFactor
}

func clamp(v, min, max int) int {
	if v < min { return min }
	if v > max { return max }
	return v
}

func estimateFakePercent(memberCount int, hasUsername bool) float64 {
	base := 15.0
	if !hasUsername {
		base = 35.0
	}
	switch {
	case memberCount < 500:
		base += 10
	case memberCount > 100000:
		base = math.Max(base-10, 5)
	}
	return math.Min(base, 80)
}
