package clock

import "time"

// Real implements usecase.Clock using the system wall clock.
type Real struct{}

func (Real) Now() time.Time { return time.Now() }
