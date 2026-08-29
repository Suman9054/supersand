package store

import (
	"testing"

	rpc_api "github.com/suman9054/supersand/rpc"
)

// keyOf derives the [16]byte storage key the same way store.CreatUser and
// store.UpdateuserData do: the first 16 bytes of the user Id.
func keyOf(id []byte) [16]byte {
	var k [16]byte
	copy(k[:], id)
	return k
}

// newTestStore builds a *store backed by a WAL in a temp dir. This keeps the
// tests hermetic: it avoids Initwal(), which would otherwise write to the real
// ~/.config path. The store APIs under test behave identically.
func newTestStore(t *testing.T) (*store, *WAL) {
	t.Helper()
	dir := t.TempDir()
	w, err := openwal(Options{Dir: dir, SegmentMaxBytes: 1 << 20, SyncOnWrite: true})
	if err != nil {
		t.Fatal(err)
	}
	return &store{Chash: NewChach(1024), Wal: w}, w
}

func TestStoreCreatUserPersists(t *testing.T) {
	s, _ := newTestStore(t)
	id := []byte("user-one-12345")
	if err := s.CreatUser(&UserObject{Id: id}); err != nil {
		t.Fatal(err)
	}
	err, got := s.GetuserData(keyOf(id))
	if err != nil {
		t.Fatalf("GetuserData after create: %v", err)
	}
	if got == nil || string(got.Id) != string(id) {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestStoreUpdateuserData(t *testing.T) {
	s, _ := newTestStore(t)
	id := []byte("user-update-123")
	if err := s.CreatUser(&UserObject{Id: id}); err != nil {
		t.Fatal(err)
	}

	updated := &UserObject{Id: id, Services: []*rpc_api.Service{{Id: []byte("svc-1")}}}
	if err := s.UpdateuserData(updated); err != nil {
		t.Fatal(err)
	}

	err, got := s.GetuserData(keyOf(id))
	if err != nil {
		t.Fatalf("GetuserData after update: %v", err)
	}
	if len(got.Services) != 1 || string(got.Services[0].Id) != "svc-1" {
		t.Fatalf("updated services not reflected: %+v", got.Services)
	}
}

func TestStoreDeleteUser(t *testing.T) {
	s, _ := newTestStore(t)
	id := []byte("user-del-12345")
	k := keyOf(id)
	if err := s.CreatUser(&UserObject{Id: id}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteUser(k); err != nil {
		t.Fatal(err)
	}
	// The key is gone from the cache and tombstoned in the WAL, so a read must
	// fail rather than return the previously created user.
	if err, _ := s.GetuserData(k); err == nil {
		t.Fatal("expected error after DeleteUser")
	}
}

func TestStoreGetMissingUser(t *testing.T) {
	s, _ := newTestStore(t)
	if err, _ := s.GetuserData(keyOf([]byte("nope-missing-12"))); err == nil {
		t.Fatal("expected error for a user that was never created")
	}
}

func TestStorePersistenceAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	w1, err := openwal(Options{Dir: dir, SegmentMaxBytes: 1 << 20, SyncOnWrite: true})
	if err != nil {
		t.Fatal(err)
	}
	s1 := &store{Chash: NewChach(1024), Wal: w1}
	id := []byte("user-persist-12")
	k := keyOf(id)
	if err := s1.CreatUser(&UserObject{Id: id}); err != nil {
		t.Fatal(err)
	}
	// Flush + close so the segment is on disk for the next open.
	if err := w1.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopen the same directory: a fresh store with an empty cache must still
	// read the user back from the WAL (exercising the bloom-recovery path).
	w2, err := openwal(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()
	s2 := &store{Chash: NewChach(1024), Wal: w2}

	err, got := s2.GetuserData(k)
	if err != nil {
		t.Fatalf("GetuserData after reopen: %v", err)
	}
	if got == nil || string(got.Id) != string(id) {
		t.Fatalf("persisted user mismatch: %+v", got)
	}
}

func TestStoreCacheSetGetRemove(t *testing.T) {
	s := &store{Chash: NewChach(1024)}
	var k [16]byte
	copy(k[:], "user-key")
	s.Chash.Set(k, &UserObject{Id: []byte("user-key")})
	v, ok := s.Chash.Get(k)
	if !ok {
		t.Fatal("expected cached value")
	}
	if v == nil || string(v.Id) != "user-key" {
		t.Fatalf("unexpected cached value: %+v", v)
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
