package repository

import (
	"sync"
	"time"

	"Baby-Rabbit/internal/domain"
)

// RingBufferQueue is a bounded, in-memory, thread-safe FIFO queue
// backed by a circular buffer. It implements domain.Queue.
type RingBufferQueue struct {
	mu       sync.Mutex
	buffer   []domain.Message
	head     int
	tail     int
	size     int
	capacity int
	now      func() time.Time
}

func NewRingBufferQueue(capacity int) *RingBufferQueue {
	return &RingBufferQueue{
		buffer:   make([]domain.Message, capacity),
		capacity: capacity,
		now:      time.Now,
	}
}

func (q *RingBufferQueue) Push(msg domain.Message) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.size == q.capacity {
		return domain.ErrQueueFull
	}
	q.buffer[q.tail] = msg
	q.tail = (q.tail + 1) % q.capacity
	q.size++
	return nil
}

// Pop is non-blocking. If the queue is empty it returns ErrQueueEmpty.
// Expired messages at the head are skipped (lazy expiration) so the
// caller never sees a stale message even if the TTL cleaner has not run.
func (q *RingBufferQueue) Pop() (domain.Message, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := q.now()
	for q.size > 0 {
		msg := q.buffer[q.head]
		q.buffer[q.head] = domain.Message{}
		q.head = (q.head + 1) % q.capacity
		q.size--

		if !msg.ExpiredAt(now) {
			return msg, nil
		}
	}
	return domain.Message{}, domain.ErrQueueEmpty
}

func (q *RingBufferQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.size
}

func (q *RingBufferQueue) Capacity() int {
	return q.capacity
}

// RemoveExpired compacts the buffer, dropping every expired message,
// and returns how many were removed.
func (q *RingBufferQueue) RemoveExpired() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := q.now()
	removed := 0
	newBuffer := make([]domain.Message, q.capacity)
	index := 0
	for i := 0; i < q.size; i++ {
		pos := (q.head + i) % q.capacity
		msg := q.buffer[pos]
		if msg.ExpiredAt(now) {
			removed++
			continue
		}
		newBuffer[index] = msg
		index++
	}
	q.buffer = newBuffer
	q.head = 0
	q.tail = index % q.capacity
	q.size = index
	return removed
}
