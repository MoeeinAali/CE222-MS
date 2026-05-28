package usecase

import (
	"strings"
	"time"

	"Baby-Rabbit/internal/domain"
)

type QueueUseCase struct {
	manager domain.QueueManager
	ids     IDGenerator
	clock   Clock
}

func NewQueueUseCase(m domain.QueueManager, ids IDGenerator, clock Clock) *QueueUseCase {
	return &QueueUseCase{manager: m, ids: ids, clock: clock}
}

func (u *QueueUseCase) CreateQueue(name string, capacity int) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", domain.ErrInvalidName
	}
	if capacity <= 0 {
		return "", domain.ErrInvalidCapacity
	}

	meta := domain.QueueMetadata{
		ID:       u.ids.NewID(),
		Name:     name,
		Capacity: capacity,
	}
	if err := u.manager.CreateQueue(meta); err != nil {
		return "", err
	}
	return meta.ID, nil
}

func (u *QueueUseCase) Push(queueID, value string, ttl time.Duration) error {
	if ttl < 0 {
		return domain.ErrInvalidTTL
	}
	q, err := u.manager.GetQueue(queueID)
	if err != nil {
		return err
	}
	return q.Push(domain.Message{
		ID:        u.ids.NewID(),
		Value:     value,
		CreatedAt: u.clock.Now(),
		TTL:       ttl,
	})
}

func (u *QueueUseCase) Pop(queueID string) (domain.Message, error) {
	q, err := u.manager.GetQueue(queueID)
	if err != nil {
		return domain.Message{}, err
	}
	return q.Pop()
}

func (u *QueueUseCase) Status(queueID string) (domain.QueueStatus, error) {
	meta, err := u.manager.GetMetadata(queueID)
	if err != nil {
		return domain.QueueStatus{}, err
	}
	q, err := u.manager.GetQueue(queueID)
	if err != nil {
		return domain.QueueStatus{}, err
	}
	return domain.QueueStatus{
		ID:       meta.ID,
		Name:     meta.Name,
		Size:     q.Size(),
		Capacity: q.Capacity(),
	}, nil
}

func (u *QueueUseCase) ListQueues() []domain.QueueMetadata {
	return u.manager.ListQueues()
}
