package usecase

import (
	"errors"
	"testing"
	"time"

	"Baby-Rabbit/internal/domain"
	"Baby-Rabbit/internal/repository"
)

type seqID struct{ n int }

func (s *seqID) NewID() string {
	s.n++
	return "id-" + string(rune('0'+s.n))
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func newSvc() *QueueUseCase {
	return NewQueueUseCase(repository.NewQueueManager(nil), &seqID{}, fixedClock{t: time.Now()})
}

func TestUseCase_CreateRejectsEmptyName(t *testing.T) {
	svc := newSvc()
	if _, err := svc.CreateQueue("", 10); !errors.Is(err, domain.ErrInvalidName) {
		t.Fatalf("got %v, want ErrInvalidName", err)
	}
}

func TestUseCase_CreateRejectsZeroCapacity(t *testing.T) {
	svc := newSvc()
	if _, err := svc.CreateQueue("q", 0); !errors.Is(err, domain.ErrInvalidCapacity) {
		t.Fatalf("got %v, want ErrInvalidCapacity", err)
	}
}

func TestUseCase_PushPopRoundTrip(t *testing.T) {
	svc := newSvc()
	id, err := svc.CreateQueue("q", 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Push(id, "hello", time.Minute); err != nil {
		t.Fatal(err)
	}
	m, err := svc.Pop(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.Value != "hello" {
		t.Fatalf("got %q", m.Value)
	}
}

func TestUseCase_Status(t *testing.T) {
	svc := newSvc()
	id, _ := svc.CreateQueue("q", 5)
	_ = svc.Push(id, "x", time.Minute)
	st, err := svc.Status(id)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size != 1 || st.Capacity != 5 || st.Name != "q" {
		t.Fatalf("bad status: %+v", st)
	}
}

func TestUseCase_QueueNotFound(t *testing.T) {
	svc := newSvc()
	_, err := svc.Pop("does-not-exist")
	if !errors.Is(err, domain.ErrQueueNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestUseCase_DuplicateName(t *testing.T) {
	svc := newSvc()
	_, _ = svc.CreateQueue("dup", 1)
	_, err := svc.CreateQueue("dup", 1)
	if !errors.Is(err, domain.ErrQueueAlreadyExists) {
		t.Fatalf("got %v", err)
	}
}
