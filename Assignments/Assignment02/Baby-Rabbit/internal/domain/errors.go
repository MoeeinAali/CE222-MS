package domain

import "errors"

var (
	ErrQueueNotFound      = errors.New("queue not found")
	ErrQueueAlreadyExists = errors.New("queue already exists")
	ErrQueueFull          = errors.New("queue full")
	ErrQueueEmpty         = errors.New("queue empty")
	ErrInvalidCapacity    = errors.New("invalid capacity")
	ErrInvalidName        = errors.New("invalid queue name")
	ErrInvalidTTL         = errors.New("invalid ttl")
)
