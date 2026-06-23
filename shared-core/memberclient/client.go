// Package memberclient کلاینت مشترک bot های فرعی برای دو کار:
//  1. چک عضویت کاربر در کانال (از طریق member-bot، نه ادمین‌شدن مستقیم)
//  2. اطلاع به ads-bot که در کانال خریدار ادمین شدند (شروع قفل‌کردن)
package memberclient

import (
	"context"
	"time"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
)

type Client struct {
	nc      *natsclient.Client
	timeout time.Duration
}

func New(nc *natsclient.Client) *Client {
	return &Client{nc: nc, timeout: 10 * time.Second}
}

// IsMember بررسی می‌کند کاربر عضو کانال هست یا نه — از مسیر متمرکز
// member-bot (با کش)، نه با ادمین‌شدن مستقیم ربات در آن کانال.
func (c *Client) IsMember(ctx context.Context, channelID, userID int64) (bool, error) {
	var resp protocol.MemberCheckResponse
	err := c.nc.Request(ctx, protocol.SubjMemberCheck, protocol.MemberCheckRequest{
		ChannelID: channelID, UserID: userID,
	}, &resp, c.timeout)
	if err != nil {
		return false, err
	}
	if resp.Error != "" {
		return false, &memberError{resp.Error}
	}
	return resp.IsMember, nil
}

// ConfirmChannelAdmin به ads-bot اطلاع می‌دهد این bot (با BotID خودش) در
// کانال خریدار ادمین شده است. از این لحظه قفل‌کردن برای آن تبلیغ شروع می‌شود.
func (c *Client) ConfirmChannelAdmin(ctx context.Context, botID int64) error {
	var resp protocol.ConfirmChannelAdminResponse
	err := c.nc.Request(ctx, protocol.SubjConfirmChannelAdmin, protocol.ConfirmChannelAdminRequest{
		BotID: botID,
	}, &resp, c.timeout)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return &memberError{resp.Error}
	}
	return nil
}

type memberError struct{ msg string }

func (e *memberError) Error() string { return e.msg }
