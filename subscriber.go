package orenoagent

import (
	"sync"
)

type Subscriber[T any] struct {
	mu      sync.RWMutex
	history []T
	stream  chan T
	closed  bool
}

func NewSubscriber[T any](bufferSize int) *Subscriber[T] {
	return &Subscriber[T]{
		history: make([]T, 0),
		stream:  make(chan T, bufferSize),
	}
}

func (s *Subscriber[T]) Publish(data T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false
	}

	s.history = append(s.history, data)

	select {
	case s.stream <- data:
		return true
	default:
		return false
	}
}

func (s *Subscriber[T]) Subscribe() <-chan T {
	return s.stream
}

func (s *Subscriber[T]) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		close(s.stream)
	}
}

func (s *Subscriber[T]) GetHistory() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]T, len(s.history))
	copy(res, s.history)
	return res
}

func (s *Subscriber[T]) FindFirst(predicate func(T) bool) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.history {
		if predicate(item) {
			return item, true
		}
	}

	var zero T
	return zero, false
}

func (s *Subscriber[T]) GetHistoryAt(index int) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	length := len(s.history)
	var zero T

	if length == 0 {
		return zero, false
	}

	if index < 0 {
		index = length + index
	}

	if index < 0 || index >= length {
		return zero, false
	}

	return s.history[index], true
}
