package store

import (
	"encoding/binary"
	"hash/crc32"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// ---- record helpers (real on-disk format, shared with the WAL) ----

func writeSegRecord(t *testing.T, path string, typ RecordType, key [16]byte, payload []byte) {
	t.Helper()
	var hdr [headerSize]byte
	crc := crc32.NewIEEE()
	crc.Write([]byte{byte(typ)})
	crc.Write(payload)
	binary.LittleEndian.PutUint32(hdr[0:4], crc.Sum32())
	hdr[4] = byte(typ)
	binary.LittleEndian.PutUint16(hdr[5:7], uint16(len(key)))
	binary.LittleEndian.PutUint32(hdr[7:11], uint32(len(payload)))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(hdr[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(key[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(payload); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func key16(b byte) [16]byte {
	var k [16]byte
	k[0] = b
	return k
}

// ---- sizing / construction ----

func TestNewBloomInitializes(t *testing.T) {
	for _, size := range []int64{1, 2, 4, 8, 64, 1024, 100000} {
		b := newBloom(size)
		if b == nil {
			t.Fatalf("newBloom(%d) returned nil", size)
		}
		if b.m <= 0 {
			t.Fatalf("newBloom(%d): m must be > 0, got %d", size, b.m)
		}
		if b.k <= 0 {
			t.Fatalf("newBloom(%d): k must be > 0, got %d", size, b.k)
		}
		if len(b.bitarr) != (b.m+7)/8 {
			t.Fatalf("newBloom(%d): bitarr len %d != (m+7)/8 = %d", size, len(b.bitarr), (b.m+7)/8)
		}
	}
}

func TestNewBloomZeroSize(t *testing.T) {
	b := newBloom(0)
	if b.m <= 0 || b.k <= 0 {
		t.Fatalf("zero size should clamp to a usable filter, got m=%d k=%d", b.m, b.k)
	}
}

func TestNewBloomSizingFormula(t *testing.T) {
	size := int64(8)
	max := -float64(size) * math.Log(0.001) / math.Pow(math.Log(2), 2)
	axh := (max / float64(size)) * math.Log(2)
	b := newBloom(size)
	if b.m != int(max) {
		t.Fatalf("m mismatch: got %d want %d", b.m, int(max))
	}
	if b.k != int(axh) {
		t.Fatalf("k mismatch: got %d want %d", b.k, int(axh))
	}
}

// ---- add / test ----

func TestBloomAddThenTestTrue(t *testing.T) {
	b := newBloom(1024)
	b.add([]byte("hello"))
	if !b.test([]byte("hello")) {
		t.Fatal("added key should test true")
	}
}

func TestBloomNoFalseNegatives(t *testing.T) {
	b := newBloom(1024)
	keys := []string{"a", "b", "c", "suman", "hello", "world", "foo", "bar", "zzz", "qq"}
	for _, k := range keys {
		b.add([]byte(k))
	}
	for _, k := range keys {
		if !b.test([]byte(k)) {
			t.Fatalf("false negative for key %q", k)
		}
	}
}

func TestBloomIdempotentAdd(t *testing.T) {
	b := newBloom(1024)
	b.add([]byte("x"))
	b.add([]byte("x"))
	if !b.test([]byte("x")) {
		t.Fatal("key should still test true after repeated add")
	}
}

func TestBloomTestDeterministic(t *testing.T) {
	b := newBloom(1024)
	b.add([]byte("key"))
	if b.test([]byte("key")) != b.test([]byte("key")) {
		t.Fatal("test result must be deterministic for a key")
	}
}

func TestBloomEmptyKey(t *testing.T) {
	b := newBloom(1024)
	b.add([]byte(""))
	if !b.test([]byte("")) {
		t.Fatal("empty key should test true after add")
	}
}

func TestBloomMissingKeyLikelyFalse(t *testing.T) {
	b := newBloom(1024)
	added := []string{"a", "b", "c", "d", "e"}
	for _, k := range added {
		b.add([]byte(k))
	}
	// p = 0.001, so an unrelated key is overwhelmingly false
	if b.test([]byte("this-key-was-never-added")) {
		t.Fatal("unexpected true for a never-added key (false positive beyond configured rate)")
	}
}

// ---- getkeyindx ----

func TestGetkeyindxCount(t *testing.T) {
	b := newBloom(1024)
	idx := getkeyindx([]byte("x"), b.k, b.m)
	if len(idx) != b.k {
		t.Fatalf("expected %d indices, got %d", b.k, len(idx))
	}
}

func TestGetkeyindxDeterministic(t *testing.T) {
	a := getkeyindx([]byte("determinism"), 10, 100000)
	b := getkeyindx([]byte("determinism"), 10, 100000)
	if len(a) != len(b) {
		t.Fatal("length mismatch between calls")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("index %d differs: %d vs %d", i, a[i], b[i])
		}
	}
}

func TestGetkeyindxWithinBounds(t *testing.T) {
	m := 100000
	idx := getkeyindx([]byte("bounds"), 8, m)
	if len(idx) != 8 {
		t.Fatalf("expected 8 indices, got %d", len(idx))
	}
	for _, v := range idx {
		if int(v) >= m {
			t.Fatalf("index %d out of bounds for m=%d", v, m)
		}
	}
}

func TestGetkeyindxZeroK(t *testing.T) {
	idx := getkeyindx([]byte("zero"), 0, 100)
	if len(idx) != 0 {
		t.Fatalf("expected 0 indices for k=0, got %d", len(idx))
	}
}

func TestGetkeyindxOneBitField(t *testing.T) {
	idx := getkeyindx([]byte("one"), 5, 1)
	if len(idx) != 5 {
		t.Fatalf("expected 5 indices, got %d", len(idx))
	}
	for _, v := range idx {
		if v != 0 {
			t.Fatalf("expected all indices 0 for m=1, got %d", v)
		}
	}
}

// Regression: the old code used h1 == h2, collapsing indices to h1*(i+1).
// With two independent hashes the probes must be distinct for a real key.
func TestGetkeyindxDualHashSpread(t *testing.T) {
	idx := getkeyindx([]byte("spread-me"), 5, 100000)
	seen := map[uint64]bool{}
	for _, v := range idx {
		if seen[v] {
			t.Fatalf("duplicate index %d: h1 and h2 are not independent", v)
		}
		seen[v] = true
	}
}

func TestGetkeyindxDistinctKeysDiffer(t *testing.T) {
	a := getkeyindx([]byte("alpha"), 10, 100000)
	b := getkeyindx([]byte("beta"), 10, 100000)
	differ := false
	for i := range a {
		if a[i] != b[i] {
			differ = true
			break
		}
	}
	if !differ {
		t.Fatal("distinct keys produced identical indices")
	}
}

// ---- recover via scanSegment / InitBloom ----

func TestBloomRecoverFromSegments(t *testing.T) {
	dir := t.TempDir()
	p := segmentPath(dir, 1)
	writeSegRecord(t, p, TypeData, key16(1), []byte("v1"))
	writeSegRecord(t, p, TypeData, key16(2), []byte("v2"))
	writeSegRecord(t, p, TypeCommit, key16(3), []byte("v3"))

	b, err := InitBloom(1024, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, kb := range []byte{1, 2, 3} {
		k := key16(kb)
		if !b.test(k[:]) {
			t.Fatalf("key %d not recovered into bloom filter", kb)
		}
	}
	// a key that was never written should not be present
	never := key16(9)
	if b.test(never[:]) {
		t.Fatal("unwritten key reported present after recover")
	}
}

func TestBloomRecoverSkipsTornTail(t *testing.T) {
	dir := t.TempDir()
	p := segmentPath(dir, 1)
	writeSegRecord(t, p, TypeData, key16(1), []byte("v1"))
	writeSegRecord(t, p, TypeData, key16(2), []byte("v2"))
	// append a torn/partial record after the valid ones
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte{0x01, 0x02, 0x03})
	f.Close()

	b, err := InitBloom(1024, dir)
	if err != nil {
		t.Fatal(err)
	}
	k1 := key16(1)
	k2 := key16(2)
	if !b.test(k1[:]) || !b.test(k2[:]) {
		t.Fatal("valid records before a torn tail were not recovered")
	}
}

func TestInitBloomMissingDirIsEmpty(t *testing.T) {
	b, err := InitBloom(1024, filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("missing dir should not error, got %v", err)
	}
	if b.m <= 0 || b.k <= 0 {
		t.Fatal("filter must still be usable")
	}
}

func TestInitBloomEmptyDir(t *testing.T) {
	dir := t.TempDir()
	b, err := InitBloom(1024, dir)
	if err != nil {
		t.Fatal(err)
	}
	if b.test([]byte("anything")) {
		t.Fatal("empty WAL dir should yield an empty filter")
	}
}
