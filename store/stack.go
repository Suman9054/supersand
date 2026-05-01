package store

import (
	"fmt"
	"sync"

	"github.com/suman9054/supersand/process"
)

type Stack[T any] struct {
	elements []interface{}
	mu       sync.Mutex
}

type ProcessPool[T any] interface {
	Push(value T)
	Pop() (T, error)
}

func (s *Stack[T]) Push(value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.elements = append(s.elements, value)
}

func (s *Stack[T]) Pop() (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.elements) == 0 {
		var zero T

		return zero, fmt.Errorf("stack is empty")
	}
	data := s.elements[0]

	s.elements = s.elements[1:]

	return data.(T), nil
}

func Newstack() ProcessPool[process.Process] {
	return &Stack[process.Process]{
		elements: make([]interface{}, 0),
	}
}
