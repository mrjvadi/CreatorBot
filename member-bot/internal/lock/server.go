// Package lock exposes the HTTP API used by uploader-bots to verify membership.
// Depends only on ports.Cache for the job queue — swap cache adapter in main.go.
package lock

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/member-bot/internal/worker"
)

const checkTimeout = 8 * time.Second

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

	jobID := uuid.New().String()
	replyKey := "memberbot:reply:" + jobID

	job := worker.CheckJob{
		JobID:      jobID,
		ChannelID:  req.ChannelID,
		UserID:     req.UserID,
		ReplyKey:   replyKey,
		EnqueuedAt: time.Now(),
	}

	if err := worker.Enqueue(c.Request.Context(), s.cache, job); err != nil {
		s.log.Error("enqueue failed", ports.F("err", err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "enqueue failed"})
		return
	}

	result, err := worker.WaitResult(c.Request.Context(), s.cache, replyKey, checkTimeout)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"ok": false, "message": "check timed out"})
		return
	}
	if result.Err != "" {
		c.JSON(http.StatusOK, gin.H{"ok": false, "message": result.Err})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "member": result.IsMember})
}
