package store

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/suman9054/supersand/healper"
)

type Servicedata struct {
	Id            uuid.UUID
	Lastacces     time.Time
	Processtatus  healper.Status
	ServiceUptime time.Duration
	Ramusage      int8
	WorkingDir    string
}

type UserObject struct {
	Id       uuid.UUID
	Survices []Servicedata
}

type Processdata struct {
	PID           int
	ProcessStatus healper.Status
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
	if _, exist := r.m.Load(key); !exist {
		r.count.Add(1)
	}
	r.m.Store(key, value)
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

func NewUserCash() stable[uint64, UserObject] {
	return &chash[uint64, UserObject]{
		m: sync.Map{},
	}
}

func Newprocesscash() stable[string, Processdata] {
	return &chash[string, Processdata]{
		m: sync.Map{},
	}
}
