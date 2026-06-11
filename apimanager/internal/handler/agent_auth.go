package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// AgentAuthRequest درخواست احراز هویت agentmanager.
type AgentAuthRequest struct {
	ServerID string `json:"server_id" binding:"required"`
	APIKey   string `json:"api_key"   binding:"required"`
}

// AgentAuthResponse پاسخ احراز هویت.
type AgentAuthResponse struct {
	ServerID  string `json:"server_id"`
	ExpiresAt int64  `json:"expires_at"`
}

// AgentAuth احراز هویت agentmanager.
// POST /api/v1/agent/auth
func (h *Handler) AgentAuth(c *gin.Context) {
	var req AgentAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// بررسی API Key
	if req.APIKey != h.agentAPIKey {
		h.log.Info("agent auth failed: wrong api key",
			ports.F("server_id", req.ServerID))
		fail(c, http.StatusUnauthorized, "invalid api key")
		return
	}

	// بررسی ServerID در DB
	servers, err := h.store.ListServers(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	var found bool
	for _, s := range servers {
		if s.ID.String() == req.ServerID {
			found = true
			h.store.MarkServerOnline(c.Request.Context(), s.ID)
			break
		}
	}
	if !found {
		fail(c, http.StatusUnauthorized, "server not registered")
		return
	}

	h.log.Info("agent authenticated", ports.F("server_id", req.ServerID))
	ok(c, AgentAuthResponse{
		ServerID:  req.ServerID,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	})
}

// BotAuth احراز هویت bot.
// POST /api/v1/bot/auth
func (h *Handler) BotAuth(c *gin.Context) {
	var req struct {
		BotToken string `json:"bot_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	parts := strings.SplitN(req.BotToken, ":", 2)
	if len(parts) != 2 {
		fail(c, http.StatusBadRequest, "invalid token format")
		return
	}
	var botID int64
	if _, err := fmt.Sscanf(parts[0], "%d", &botID); err != nil {
		fail(c, http.StatusBadRequest, "invalid bot id")
		return
	}

	inst, err := h.store.FindInstanceByBotID(c.Request.Context(), botID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if inst == nil {
		fail(c, http.StatusUnauthorized, "bot not registered")
		return
	}

	apiKey := fmt.Sprintf("%x", fmt.Sprintf("bot_%d_%s", botID, h.encryptKey[:16]))[:32]

	expiresAt := time.Now().Add(24 * time.Hour)
	_ = jwt.New(jwt.SigningMethodHS256) // suppress import

	h.log.Info("bot authenticated", ports.F("bot_id", botID))
	ok(c, gin.H{
		"bot_id":    botID,
		"api_key":   apiKey,
		"expires_at": expiresAt.Unix(),
	})
}
