package store

import "fmt"

type faildquequs struct {
	elements []interface{}
}

type Stack interface {
	Push(value taskvalue)
	Pop() (taskvalue, error)
}

func (s *faildquequs) Push(value taskvalue) {
	s.elements = append(s.elements, value)
}

func (s *faildquequs) Pop() (taskvalue, error) {
	if len(s.elements) == 0 {
		var zero taskvalue

		return zero, fmt.Errorf("stack is empty")
	}
	n := len(s.elements) - 1
	data := s.elements[n]
	s.elements[n] = nil
	s.elements = s.elements[:n]

	return data.(taskvalue), nil
}

func Newstack() Stack {
	return &faildquequs{
		elements: make([]interface{}, 0),
	}
}
