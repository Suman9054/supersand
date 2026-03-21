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
	startnewsesion = iota
	stopesesion
	removesesion
)

type sesioninfo struct {
	user    string
	process process.Process
}

type taskvalue struct {
	Tasktype tasks
	sesioninfo
}

type queys interface {
	Enqueue(value taskvalue)
	Dqueue() (taskvalue, error)
	Isempty() bool
	Lenth() int

}

func (q *Query) Enqueue(value taskvalue) {
	q.data.PushBack(value)
}

func (q *Query) Isempty() bool {
	if q.data.Len() == 0 {
		return true
	}
	return false
}

func (q *Query) Dqueue() (taskvalue, error) {
	if q.Isempty() {
		var zero taskvalue
		return zero, fmt.Errorf("quey is empty")
	}
	data := q.data.Front()
	q.data.Remove(data)
	return data.Value.(taskvalue), nil
}

func (q *Query) Lenth() int {
	if q.Isempty(){
		return 0
	}
	return q.Lenth()
}

func NewTasks() queys {
	return &Query{
		data: list.New(),
	}
}
