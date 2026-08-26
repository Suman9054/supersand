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
// newBloom init an bloom filter it automaticly detect what will be best number of hash function and 
// bit requard i can make it as uint64 array but i dont want to waste most of the bits so i chose uint8 it will
// slow to read but it will be fine and it is very simple to implement.
func newBloom(size int64) *bloom {

	max:= - float64(size)*math.Log(0.001)/ math.Pow(math.Log(2),2)
  
	axh:= (max/float64(size))*math.Log(2)

	return &bloom{
		bitarr:    make([]uint8, int(max)),
	  m:    int(max),
		k:    int8(axh),
	}
}

// getkeyindx i can make it like opps but it is just an privet function for only for hash keys 
// it just return an slice of uit64 
func getkeyindx(key []byte,k int,m int8)[]uint64{


	//i chose xxh3 hash becuase it is very frist compire to any hash algo 

	h1:=xxh3.Hash(key)
	h2:=xxh3.Hash(key)
	var indx []uint64

  //this just idx(i) = (h1(hash1) +i*h2(hash2))% m(size of the array) 
	// i is just 0..k(number of hash function)
	
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
