package store

import (
	"container/list"
	"fmt"

	

	"github.com/suman9054/supersand/process"
)

type Query struct {
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

type unprioritytasks struct{
	comand string 
	Sesioninfo
}

type queys interface {
	Enqueue(value Prioritytaskvalue)
	Dqueue() (Prioritytaskvalue, error)
	Isempty() bool
	Lenth() int

}

func (q *Query) Enqueue(value Prioritytaskvalue) {
	q.data.PushBack(value)
}

func (q *Query) Isempty() bool {
	if q.data.Len() == 0 {
		return true
	}
	return false
}

func (q *Query) Dqueue() (Prioritytaskvalue, error) {
	if q.Isempty() {
		var zero Prioritytaskvalue
		return zero, fmt.Errorf("quey is empty")
	}
	data := q.data.Front()
	q.data.Remove(data)
	return data.Value.(Prioritytaskvalue), nil
}

func (q *Query) Lenth() int {
	if q.Isempty(){
		return 0
	}
	return q.Lenth()
}

func NewprorityTasks() queys {
	return &Query{
		data: list.New(),
	}
}
