// Package memberresponder member-bot را به‌عنوان NATS responder متمرکز
// چک عضویت راه‌اندازی می‌کند. bot های فرعی (uploader و...) به‌جای ادمین‌شدن
// در هر کانال، از طریق member.check می‌پرسند «کاربر X عضو کانال Y هست؟».
//
// همان منطق cache-first که در lock.CheckMembership پیاده شده استفاده می‌شود
// (یک منبع حقیقت برای هم HTTP و هم NATS).
package memberresponder

import (
	"context"
	"encoding/json"
	"time"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"

	"github.com/mrjvadi/creatorbot/member-bot/internal/lock"
)

// Responder درخواست‌های member.check را پاسخ می‌دهد.
type Responder struct {
	nc    *natsclient.Client
	cache ports.Cache
	log   ports.Logger
}

func New(nc *natsclient.Client, cache ports.Cache, log ports.Logger) *Responder {
	return &Responder{nc: nc, cache: cache, log: log}
}

// Start با queue group ثبت می‌شود — اگر چند instance از member-bot بالا
// باشد، فقط یکی هر درخواست را پاسخ می‌دهد.
func (r *Responder) Start() error {
	if r.nc == nil {
		return nil
	}
	return r.nc.QueueRespond(protocol.SubjMemberCheck, protocol.SubjMemberQueue, r.handleCheck)
}

func (r *Responder) handleCheck(data []byte) (any, error) {
	var req protocol.MemberCheckRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return protocol.MemberCheckResponse{Error: "bad request"}, nil
	}
	if req.ChannelID == 0 || req.UserID == 0 {
		return protocol.MemberCheckResponse{Error: "channel_id and user_id required"}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	isMember, cached, err := lock.CheckMembership(ctx, r.cache, r.log, req.ChannelID, req.UserID)
	if err != nil {
		return protocol.MemberCheckResponse{Error: err.Error()}, nil
	}

	return protocol.MemberCheckResponse{IsMember: isMember, Cached: cached}, nil
}
