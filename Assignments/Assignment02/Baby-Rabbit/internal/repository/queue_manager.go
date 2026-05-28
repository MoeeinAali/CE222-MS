package repository

import (
	"sync"

	"Baby-Rabbit/internal/domain"
)

// QueueManager is an in-memory implementation of domain.QueueManager.
// It is storage-agnostic: the concrete Queue is built by an injected
// QueueFactory, so swapping the queue implementation (linked list,
// Redis-backed, etc.) does not require modifying this type.
type QueueManager struct {
	mu        sync.RWMutex
	factory   domain.QueueFactory
	queues    map[string]domain.Queue
	metadata  map[string]domain.QueueMetadata
	nameIndex map[string]string
}

func NewQueueManager(factory domain.QueueFactory) *QueueManager {
	if factory == nil {
		factory = domain.QueueFactoryFunc(func(meta domain.QueueMetadata) domain.Queue {
			return NewRingBufferQueue(meta.Capacity)
		})
	}
	return &QueueManager{
		factory:   factory,
		queues:    make(map[string]domain.Queue),
		metadata:  make(map[string]domain.QueueMetadata),
		nameIndex: make(map[string]string),
	}
}

func (m *QueueManager) CreateQueue(meta domain.QueueMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.nameIndex[meta.Name]; exists {
		return domain.ErrQueueAlreadyExists
	}
	m.queues[meta.ID] = m.factory.New(meta)
	m.metadata[meta.ID] = meta
	m.nameIndex[meta.Name] = meta.ID
	return nil
}

func (m *QueueManager) GetQueue(id string) (domain.Queue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	q, ok := m.queues[id]
	if !ok {
		return nil, domain.ErrQueueNotFound
	}
	return q, nil
}

func (m *QueueManager) GetMetadata(id string) (domain.QueueMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	meta, ok := m.metadata[id]
	if !ok {
		return domain.QueueMetadata{}, domain.ErrQueueNotFound
	}
	return meta, nil
}

func (m *QueueManager) ListQueues() []domain.QueueMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]domain.QueueMetadata, 0, len(m.metadata))
	for _, meta := range m.metadata {
		result = append(result, meta)
	}
	return result
}

// RingBufferFactory is the default factory used in production.
type RingBufferFactory struct{}

func (RingBufferFactory) New(meta domain.QueueMetadata) domain.Queue {
	return NewRingBufferQueue(meta.Capacity)
}
