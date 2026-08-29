package store

import (
	"errors"
	"fmt"
	"io/fs"
	"math"

	"github.com/zeebo/xxh3"
)

// bloom is a bit array. We pass the key through a hash function which returns
// an index, then set the corresponding bit. Reads are O(k) with k hash probes.

type bloom struct {
	bitarr []uint8
	m      int // number of bits in the filter
	k      int // number of hash functions
}

// newBloom sizes a filter for the expected element count `size`. It picks the
// optimal bit count m = -n*ln(p)/(ln2)^2 (with p = 0.001) and optimal hash
// count k = (m/n)*ln2. The backing array holds ceil(m/8) bytes.
func newBloom(size int64) *bloom {
	if size <= 0 {
		size = 1
	}

	bits := -float64(size) * math.Log(0.001) / math.Pow(math.Log(2), 2)
	k := (bits / float64(size)) * math.Log(2)

	m := int(bits)
	if m < 1 {
		m = 1
	}
	kh := int(k)
	if kh < 1 {
		kh = 1
	}

	return &bloom{
		bitarr: make([]uint8, (m+7)/8),
		m:      m,
		k:      kh,
	}
}

// getkeyindx returns the k bit indices for key using double hashing:
// idx(i) = (h1 + i*h2) % m. Two independent 64-bit halves of the xxh3 128-bit
// hash are used so the probes are well distributed.
func getkeyindx(key []byte, k int, m int) []uint64 {
	h := xxh3.Hash128(key)
	h1 := h.Lo
	h2 := h.Hi
	// guard against a zero h2, which would collapse every probe to h1
	if h2 == 0 {
		h2 = 1
	}

	var indx []uint64
	for i := 0; i < k; i++ {
		xi := (h1 + uint64(i)*h2) % uint64(m)
		indx = append(indx, xi)
	}
	return indx
}

func (b *bloom) add(key []byte) {
	for _, v := range getkeyindx(key, b.k, b.m) {
		byteindex := v / 8
		bitindex := v % 8
		b.bitarr[byteindex] |= uint8(1) << uint(bitindex)
	}
}

func (b *bloom) test(key []byte) bool {
	for _, v := range getkeyindx(key, b.k, b.m) {
		byteindex := v / 8
		bitindex := v % 8
		if b.bitarr[byteindex]&(uint8(1)<<uint(bitindex)) == 0 {
			return false
		}
	}
	return true
}

// recover rebuilds the filter's set of present keys by scanning every WAL
// segment on disk. This mirrors how openwal recovers its offset-lookup map by
// scanning segments at startup: each valid record's key is marked present.
// A key that was later deleted may still be marked, which only ever yields an
// occasional false positive (the real existence check still goes through the
// WAL/store), so that over-approximation is acceptable here.
func (b *bloom) recover(segments []segmentMeta) error {
	for _, s := range segments {
		_, _, offsetlookup, err := scanSegment(s.path)
		if err != nil {
			return fmt.Errorf("bloom: recover scan %s: %w", s.path, err)
		}
		for key := range offsetlookup {
			b.add(key[:])
		}
	}
	return nil
}

// InitBloom creates a bloom filter sized for the expected element count. When
// dir points at an existing WAL directory it recovers the filter state from the
// on-disk segments via recover. A missing dir is treated as "nothing to
// recover" and yields an empty filter. This parallels Initwal/openwal, which
// reopen a WAL by listing and scanning its segments.
func InitBloom(size int64, dir string) (*bloom, error) {
	b := newBloom(size)
	if dir == "" {
		return b, nil
	}
	segs, err := listSegments(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return b, nil
		}
		return nil, err
	}
	if err := b.recover(segs); err != nil {
		return nil, err
	}
	return b, nil
}
