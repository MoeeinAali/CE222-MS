package logger

import (
	"go.uber.org/zap"

	"Baby-Rabbit/internal/domain"
)

// Zap adapts go.uber.org/zap to the domain.Logger port so the inner
// layers depend only on the abstraction, never on zap itself.
type Zap struct {
	sugar *zap.SugaredLogger
}

func NewZap() (*Zap, error) {
	l, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &Zap{sugar: l.Sugar()}, nil
}

func (z *Zap) Debugf(f string, a ...any) { z.sugar.Debugf(f, a...) }
func (z *Zap) Infof(f string, a ...any)  { z.sugar.Infof(f, a...) }
func (z *Zap) Warnf(f string, a ...any)  { z.sugar.Warnf(f, a...) }
func (z *Zap) Errorf(f string, a ...any) { z.sugar.Errorf(f, a...) }
func (z *Zap) Fatalf(f string, a ...any) { z.sugar.Fatalf(f, a...) }
func (z *Zap) Sync()                     { _ = z.sugar.Sync() }

// Compile-time assertion: Zap must satisfy domain.Logger.
var _ domain.Logger = (*Zap)(nil)
