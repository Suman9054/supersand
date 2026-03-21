package store

import (
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

type status int

const (
	active status = iota
	stope
	pending
)

type userdata struct {
	id            string
	useuniqename  string
	lastacces     time.Time
	processstatus status
	process       process.Process

}

type chash[k comparable, v any] struct {
	defalt v
	m      sync.Map
	count  atomic.Int64
}

type stable[k comparable, v any] interface {
	Get(key k) (v, bool)
	Set(key k, value v) bool
	Remove(key k) bool
}

func (r *chash[k, v]) Get(key k) (v, bool) {
	value, ok := r.m.Load(key)
	if !ok {
		return r.defalt, false
	}
	return value.(v), true
}

func (r *chash[k, v]) Set(key k, value v) bool {
	r.m.Store(key, value)

	r.count.Add(1)
	return true
}

func (r *chash[k, v]) Remove(key k) bool {
	r.m.Delete(key)
	_, ok := r.Get(key)
	if ok {
		return false
	}
	r.count.Add(-1)
	return true
}

func Newstoremap() stable[string, userdata] {
	return &chash[string, userdata]{
		m: sync.Map{},
	}
}
