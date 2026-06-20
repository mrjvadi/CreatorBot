// Package lock exposes the HTTP API used by uploader-bots to verify membership.
// Depends only on ports.Cache for the job queue — swap cache adapter in main.go.
package lock

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/member-bot/internal/worker"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

const checkTimeout = 8 * time.Second

// resultCacheTTL مدت زمانی که نتیجه‌ی عضویت معتبر می‌ماند. در این بازه،
// درخواست تکراری برای همان (channel, user) بدون صف‌کردن job جدید و بدون
// زدن به API تلگرام، همان لحظه از کش پاسخ می‌گیرد.
// عضویت به‌ندرت تغییر می‌کند (کاربر معمولاً عضو می‌ماند) پس مدت طولانی است.
const resultCacheTTL = 72 * time.Hour

// negativeCacheTTL برای حالت "عضو نیست" کوتاه‌تر است — چون ممکن است
// کاربر دقیقاً همین حالا در فرایند جوین‌شدن باشد و نباید جواب قدیمی
// "عضو نیست" را برایش نگه داریم. ولی برای کاهش فشار روی API تلگرام،
// به‌اندازه‌ی کافی بزرگ است که همان کاربر در چند دقیقه‌ی آینده دوباره
// API تلگرام را صدا نزند.
const negativeCacheTTL = 15 * time.Minute

type Server struct {
	cache  ports.Cache
	log    ports.Logger
	port   int
	apiKey string
}

func NewServer(cache ports.Cache, log ports.Logger, port int, apiKey string) *Server {
	return &Server{cache: cache, log: log, port: port, apiKey: apiKey}
}

func (s *Server) Start() error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(s.authMiddleware())

	r.POST("/check", s.handleCheck)
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	s.log.Info("lock api listening", ports.F("port", s.port))
	return r.Run(fmt.Sprintf(":%d", s.port))
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-API-Key") != s.apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "unauthorized"})
			return
		}
		c.Next()
	}
}

// resultCacheKey کلید کش نتیجه‌ی عضویت برای یک (channel, user) خاص.
func resultCacheKey(channelID, userID int64) string {
	return fmt.Sprintf("memberbot:result:%d:%d", channelID, userID)
}

// CheckMembership منطق اصلی چک عضویت — مشترک بین HTTP handler و NATS
// responder. اول کش، نبود صف Redis Stream برای چک واقعی، بعد کش کردن نتیجه.
func CheckMembership(ctx context.Context, cache ports.Cache, log ports.Logger, channelID, userID int64) (isMember bool, cached bool, err error) {
	cacheKey := resultCacheKey(channelID, userID)
	if val, gerr := cache.Get(ctx, cacheKey); gerr == nil && val != "" {
		return val == "1", true, nil
	}

	jobID := uuid.New().String()
	replyKey := "memberbot:reply:" + jobID
	job := worker.CheckJob{
		JobID:      jobID,
		ChannelID:  channelID,
		UserID:     userID,
		ReplyKey:   replyKey,
		EnqueuedAt: time.Now(),
	}

	if eerr := worker.Enqueue(ctx, cache, job); eerr != nil {
		log.Error("enqueue failed", ports.F("err", eerr))
		return false, false, fmt.Errorf("enqueue failed: %w", eerr)
	}

	result, werr := worker.WaitResult(ctx, cache, replyKey, checkTimeout)
	if werr != nil {
		return false, false, fmt.Errorf("check timed out: %w", werr)
	}
	if result.Err != "" {
		return false, false, fmt.Errorf("%s", result.Err)
	}

	ttl := negativeCacheTTL
	val := "0"
	if result.IsMember {
		ttl = resultCacheTTL
		val = "1"
	}
	_ = cache.Set(ctx, cacheKey, val, ttl)

	return result.IsMember, false, nil
}

// POST /check  { "user_id": 123, "channel_id": -100xxx }
func (s *Server) handleCheck(c *gin.Context) {
	var req struct {
		UserID    int64 `json:"user_id"    binding:"required"`
		ChannelID int64 `json:"channel_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": err.Error()})
		return
	}

	isMember, cached, err := CheckMembership(c.Request.Context(), s.cache, s.log, req.ChannelID, req.UserID)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"ok": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "member": isMember, "cached": cached})
}
