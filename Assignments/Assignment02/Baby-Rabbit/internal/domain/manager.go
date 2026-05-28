package domain

// QueueManager is the port for the collection of queues.
type QueueManager interface {
	CreateQueue(meta QueueMetadata) error
	GetQueue(id string) (Queue, error)
	GetMetadata(id string) (QueueMetadata, error)
	ListQueues() []QueueMetadata
}

// QueueFactory builds a fresh Queue for the given metadata.
// Injecting this into QueueManager keeps the manager open for new
// storage backends (ring buffer, linked list, Redis, ...) without
// modification.
type QueueFactory interface {
	New(meta QueueMetadata) Queue
}

// QueueFactoryFunc adapts a plain function to QueueFactory.
type QueueFactoryFunc func(QueueMetadata) Queue

func (f QueueFactoryFunc) New(meta QueueMetadata) Queue { return f(meta) }
