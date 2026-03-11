package cach

import (
	"sync"
	"sync/atomic"
)

type userdata struct {
	id            string
	useuniqename  string
	lastacces     string
	processactive bool
}

type chash[k comparable, v userdata] struct {
	defalt v
	m      sync.Map
	count  atomic.Int64
}

func (r *chash[k, v]) get(key k) (v, bool) {
	value, ok := r.m.Load(key)
	if !ok {
		return r.defalt, false
	}
	return value.(v), true
}

func (r *chash[k, v]) set(key k, value v) bool {
	r.m.Store(key, value)

	_, ok := r.m.Load(key)
	if !ok {
		return false
	}
	r.count.Add(1)
	return true
}

func (r *chash[k, v]) delet(key k) bool {
	r.m.Delete(key)
	_, ok := r.get(key)
	if ok {
		return false
	}
	return true
}
