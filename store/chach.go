package store

import (
	"container/list"
	"sync"
	"sync/atomic"
)

type entry[k comparable, v any] struct {
	key   k
	value v
	freq  int
}

type chach[k comparable, v any] struct {
	defalt       v
	capacity     int64
	minfrequency int
	m            map[k]*list.Element
	f            map[int]*list.List
	list         list.List
	count        atomic.Int64
	mu           sync.Mutex
}

type stable[k comparable, v any] interface {
	Get(key k) (v, bool)
	Set(key k, value v)
	Remove(key k) bool
}

func NewChach(capacity int64) stable[uint64, UserObject] {
	return &chach[uint64, UserObject]{
		capacity: capacity,
		m:        map[uint64]*list.Element{},
		f:        map[int]*list.List{},
		list:     *list.New(),
	}
}

func (r *chach[k, v]) Get(key k) (v, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	node, ok := r.m[key]
	if !ok {
		return r.defalt, false
	}
	node = r.updatefreq(node)
	r.m[key] = node
	return node.Value.(*entry[k, v]).value, true
}

func (r *chach[k, v]) Set(key k, value v) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.capacity <= 0 {
		return
	}
	node, ok := r.m[key]

	if ok {
		node.Value.(*entry[k, v]).value = value
		r.m[key] = node
		return
	}

	if r.capacity == int64(r.count.Load()) {
		r.handeloverlode()
	}

	bucket, ok := r.f[1]

	if !ok {
		bucket = list.New()
		r.f[1] = bucket
	}
	r.m[key] = bucket.PushFront(&entry[k, v]{
		key:   key,
		value: value,
		freq:  1,
	})

	r.minfrequency = 1
	r.count.Add(1)

	return
}

func (r *chach[k, v]) Remove(key k) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	node, ok := r.m[key]
	if !ok {
		return false
	}

	en := node.Value.(*entry[k, v])
	bucket := r.f[en.freq]
	bucket.Remove(node)
	delete(r.m, key)
	r.count.Add(-1)

	if bucket.Len() == 0 {
		delete(r.f, en.freq)
		if r.minfrequency == en.freq {
			go func() {
				for f, _ := range r.f {
					if f == 0 || f < r.minfrequency {
						r.minfrequency = f
						return
					}
				}
			}()
		}
	}
	return true
}

func (r *chach[k, v]) updatefreq(e *list.Element) *list.Element {
	en := e.Value.(*entry[k, v])

	old := r.f[en.freq]

	old.Remove(e)

	if old.Len() == 0 {
		delete(r.f, en.freq)

		if r.minfrequency == en.freq {
			r.minfrequency++
		}
	}

	en.freq++
	next, ok := r.f[en.freq]

	if !ok {
		next = list.New()
		r.f[en.freq] = next
	}

	return next.PushFront(en)
}

func (r *chach[k, v]) handeloverlode() {
	bucket := r.f[r.minfrequency]
	back := bucket.Back()
	bucket.Remove(back)
	delete(r.m, back.Value.(*entry[k, v]).key)
	r.count.Add(-1)

	if bucket.Len() == 0 {
		delete(r.f, r.minfrequency)
	}

}
