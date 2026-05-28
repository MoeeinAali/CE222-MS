package domain

import "time"

// Message is the core entity stored in queues.
// TTL == 0 means "never expires".
type Message struct {
	ID        string
	Value     string
	CreatedAt time.Time
	TTL       time.Duration
}

func (m Message) ExpiredAt(now time.Time) bool {
	if m.TTL <= 0 {
		return false
	}
	return now.Sub(m.CreatedAt) >= m.TTL
}
