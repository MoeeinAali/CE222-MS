package domain

// Queue is the port for a single FIFO message queue.
// Implementations live in the repository layer.
type Queue interface {
	Push(Message) error
	Pop() (Message, error)
	Size() int
	Capacity() int
	RemoveExpired() int
}

// QueueMetadata is an immutable descriptor of a queue.
type QueueMetadata struct {
	ID       string
	Name     string
	Capacity int
}

// QueueStatus is the runtime snapshot of a queue.
type QueueStatus struct {
	ID       string
	Name     string
	Size     int
	Capacity int
}
