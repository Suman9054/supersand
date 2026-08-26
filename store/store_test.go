package store

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/suman9054/supersand/healper"
)

func TestCashSetGetUpdateAndDelete(t *testing.T) {
	s := Newstore()

	now := time.Now()

	id := uint64(1)
	obj := UserObject{
		Id: uuid.New(),
		Survices: []Servicedata{
			{
				Id:           uuid.New(),
				Lastacces:    now,
				Processtatus: healper.Active,
			},
		},
	}

	s.Chash.Set(id, obj)

	_, ok := s.Chash.Get(id)
	if !ok {
		t.Fatal("expected to get the value but got nothing")
	}

	if !s.Chash.Remove(id) {
		t.Fatal("expected remove to succeed")
	}

	_, ok = s.Chash.Get(id)
	if ok {
		t.Fatal("expected value to be deleted")
	}
}

func TestQueueEnqueueDequeue(t *testing.T) {
	s := Newstore()

	task1 := Prioritytaskvalue{
		Tasktype: Startnewsesion,
		Sesioninfo: Sesioninfo{
			User: "suman",
		},
	}
	task2 := Prioritytaskvalue{
		Tasktype: Startnewsesion,
		Sesioninfo: Sesioninfo{
			User: "suman",
		},
	}
	s.Querys.Enqueue(task1)
	s.Querys.Enqueue(task2)
	len := s.Querys.Lenth()
	if len != 2 {
		t.Fatal("expected length to be 2")
	}
	empty := s.Querys.Isempty()
	if empty {
		t.Fatal("expected queue to not be empty")
	}
	dequaue1, err := s.Querys.Dqueue()
	if err != nil {
		t.Fatal("expected to dequeue a task")
	}

	if dequaue1.Tasktype != Startnewsesion || dequaue1.Sesioninfo.User != "suman" {
		t.Fatal("dequeued wrong task")
	}
	len = s.Querys.Lenth()
	if len != 1 {
		t.Fatal("expected length to be 1 after one dequeue")
	}
	empty = s.Querys.Isempty()
	if empty {
		t.Fatal("expected queue to not be empty after one dequeue")
	}
	dequaue2, err := s.Querys.Dqueue()
	if err != nil {
		t.Fatal("expected to dequeue a second task")
	}
	if dequaue2.Tasktype != Startnewsesion || dequaue2.Sesioninfo.User != "suman" {
		t.Fatal("dequeued wrong second task")
	}
	len = s.Querys.Lenth()
	if len != 0 {
		t.Fatal("expected length to be 0 after two dequeues")
	}
	empty = s.Querys.Isempty()
	if !empty {
		t.Fatal("expected queue to be empty after two dequeues")
	}
	_, err = s.Querys.Dqueue()
	if err == nil {
		t.Fatal("expected no more tasks to dequeue")
	}
}
