package store

import (
	"container/list"
	"fmt"

	

	"github.com/suman9054/supersand/process"
)

type Query[T any] struct {
	data *list.List
}

type tasks int

const (
	Startnewsesion = iota
	Stopesesion
	Removesesion
)

type Sesioninfo struct {
	User    string
	Process process.Process
}

type Prioritytaskvalue struct {
	Tasktype tasks
	Sesioninfo
}

type Unprioritytasks struct{
	comand string 
	Sesioninfo
}

type queys[T any] interface {
	Enqueue(value T)
	Dqueue() (T, error)
	Isempty() bool
	Lenth() int

}

func (q *Query[T]) Enqueue(value T) {

	q.data.PushBack(value)
}

func (q *Query[T]) Isempty() bool {
	if q.data.Len() == 0 {
		return true
	}
	return false
}

func (q *Query[T]) Dqueue() (T, error) {
	if q.Isempty() {
		var zero T
		return zero, fmt.Errorf("quey is empty")
	}
	data := q.data.Front()
	q.data.Remove(data)
	return data.Value.(T), nil
}

func (q *Query[T]) Lenth() int {
	
	return q.Lenth()
}

func NewprorityTasks() queys[Prioritytaskvalue] {
	return &Query[Prioritytaskvalue]{
		data: list.New(),
	}
}

func Newunproritytsks() queys[Unprioritytasks] {
	return &Query[Unprioritytasks]{
		data: list.New(),
	}
}