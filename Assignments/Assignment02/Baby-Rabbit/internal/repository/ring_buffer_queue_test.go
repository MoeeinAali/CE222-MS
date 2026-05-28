package repository

import (
	"errors"
	"testing"
	"time"

	"Baby-Rabbit/internal/domain"
)

func TestRingBufferQueue_PushPopFIFO(t *testing.T) {
	q := NewRingBufferQueue(3)
	for i, v := range []string{"a", "b", "c"} {
		if err := q.Push(domain.Message{ID: string(rune('A' + i)), Value: v}); err != nil {
			t.Fatalf("push %d: %v", i, err)
		}
	}
	if q.Size() != 3 {
		t.Fatalf("size = %d, want 3", q.Size())
	}
	for _, want := range []string{"a", "b", "c"} {
		m, err := q.Pop()
		if err != nil {
			t.Fatalf("pop: %v", err)
		}
		if m.Value != want {
			t.Fatalf("got %q, want %q", m.Value, want)
		}
	}
}

func TestRingBufferQueue_PushFull(t *testing.T) {
	q := NewRingBufferQueue(1)
	_ = q.Push(domain.Message{Value: "x"})
	err := q.Push(domain.Message{Value: "y"})
	if !errors.Is(err, domain.ErrQueueFull) {
		t.Fatalf("got %v, want ErrQueueFull", err)
	}
}

func TestRingBufferQueue_PopEmpty(t *testing.T) {
	q := NewRingBufferQueue(1)
	_, err := q.Pop()
	if !errors.Is(err, domain.ErrQueueEmpty) {
		t.Fatalf("got %v, want ErrQueueEmpty", err)
	}
}

func TestRingBufferQueue_PopSkipsExpired(t *testing.T) {
	q := NewRingBufferQueue(3)
	past := time.Now().Add(-1 * time.Hour)
	_ = q.Push(domain.Message{Value: "old", CreatedAt: past, TTL: time.Minute})
	_ = q.Push(domain.Message{Value: "fresh", CreatedAt: time.Now(), TTL: time.Hour})

	m, err := q.Pop()
	if err != nil {
		t.Fatalf("pop: %v", err)
	}
	if m.Value != "fresh" {
		t.Fatalf("got %q, want %q", m.Value, "fresh")
	}
}

func TestRingBufferQueue_RemoveExpired(t *testing.T) {
	q := NewRingBufferQueue(4)
	past := time.Now().Add(-1 * time.Hour)
	_ = q.Push(domain.Message{Value: "old", CreatedAt: past, TTL: time.Minute})
	_ = q.Push(domain.Message{Value: "keep", CreatedAt: time.Now(), TTL: time.Hour})
	_ = q.Push(domain.Message{Value: "old2", CreatedAt: past, TTL: time.Minute})

	removed := q.RemoveExpired()
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	if q.Size() != 1 {
		t.Fatalf("size = %d, want 1", q.Size())
	}
	m, _ := q.Pop()
	if m.Value != "keep" {
		t.Fatalf("got %q, want %q", m.Value, "keep")
	}
}

func TestRingBufferQueue_WrapAround(t *testing.T) {
	q := NewRingBufferQueue(2)
	_ = q.Push(domain.Message{Value: "1"})
	_ = q.Push(domain.Message{Value: "2"})
	_, _ = q.Pop()
	_ = q.Push(domain.Message{Value: "3"})

	m1, _ := q.Pop()
	m2, _ := q.Pop()
	if m1.Value != "2" || m2.Value != "3" {
		t.Fatalf("got %q,%q want 2,3", m1.Value, m2.Value)
	}
}
