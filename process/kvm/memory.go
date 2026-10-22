package kvm

// this is to stup the memory region for the kvm and setup shard memory for ipc
type UserMemoryRegion struct {
	Slot          uint32
	Flags         uint32
	GuestPhysAddr uint64
	MemorySize    uint64
	UserspaceAddr uint64
}

// this is 2mb thf memory regions for dynamic memory asingment to kvm
type region struct {
	sloat   uint32 // ram sloat
	gstpadr uint32 // gest physical addr
	mem     []byte // 2mb memory page
	full    []byte // raw bites
}

type memoryPool struct {
	base map[int]*region // it is for memory region search
	free []int           // just stack of memory region id (fifo)
}

func roundup(x, a uint64) uint64 {
	return (x + a - 1) &^ (a - 1)
}

func (k *kvm) Setupmemory(size int) error {

	return nil
}
