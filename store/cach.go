package store

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/suman9054/supersand/process"
)

type Status int

const (
	Active Status = iota
	Stopped
	Pending
)

type Userdata struct {
	Id            string
	Useuniqename  string
	Lastacces     time.Time
	Processstatus Status
	Process       process.Sandbox
}

type chash[k comparable, v any] struct {
	defalt v
	m      sync.Map
	count  atomic.Int64
	mu     sync.Mutex
}

type stable[k comparable, v any] interface {
	Get(key k) (v, bool)
	Set(key k, value v)
	Remove(key k) bool
	Allitems() map[k]v
	Update(key k, fn func(v) v) (error, bool)
}

func (r *chash[k, v]) Get(key k) (v, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	value, ok := r.m.Load(key)
	if !ok {
		return r.defalt, false
	}
	return value.(v), true
}

func (r *chash[k, v]) Set(key k, value v) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.m.Store(key, value)

	r.count.Add(1)
}

func (r *chash[k, v]) Update(key k, fn func(v) v) (error, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.m.Load(key)
	if !ok {
		return fmt.Errorf("user does not exist"), false
	}

	updated := fn(value.(v))
	r.m.Store(key, updated)
	return nil, true
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

func (r *chash[k, v]) Allitems() map[k]v {
	items := make(map[k]v)
	r.m.Range(func(key, value any) bool {
		items[key.(k)] = value.(v)
		return true
	})
	return items
}

func Newstoremap() stable[string, Userdata] {
	return &chash[string, Userdata]{
		m: sync.Map{},
	}
}
