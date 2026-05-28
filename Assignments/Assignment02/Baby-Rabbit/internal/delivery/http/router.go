package http

import "github.com/gin-gonic/gin"

// NewRouter wires the HTTP routes to the given Handler.
// Keeping route declarations here (rather than in main) keeps the
// delivery layer self-contained.
func NewRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	r.POST("/queues", h.CreateQueue)
	r.GET("/queues", h.ListQueues)
	r.GET("/queues/:queue", h.Status)
	r.POST("/queues/:queue/push", h.Push)
	r.POST("/queues/:queue/pop", h.Pop)

	return r
}
