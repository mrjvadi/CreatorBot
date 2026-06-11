package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/source-service/internal/store"
	"github.com/mrjvadi/creatorbot/source-service/internal/userbot"
)

type Server struct {
	store   *store.Store
	cache   ports.Cache
	userbot *userbot.Userbot
	log     ports.Logger
	port    int
	apiKey  string
}

func NewServer(st *store.Store, cache ports.Cache, ub *userbot.Userbot, log ports.Logger, port int, apiKey string) *Server {
	return &Server{store: st, cache: cache, userbot: ub, log: log, port: port, apiKey: apiKey}
}

func (s *Server) Start(ctx context.Context) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		if c.GetHeader("X-API-Key") != s.apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false})
			return
		}
		c.Next()
	})
	r.POST("/files", s.handleRegister)
	r.GET("/files/:uuid", s.handleGet)
	r.POST("/files/:uuid/cache", s.handleCacheFileID)
	return r.Run(fmt.Sprintf(":%d", s.port))
}

func (s *Server) handleRegister(c *gin.Context)    { c.JSON(200, gin.H{"ok": true, "uuid": "TODO"}) }
func (s *Server) handleGet(c *gin.Context)          { c.JSON(200, gin.H{"ok": true, "file_id": "TODO"}) }
func (s *Server) handleCacheFileID(c *gin.Context)  { c.JSON(200, gin.H{"ok": true}) }
