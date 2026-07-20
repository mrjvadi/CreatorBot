package memberclient

import (
	"context"
	"sync"
	"time"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// CheckBotStatus می‌پرسد آیا این bot (با BotID تلگرامیِ خودش) الان به یک
// کمپینِ اجاره‌ی قفلِ فعال در ads-bot وصل است.
func (c *Client) CheckBotStatus(ctx context.Context, botID int64) (inCampaign bool, campaignID string, err error) {
	var resp protocol.BotStatusResponse
	err = c.nc.Request(ctx, protocol.SubjBotStatusCheck, protocol.BotStatusRequest{BotID: botID}, &resp, c.timeout)
	if err != nil {
		return false, "", err
	}
	if resp.Error != "" {
		return false, "", &memberError{resp.Error}
	}
	return resp.InCampaign, resp.CampaignID, nil
}

// RentalStatus وضعیتِ فعلیِ «آیا این bot به یک کمپینِ اجاره‌ی قفلِ فعال
// وصل است» را نگه می‌دارد — thread-safe چون هم از RunStatusLoop نوشته
// می‌شود هم از handlerهای تلگرام (chat-member/join) خوانده می‌شود.
type RentalStatus struct {
	mu         sync.RWMutex
	inCampaign bool
	campaignID string
}

// IsInCampaign برای گیت‌کردنِ منطقِ مخصوصِ ربات‌های رایگان استفاده می‌شود.
func (r *RentalStatus) IsInCampaign() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.inCampaign
}

// CampaignID برای attribution گزارش‌های fraud-engine — اگر IsInCampaign
// false باشد رشته‌ی خالی برمی‌گردد.
func (r *RentalStatus) CampaignID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.campaignID
}

func (r *RentalStatus) set(inCampaign bool, campaignID string) {
	r.mu.Lock()
	r.inCampaign, r.campaignID = inCampaign, campaignID
	r.mu.Unlock()
}

// RunStatusLoop وضعیتِ اجاره را همین الان (startup) و بعد هر ۵ دقیقه یک‌بار
// از ads-bot می‌پرسد — معادلِ همان الگویِ licenseclient.RunLicenseLoop برای
// check-in دوره‌ای؛ fail-open (خطای شبکه یعنی آخرین وضعیتِ شناخته‌شده حفظ
// می‌شود، نه ریست به false).
func RunStatusLoop(ctx context.Context, nc *natsclient.Client, botID int64, status *RentalStatus, log ports.Logger) {
	c := New(nc)
	check := func() {
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		inCampaign, campaignID, err := c.CheckBotStatus(cctx, botID)
		if err != nil {
			log.Warn("ads-bot status check-in failed (fail-open, keeping previous status)",
				ports.F("bot_id", botID), ports.F("err", err))
			return
		}
		status.set(inCampaign, campaignID)
		log.Info("ads-bot status check-in",
			ports.F("bot_id", botID), ports.F("in_campaign", inCampaign))
	}
	check()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}
