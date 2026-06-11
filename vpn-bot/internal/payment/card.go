// Package payment contains the card-to-card payment adapter for vpn-bot.
// Card payment is bot-specific (receipt image + admin confirm),
// so it lives here instead of shared/pkg/adapters.
package payment

import (
	"context"
	"fmt"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// CardGateway implements ports.PaymentGateway for manual card-to-card payments.
// Flow: user sends a receipt photo → admin confirms → balance is added.
type CardGateway struct {
	cardNumber string
	cardOwner  string
	sender     ports.BotSender
	adminID    int64
}

var _ ports.PaymentGateway = (*CardGateway)(nil)

func NewCardGateway(cardNumber, cardOwner string, sender ports.BotSender, adminID int64) *CardGateway {
	return &CardGateway{cardNumber: cardNumber, cardOwner: cardOwner, sender: sender, adminID: adminID}
}

func (g *CardGateway) Name() string { return "card" }

// CreatePayment sends the card details to the user so they can transfer.
func (g *CardGateway) CreatePayment(ctx context.Context, req ports.PaymentRequest) (*ports.PaymentResponse, error) {
	msg := fmt.Sprintf(
		"💳 پرداخت کارت‌به‌کارت\n\nمبلغ: <b>%g تومان</b>\nشماره کارت: <code>%s</code>\nصاحب حساب: %s\n\nپس از پرداخت تصویر رسید را ارسال کنید.",
		req.Amount, g.cardNumber, g.cardOwner,
	)
	if err := g.sender.Send(ctx, req.UserID, msg, ports.WithHTML()); err != nil {
		return nil, err
	}
	return &ports.PaymentResponse{RefID: req.OrderID}, nil
}

// VerifyPayment is called by the admin after checking the receipt.
// refID here is the orderID stored when CreatePayment was called.
func (g *CardGateway) VerifyPayment(_ context.Context, refID string) (*ports.VerifyResponse, error) {
	// Admin confirms via bot command — this is a no-op in the gateway layer.
	return &ports.VerifyResponse{RefID: refID, Success: true}, nil
}
