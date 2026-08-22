package store

//bloomfilter is just an aray of bits. we going to pass the key by an hash function it will return an index
// then we will just set the corrosponding bit 1 on the array

import (
	"github.com/spaolacci/murmur3"
)

type bloom struct {
	b    []byte
	size int
}

func newBloom(size int) *bloom {
	return &bloom{
		b:    make([]byte, size),
		size: size,
	}
}

func (b *bloom) add(key string) {

}
