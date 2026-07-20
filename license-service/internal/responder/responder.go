// Package responder سرویس لایسنس را به‌عنوان NATS responder راه می‌اندازد.
package responder

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mrjvadi/creatorbot/license-service/internal/licensing"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Responder struct {
	nc  *natsclient.Client
	svc *licensing.Service
	log ports.Logger

	// serviceHMACSecret همان راز مشترکی است که botpay هم برای احراز
	// service_key استفاده می‌کند — فقط کنترل‌پلین‌های مجاز (agentmanager،
	// botmanager و apimanager) اجازه license.issue/license.revoke دارند.
	serviceHMACSecret string
}

func New(nc *natsclient.Client, svc *licensing.Service, log ports.Logger, serviceHMACSecret string) *Responder {
	return &Responder{nc: nc, svc: svc, log: log, serviceHMACSecret: serviceHMACSecret}
}

func (r *Responder) authorizeService(serviceID, serviceKey string) bool {
	if r.serviceHMACSecret == "" {
		return false // fail-closed
	}
	switch serviceID {
	case "agentmanager", "botmanager", "apimanager":
		return auth.ValidateServiceKey(r.serviceHMACSecret, serviceID, serviceKey)
	default:
		return false
	}
}

func (r *Responder) Start() error {
	if err := r.nc.QueueRespond(protocol.SubjLicenseIssue, protocol.SubjLicenseQueue, r.handleIssue); err != nil {
		return err
	}
	if err := r.nc.QueueRespond(protocol.SubjLicenseVerify, protocol.SubjLicenseQueue, r.handleVerify); err != nil {
		return err
	}
	if err := r.nc.QueueRespond(protocol.SubjLicenseRevoke, protocol.SubjLicenseQueue, r.handleRevoke); err != nil {
		return err
	}
	r.log.Info("license responder started — listening on license.* subjects")
	return nil
}

func (r *Responder) handleIssue(data []byte) (any, error) {
	var req protocol.LicenseIssueRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return protocol.LicenseIssueResponse{Error: "bad request"}, nil
	}
	if !r.authorizeService(req.ServiceID, req.ServiceKey) {
		r.log.Warn("license.issue: unauthorized", ports.F("service", req.ServiceID))
		return protocol.LicenseIssueResponse{Error: "unauthorized"}, nil
	}
	if req.BotID == 0 {
		return protocol.LicenseIssueResponse{Error: "bot_id required"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	token, err := r.svc.Issue(ctx, req)
	if err != nil {
		return protocol.LicenseIssueResponse{Error: err.Error()}, nil
	}
	return protocol.LicenseIssueResponse{Success: true, Token: token}, nil
}

func (r *Responder) handleVerify(data []byte) (any, error) {
	var req protocol.LicenseVerifyRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return protocol.LicenseVerifyResponse{Error: "bad request"}, nil
	}
	if req.BotID == 0 || req.Token == "" {
		return protocol.LicenseVerifyResponse{Error: "bot_id and token required"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	valid, status, clone, err := r.svc.Verify(ctx, req)
	if err != nil {
		return protocol.LicenseVerifyResponse{Valid: false, Error: err.Error()}, nil
	}
	return protocol.LicenseVerifyResponse{Valid: valid, Status: status, CloneWarning: clone}, nil
}

func (r *Responder) handleRevoke(data []byte) (any, error) {
	var req protocol.LicenseRevokeRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return protocol.LicenseRevokeResponse{Error: "bad request"}, nil
	}
	if !r.authorizeService(req.ServiceID, req.ServiceKey) {
		r.log.Warn("license.revoke: unauthorized", ports.F("service", req.ServiceID))
		return protocol.LicenseRevokeResponse{Error: "unauthorized"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.svc.Revoke(ctx, req.BotID, req.Reason); err != nil {
		return protocol.LicenseRevokeResponse{Error: err.Error()}, nil
	}
	return protocol.LicenseRevokeResponse{Success: true}, nil
}
