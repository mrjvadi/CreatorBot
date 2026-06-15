// Package api REST API برای سرویس‌های دیگه پلتفرم.
// هر سرویس با API Key احراز هویت می‌شود.
//
// Endpoints:
//   POST /api/v1/pay/balance          → موجودی کاربر
//   POST /api/v1/pay/deduct           → کسر از موجودی
//   POST /api/v1/pay/invoice/create   → ساخت invoice واریز
//   POST /api/v1/pay/invoice/status   → وضعیت invoice
//   POST /api/v1/pay/credit/add       → افزایش اعتبار (ادمین)
package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mrjvadi/creatorbot/botpay/internal/wallet"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Config تنظیمات API.
type Config struct {
	// ServiceKeys map از service_id → api_key
	// هر سرویس یک key اختصاصی دارد
	ServiceKeys map[string]string
	AdminKey    string
}

type Handler struct {
	wallet  *wallet.Service
	store   interface{} // botpay store
	cfg     Config
	log     ports.Logger
}

func New(w *wallet.Service, cfg Config, log ports.Logger) *Handler {
	return &Handler{wallet: w, cfg: cfg, log: log}
}

// Register route ها را ثبت می‌کند.
func (h *Handler) Register(r *gin.Engine) {
	api := r.Group("/api/v1/pay")
	api.Use(h.authMiddleware())

	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "service": "botpay"})
	})
	api.POST("/balance",         h.getBalance)
	api.POST("/deduct",          h.deduct)
	api.POST("/invoice/create",  h.createInvoice)
	api.POST("/invoice/status",  h.invoiceStatus)
	api.POST("/credit/add",      h.addCredit)   // فقط admin
	api.POST("/transfer",        h.transfer)
}

// ── Auth middleware ────────────────────────────────────────

func (h *Handler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		serviceID := c.GetHeader("X-Service-ID")

		if key == "" || serviceID == "" {
			fail(c, http.StatusUnauthorized, "missing credentials")
			return
		}

		// بررسی کلید سرویس
		expectedKey, ok := h.cfg.ServiceKeys[serviceID]
		if !ok || key != expectedKey {
			// بررسی admin key
			if key != h.cfg.AdminKey {
				fail(c, http.StatusUnauthorized, "invalid api key")
				return
			}
			c.Set("is_admin", true)
		}

		c.Set("service_id", serviceID)
		c.Next()
	}
}

// ── Handlers ──────────────────────────────────────────────

// POST /balance — موجودی کاربر
// Body: { "telegram_id": 123 }
func (h *Handler) getBalance(c *gin.Context) {
	var req struct {
		TelegramID int64 `json:"telegram_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	w, err := h.wallet.GetOrCreate(ctx, req.TelegramID)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{
		"telegram_id":  req.TelegramID,
		"ton_balance":  w.BalanceTON(),
		"credit":       w.CreditTON(),
		"total":        w.TotalTON(),
		"frozen":       wallet.NanoToTON(w.Frozen),
		"ton_address":  w.TONAddress,
	})
}

// POST /deduct — کسر از موجودی
// Body: { "telegram_id": 123, "amount_ton": 1.5, "ref": "plan_abc", "description": "خرید پلن" }
func (h *Handler) deduct(c *gin.Context) {
	serviceID := c.GetString("service_id")

	var req struct {
		TelegramID  int64   `json:"telegram_id" binding:"required"`
		AmountTON   float64 `json:"amount_ton" binding:"required"`
		Ref         string  `json:"ref"`
		Description string  `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	amountNano := wallet.TONToNano(req.AmountTON)
	ctx := c.Request.Context()

	tx, err := h.wallet.Pay(ctx, req.TelegramID, amountNano, serviceID, req.Ref, req.Description)
	if err != nil {
		if err.Error()[:13] == "insufficient" {
			fail(c, http.StatusPaymentRequired, err.Error())
			return
		}
		h.log.Error("deduct failed", ports.F("err", err))
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{
		"tx_id":      tx.ID.String(),
		"amount_ton": req.AmountTON,
		"ref":        req.Ref,
	})
}

// POST /invoice/create — ساخت invoice برای واریز
// Body: { "telegram_id": 123, "amount_ton": 5.0, "ref": "plan_abc" }
func (h *Handler) createInvoice(c *gin.Context) {
	serviceID := c.GetString("service_id")

	var req struct {
		TelegramID int64   `json:"telegram_id" binding:"required"`
		AmountTON  float64 `json:"amount_ton" binding:"required"`
		Ref        string  `json:"ref"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	amountNano := wallet.TONToNano(req.AmountTON)
	ctx := c.Request.Context()

	code, payURL, err := h.wallet.DepositInstructions(ctx, req.TelegramID, amountNano, serviceID, req.Ref)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{
		"invoice_code": code,
		"pay_url":      payURL,
		"amount_ton":   req.AmountTON,
		"expires_in":   "30m",
		// NATS subject که سرویس باید subscribe کنه تا تأیید را دریافت کند
		"nats_subject": "botpay.invoice." + code,
	})
}

// POST /invoice/status — وضعیت invoice
func (h *Handler) invoiceStatus(c *gin.Context) {
	var req struct{ Code string `json:"code" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ctx := c.Request.Context()
	inv, err := h.wallet.Store().FindInvoiceByCode(ctx, req.Code)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if inv == nil {
		ok(c, gin.H{"code": req.Code, "status": "not_found"})
		return
	}
	ok(c, gin.H{"code": inv.Code, "status": inv.Status, "paid_at": inv.PaidAt})
}

// POST /credit/add — افزایش اعتبار (فقط ادمین)
func (h *Handler) addCredit(c *gin.Context) {
	if !c.GetBool("is_admin") {
		fail(c, http.StatusForbidden, "admin only")
		return
	}

	var req struct {
		TelegramID  int64   `json:"telegram_id" binding:"required"`
		AmountTON   float64 `json:"amount_ton" binding:"required"`
		Description string  `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	w, err := h.wallet.GetOrCreate(ctx, req.TelegramID)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	amountNano := wallet.TONToNano(req.AmountTON)
	if err := h.wallet.Store().AddCredit(ctx, w.ID, amountNano, req.Description); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{"added_ton": req.AmountTON})
}

// POST /transfer — انتقال داخلی
// Body: { "from_telegram_id": 123, "to_telegram_id": 456, "amount_ton": 1.0 }
func (h *Handler) transfer(c *gin.Context) {
	serviceID := c.GetString("service_id")

	var req struct {
		FromTelegramID int64   `json:"from_telegram_id" binding:"required"`
		ToTelegramID   int64   `json:"to_telegram_id" binding:"required"`
		AmountTON      float64 `json:"amount_ton" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	_ = serviceID
	amountNano := wallet.TONToNano(req.AmountTON)
	ctx := c.Request.Context()

	if err := h.wallet.Transfer(ctx, req.FromTelegramID, req.ToTelegramID, amountNano,
		fmt.Sprintf("api-transfer-%s", serviceID)); err != nil {
		if strings.Contains(err.Error(), "insufficient") {
			fail(c, http.StatusPaymentRequired, err.Error())
			return
		}
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{"transferred_ton": req.AmountTON})
}

// ── helpers ────────────────────────────────────────────────

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": data})
}

func fail(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, gin.H{"ok": false, "message": msg})
}
