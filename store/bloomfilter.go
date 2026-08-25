package store

import "sync"

//bloomfilter is just an aray of bits. we going to pass the key by an hash function it will return an index
// then we will just set the corrosponding bit 1 on the array

type bloom struct {
	b    []uint64
 rw sync.RWMutex
 m uint64
	size int
	k uint64
	n uint64
}

func newBloom(size int) *bloom {
	return &bloom{
		b:    make([]uint64, size),
		size: size,
	}
}

func (b *bloom) add(key string) {
   
} 
