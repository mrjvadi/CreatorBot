package telegram

import (
	"context"
	"time"

	"github.com/gotd/td/tg"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
)

// NATSCodeSource asks whoever holds this worker's license (usually
// botmanager, relaying a human) to submit the Telegram login code, via a
// NATS request-reply call. Build the subject with
// worker.AuthCodeSubject(workerID).
type NATSCodeSource struct {
	nc      *natsclient.Client
	subject string
	timeout time.Duration
}

func NewNATSCodeSource(nc *natsclient.Client, subject string, timeout time.Duration) *NATSCodeSource {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &NATSCodeSource{nc: nc, subject: subject, timeout: timeout}
}

type codeRequest struct{}

type codeReply struct {
	Code string `json:"code"`
}

// Code blocks (up to timeout) waiting for someone to reply on s.subject with
// {"code": "12345"}.
func (s *NATSCodeSource) Code(ctx context.Context, _ *tg.AuthSentCode) (string, error) {
	var reply codeReply
	if err := s.nc.Request(ctx, s.subject, codeRequest{}, &reply, s.timeout); err != nil {
		return "", err
	}
	return reply.Code, nil
}
