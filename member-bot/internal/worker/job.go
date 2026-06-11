// Package worker implements the distributed membership check system.
//
// Flow:
//  1. Lock API  →  Enqueue(CheckJob) onto Redis stream
//  2. All BotWorkers read via XREADGROUP
//  3. Each worker filters: "am I a member of this channel?" (Redis SISMEMBER)
//  4. Eligible workers race via SETNX claim lock
//  5. Winner calls Telegram getChatMember, writes result, ACKs stream message
//  6. HTTP handler unblocks via BLPOP on reply key
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

const (
	StreamKey     = "memberbot:check:stream"
	ConsumerGroup = "check_bots"
	ClaimTTL      = 5 * time.Second
	JobTimeout    = 10 * time.Second
)

// CheckJob is the payload pushed onto the stream for each membership check request.
type CheckJob struct {
	JobID      string    `json:"job_id"`
	ChannelID  int64     `json:"channel_id"`
	UserID     int64     `json:"user_id"`
	ReplyKey   string    `json:"reply_key"` // BLPOP key for the HTTP handler
	EnqueuedAt time.Time `json:"enqueued_at"`
}

// CheckResult is written to ReplyKey once the job is processed.
type CheckResult struct {
	JobID    string `json:"job_id"`
	IsMember bool   `json:"is_member"`
	BotID    string `json:"bot_id"`
	ReplyKey string `json:"reply_key"` // FIX: was missing, caused compile error in pool.go
	Err      string `json:"err,omitempty"`
}

func BotChannelKey(botID string) string {
	return fmt.Sprintf("memberbot:bot_channels:%s", botID)
}

func claimKey(jobID string) string {
	return fmt.Sprintf("memberbot:claim:%s", jobID)
}

func Enqueue(ctx context.Context, cache ports.Cache, job CheckJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	_, err = cache.XAdd(ctx, StreamKey, map[string]any{"payload": string(data)})
	return err
}

func EnsureGroup(ctx context.Context, cache ports.Cache) error {
	return cache.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "0")
}

func TryClaim(ctx context.Context, cache ports.Cache, jobID, botID string) (bool, error) {
	return cache.SetNX(ctx, claimKey(jobID), botID, ClaimTTL)
}

func WriteResult(ctx context.Context, cache ports.Cache, result CheckResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if err := cache.LPush(ctx, result.ReplyKey, string(data)); err != nil {
		return err
	}
	return cache.Set(ctx, result.ReplyKey+"_ttl", "1", 30*time.Second)
}

func WaitResult(ctx context.Context, cache ports.Cache, replyKey string, timeout time.Duration) (*CheckResult, error) {
	vals, err := cache.BLPop(ctx, timeout, replyKey)
	if err != nil {
		return nil, err
	}
	if len(vals) < 2 {
		return nil, fmt.Errorf("empty result")
	}
	var cr CheckResult
	if err := json.Unmarshal([]byte(vals[1]), &cr); err != nil {
		return nil, err
	}
	return &cr, nil
}
