package store

import (
	"testing"
)

func TestNewstoreNoopMethods(t *testing.T) {
	s := Newstore()
	if s == nil {
		t.Fatal("Newstore returned nil")
	}
	if err := s.CreatUser(&UserObject{Id: []byte("u1")}); err != nil {
		t.Fatalf("CreatUser: %v", err)
	}
	if err := s.Updateuser(&UserObject{Id: []byte("u1")}); err != nil {
		t.Fatalf("Updateuser: %v", err)
	}
	var k [16]byte
	copy(k[:], "u1")
	if err := s.DeleteUser(k); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
}

func TestStoreCacheSetGetRemove(t *testing.T) {
	s := Newstore().(*store)
	var k [16]byte
	copy(k[:], "user-key")
	s.Chash.Set(k, UserObject{Id: []byte("user-key")})
	v, ok := s.Chash.Get(k)
	if !ok {
		t.Fatal("expected cached value")
	}
	if string(v.Id) != "user-key" {
		t.Fatalf("unexpected cached value: %s", v.Id)
	}
	if !s.Chash.Remove(k) {
		t.Fatal("expected remove to succeed")
	}
	_, ok = s.Chash.Get(k)
	if ok {
		t.Fatal("expected value gone after remove")
	}
}

func TestStorePriorityQueue(t *testing.T) {
	q := NewprorityTasks()
	task1 := Prioritytaskvalue{Tasktype: Startnewsesion, Sesioninfo: Sesioninfo{User: "suman"}}
	task2 := Prioritytaskvalue{Tasktype: Startnewsesion, Sesioninfo: Sesioninfo{User: "suman"}}
	q.Enqueue(task1)
	q.Enqueue(task2)
	if q.Lenth() != 2 {
		t.Fatalf("expected length 2, got %d", q.Lenth())
	}
	if q.Isempty() {
		t.Fatal("queue should not be empty")
	}
	d1, err := q.Dqueue()
	if err != nil {
		t.Fatal(err)
	}
	if d1.Sesioninfo.User != "suman" {
		t.Fatal("wrong task dequeued")
	}
	if q.Lenth() != 1 {
		t.Fatalf("expected length 1, got %d", q.Lenth())
	}
	if _, err := q.Dqueue(); err != nil {
		t.Fatal(err)
	}
	if q.Lenth() != 0 || !q.Isempty() {
		t.Fatal("queue should be empty after two dequeues")
	}
	if _, err := q.Dqueue(); err == nil {
		t.Fatal("expected error dequeuing empty queue")
	}
}
