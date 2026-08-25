package store

import (
	"math"
	"github.com/zeebo/xxh3"
)

//bloomfilter is just an aray of bits. we going to pass the key by an hash function it will return an index
// then we will just set the corrosponding bit 1 on the array

type bloom struct {
	bitarr    []uint8
  m     int    // number of bits requard for bloom filter 
	k     int8     // number of hash function requard 
}

func newBloom(size int64) *bloom {

	max:= - float64(size)*math.Log(0.001)/ math.Pow(math.Log(2),2)
  
	axh:= (max/float64(size))*math.Log(2)

	return &bloom{
		bitarr:    make([]uint8, int(max)),
	  m:    int(max),
		k:    int8(axh),
	}
}


func getkeyindx(key []byte,k int,m int8)[]uint64{

	h1:=xxh3.Hash(key)
	h2:=xxh3.Hash(key)
	var indx []uint64
  
	for i:=0;i<k;i++{
		xi:=(h1 + uint64(i)*h2)%uint64(m)
   indx= append(indx, xi)
	}
	return indx
}

func (b *bloom) add(key string) {
   
	  indexs:=getkeyindx([]byte(key),int(b.k),int8(b.m))

		for _,v:= range indexs{
			byteindex:= v /uint64(8)
			bitindex :=v%uint64(8)
			b.bitarr[byteindex] |=uint8(1<<bitindex)
		}
} 


func (b *bloom) test(key string) bool{
	indexs:=getkeyindx([]byte(key),int(b.k),int8(b.m))

	for _,v:= range indexs{
		byteindex:=v/uint64(8)
		bitindex:=v%uint64(8)
		bitmask:=uint8(1<<bitindex)
		if b.bitarr[byteindex]&bitmask == 0 {
			return false
		}
	}
	return true
}
