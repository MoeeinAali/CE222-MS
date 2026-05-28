package domain

// Logger is the port the inner layers use for diagnostics.
// Implementations live in the infrastructure layer (e.g. internal/pkg/logger).
// Defined in domain because both usecase and service consume it.
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// Nop is a Logger that discards everything. Useful for tests.
type Nop struct{}

func (Nop) Debugf(string, ...any) {}
func (Nop) Infof(string, ...any)  {}
func (Nop) Warnf(string, ...any)  {}
func (Nop) Errorf(string, ...any) {}
