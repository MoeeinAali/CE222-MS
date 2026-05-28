package usecase

import (
	"time"

	"Baby-Rabbit/internal/domain"
)

// QueueService is the inbound port consumed by the delivery layer.
// Defining it here (and not in domain) keeps the application boundary
// explicit while the delivery layer depends only on this interface.
type QueueService interface {
	CreateQueue(name string, capacity int) (string, error)
	Push(queueID, value string, ttl time.Duration) error
	Pop(queueID string) (domain.Message, error)
	Status(queueID string) (domain.QueueStatus, error)
	ListQueues() []domain.QueueMetadata
}

// IDGenerator abstracts ID creation so the usecase layer does not
// depend on a specific UUID library.
type IDGenerator interface {
	NewID() string
}

// Clock abstracts time for testability.
type Clock interface {
	Now() time.Time
}
