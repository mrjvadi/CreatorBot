// Package fraudclient کلاینت NATS برای fraud-engine.
// همه سرویس‌ها از این یک کلاینت استفاده می‌کنند.
// ارتباط فقط از طریق NATS — بدون HTTP.
//
// Subjects:
//
//	fraud.user.score.request   → درخواست امتیاز کاربر
//	fraud.user.score.response  → پاسخ امتیاز کاربر
//	fraud.community.score.request  → درخواست امتیاز community
//	fraud.community.score.response → پاسخ امتیاز community
//	fraud.event.join           → اطلاع join به fraud-engine
//	fraud.event.leave          → اطلاع leave به fraud-engine
//	fraud.event.activity       → اطلاع فعالیت
package fraudclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/nats-io/nats.go"
)

// ── Types ────────────────────────────────────────────────

type UserScore struct {
	TelegramID int64  `json:"telegram_id"`
	Score      int    `json:"score"`
	Label      string `json:"label"` // high_risk | suspicious | normal | trusted
	Known      bool   `json:"known"`
}

type CommunityScore struct {
	CommunityID       int64   `json:"community_id"`
	Score             int     `json:"score"`
	RevenueStatus     string  `json:"revenue_status"`
	RevenueMultiplier float64 `json:"revenue_multiplier"`
	Known             bool    `json:"known"`
}

func (u *UserScore) IsFake() bool             { return u.Score < 30 }
func (u *UserScore) IsSuspicious() bool       { return u.Score < 60 }
func (u *UserScore) TrustMultiplier() float64 { return float64(u.Score) / 100.0 }

// ── NATS Subjects ─────────────────────────────────────────

const (
	SubUserScoreReq       = "fraud.user.score.request"
	SubUserScoreResp      = "fraud.user.score.response"
	SubCommunityScoreReq  = "fraud.community.score.request"
	SubCommunityScoreResp = "fraud.community.score.response"
	SubEventJoin          = "fraud.event.join"
	SubEventLeave         = "fraud.event.leave"
	SubEventActivity      = "fraud.event.activity"
	SubEventProfile       = "fraud.event.profile"

	requestTimeout = 3 * time.Second
)

// ── Client ────────────────────────────────────────────────

type Client struct {
	nc *natsclient.Client
}

func New(nc *natsclient.Client) *Client {
	return &Client{nc: nc}
}

// GetUserScore امتیاز کاربر را از fraud-engine می‌خواهد (NATS request/reply).
func (c *Client) GetUserScore(ctx context.Context, telegramID int64) (*UserScore, error) {
	req, _ := json.Marshal(map[string]any{"telegram_id": telegramID})

	msg, err := c.nc.NC().RequestWithContext(ctx, SubUserScoreReq, req)
	if err != nil {
		// fallback — اگه fraud-engine جواب نداد، neutral score
		return &UserScore{TelegramID: telegramID, Score: 60, Label: "normal", Known: false}, nil
	}

	var score UserScore
	if err := json.Unmarshal(msg.Data, &score); err != nil {
		return &UserScore{TelegramID: telegramID, Score: 60, Label: "normal"}, nil
	}
	return &score, nil
}

// GetCommunityScore امتیاز community را می‌خواهد.
func (c *Client) GetCommunityScore(ctx context.Context, chatID int64) (*CommunityScore, error) {
	req, _ := json.Marshal(map[string]any{"community_id": chatID})

	msg, err := c.nc.NC().RequestWithContext(ctx, SubCommunityScoreReq, req)
	if err != nil {
		return &CommunityScore{CommunityID: chatID, Score: 70, RevenueMultiplier: 0.8, Known: false}, nil
	}

	var score CommunityScore
	if err := json.Unmarshal(msg.Data, &score); err != nil {
		return &CommunityScore{CommunityID: chatID, Score: 70, RevenueMultiplier: 0.8}, nil
	}
	return &score, nil
}

// ReportJoin به fraud-engine اطلاع می‌دهد که کاربر join کرده (fire-and-forget).
func (c *Client) ReportJoin(telegramID, communityID int64, source, campaignID string) {
	c.nc.PublishCore(SubEventJoin, map[string]any{
		"telegram_id":  telegramID,
		"community_id": communityID,
		"source":       source,
		"campaign_id":  campaignID,
	})
}

// ReportLeave به fraud-engine اطلاع می‌دهد که کاربر leave کرده.
func (c *Client) ReportLeave(telegramID, communityID int64) {
	c.nc.PublishCore(SubEventLeave, map[string]any{
		"telegram_id":  telegramID,
		"community_id": communityID,
	})
}

// ReportActivity فعالیت کاربر را گزارش می‌دهد.
func (c *Client) ReportActivity(telegramID, communityID int64, messages, replies, reactions int) {
	c.nc.PublishCore(SubEventActivity, map[string]any{
		"telegram_id":  telegramID,
		"community_id": communityID,
		"messages":     messages,
		"replies":      replies,
		"reactions":    reactions,
	})
}

// ReportProfileUpdate تغییر پروفایل را گزارش می‌دهد.
func (c *Client) ReportProfileUpdate(telegramID int64, username, firstName string, hasPhoto bool) {
	c.nc.PublishCore(SubEventProfile, map[string]any{
		"telegram_id": telegramID,
		"username":    username,
		"first_name":  firstName,
		"has_photo":   hasPhoto,
	})
}

// ── Noop برای test/dev ────────────────────────────────────

type NoopClient struct{}

func NewNoop() *NoopClient { return &NoopClient{} }

func (n *NoopClient) GetUserScore(_ context.Context, id int64) (*UserScore, error) {
	return &UserScore{TelegramID: id, Score: 70, Label: "normal"}, nil
}
func (n *NoopClient) GetCommunityScore(_ context.Context, id int64) (*CommunityScore, error) {
	return &CommunityScore{CommunityID: id, Score: 70, RevenueMultiplier: 1.0}, nil
}
func (n *NoopClient) ReportJoin(_, _ int64, _, _ string)               {}
func (n *NoopClient) ReportLeave(_, _ int64)                           {}
func (n *NoopClient) ReportActivity(_, _ int64, _, _, _ int)           {}
func (n *NoopClient) ReportProfileUpdate(_ int64, _, _ string, _ bool) {}

// suppress unused
var _ = fmt.Sprintf
var _ = nats.ErrTimeout
