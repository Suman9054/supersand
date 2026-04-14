package store

import (
	"container/list"
	"fmt"
	"sync"

	
)

type Query[T any] struct {
	data *list.List
	cond *sync.Cond
	
}

type Tasks int

var Sheardcond = sync.NewCond(&sync.Mutex{})

const (
	Startnewsesion = iota
	Stopesesion
	Removesesion
	Runcomand
)

type Sesioninfo struct {
	User    string
	
}

type Responschannel struct {
	Msg    any
	Status int
}

type Prioritytaskvalue struct {
	Tasktype Tasks
    Respons chan Responschannel
	Sesioninfo
}

type Unprioritytasks struct{
	Tasktype Tasks
	Comand string 
	Respons chan Responschannel
	Sesioninfo
}

type queys[T any] interface {
	Enqueue(value T)
	Dqueue() (T, error)
	Isempty() bool
	Lenth() int

}

func (q *Query[T]) Enqueue(value T) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.data.PushBack(value)
	q.cond.Signal() // Signal that a new item is added

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
	
	return q.data.Len()
}

func NewprorityTasks() queys[Prioritytaskvalue] {
	return &Query[Prioritytaskvalue]{
		data: list.New(),
		cond:Sheardcond,
	}
}

func Newunproritytsks() queys[Unprioritytasks] {
	return &Query[Unprioritytasks]{
		data: list.New(),
		cond:Sheardcond,

	}
}