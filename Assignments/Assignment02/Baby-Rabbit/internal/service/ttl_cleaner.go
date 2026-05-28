package service

import (
	"context"
	"time"

	"Baby-Rabbit/internal/domain"
)

// TTLCleaner periodically asks every queue to drop expired messages.
// It depends only on domain ports — never on a concrete logger or clock.
type TTLCleaner struct {
	manager  domain.QueueManager
	log      domain.Logger
	interval time.Duration
}

func NewTTLCleaner(m domain.QueueManager, log domain.Logger, interval time.Duration) *TTLCleaner {
	if log == nil {
		log = domain.Nop{}
	}
	if interval <= 0 {
		interval = time.Second
	}
	return &TTLCleaner{manager: m, log: log, interval: interval}
}

// Run blocks until ctx is canceled, periodically sweeping expired messages.
func (c *TTLCleaner) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Infof("ttl cleaner stopped")
			return
		case <-ticker.C:
			c.sweep()
		}
	}
}

func (c *TTLCleaner) sweep() {
	for _, meta := range c.manager.ListQueues() {
		q, err := c.manager.GetQueue(meta.ID)
		if err != nil {
			continue
		}
		if removed := q.RemoveExpired(); removed > 0 {
			c.log.Debugf("ttl cleaner removed %d expired messages from %s", removed, meta.Name)
		}
	}
}
