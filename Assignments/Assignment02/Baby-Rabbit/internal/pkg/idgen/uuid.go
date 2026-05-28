package idgen

import "github.com/google/uuid"

// UUID implements usecase.IDGenerator using google/uuid.
type UUID struct{}

func (UUID) NewID() string { return uuid.NewString() }
