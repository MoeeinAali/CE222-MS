package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"Baby-Rabbit/internal/domain"
	"Baby-Rabbit/internal/usecase"
)

// Handler adapts HTTP requests to the QueueService inbound port.
// It depends only on the usecase.QueueService interface, never on
// repository implementations — preserving the dependency rule.
type Handler struct {
	svc usecase.QueueService
}

func NewHandler(svc usecase.QueueService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateQueue(c *gin.Context) {
	var req createQueueReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp{Error: err.Error()})
		return
	}
	id, err := h.svc.CreateQueue(req.Name, req.Capacity)
	if err != nil {
		writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, createQueueResp{
		ID:       id,
		Name:     req.Name,
		Capacity: req.Capacity,
	})
}

func (h *Handler) Push(c *gin.Context) {
	queueID := c.Param("queue")
	var req pushReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResp{Error: err.Error()})
		return
	}
	ttl := time.Duration(req.TTL) * time.Second
	if err := h.svc.Push(queueID, req.Value, ttl); err != nil {
		writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "ok"})
}

func (h *Handler) Pop(c *gin.Context) {
	queueID := c.Param("queue")
	msg, err := h.svc.Pop(queueID)
	if err != nil {
		writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, messageResp{
		ID:        msg.ID,
		Value:     msg.Value,
		CreatedAt: msg.CreatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) Status(c *gin.Context) {
	queueID := c.Param("queue")
	s, err := h.svc.Status(queueID)
	if err != nil {
		writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, statusResp{
		ID:       s.ID,
		Name:     s.Name,
		Size:     s.Size,
		Capacity: s.Capacity,
	})
}

func (h *Handler) ListQueues(c *gin.Context) {
	metas := h.svc.ListQueues()
	out := make([]queueListItem, 0, len(metas))
	for _, m := range metas {
		out = append(out, queueListItem{ID: m.ID, Name: m.Name, Capacity: m.Capacity})
	}
	c.JSON(http.StatusOK, out)
}

// writeDomainError maps domain errors to HTTP status codes so the
// inner layers stay transport-agnostic.
func writeDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrQueueNotFound):
		c.JSON(http.StatusNotFound, errorResp{Error: err.Error()})
	case errors.Is(err, domain.ErrQueueAlreadyExists):
		c.JSON(http.StatusConflict, errorResp{Error: err.Error()})
	case errors.Is(err, domain.ErrQueueFull):
		c.JSON(http.StatusConflict, errorResp{Error: err.Error()})
	case errors.Is(err, domain.ErrQueueEmpty):
		c.Status(http.StatusNoContent)
	case errors.Is(err, domain.ErrInvalidCapacity),
		errors.Is(err, domain.ErrInvalidName),
		errors.Is(err, domain.ErrInvalidTTL):
		c.JSON(http.StatusBadRequest, errorResp{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, errorResp{Error: err.Error()})
	}
}
