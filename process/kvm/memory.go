package kvm

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

// kvm base memory it the base memory block for every kvm size canbe varied based on kvm need at creation time
type kvmbaseMemory struct {
	size          int
	memfd         int
	gestphysaddr  uint64
	userspaceAddr uint64
	raw           []byte
}

// this is structer for memory register in kvm
type UserMemoryRegion struct {
	Slot          uint32
	Flags         uint32
	GuestPhysAddr uint64
	MemorySize    uint64
	UserspaceAddr uint64
}

// this is 2mb thf anonmus mapd memory regions for dynamic memory asingment to kvm
type region struct {
	slot    uint32 // ram slot
	gstpadr uint32 // gest physical addr
	mem     []byte // 2mb memory
	full    []byte // raw byts
}

type memoryPool struct {
	base map[int]*region // it is for memory region search
	free []int           // just stack of memory region id (fifo)
}

func roundup(x, a uint64) uint64 {
	return (x + a - 1) &^ (a - 1)
}

func (k *kvm) SetupbaseMemory(size int) error {
	fd, err := unix.MemfdCreate("kvm base-memory", unix.MFD_CLOEXEC)
	if err != nil {
		unix.Close(fd)
		return fmt.Errorf("failed to creat base-memory error:%v", err)
	}
	if err := unix.Ftruncate(fd, int64(size)); err != nil {
		unix.Close(fd)
		return fmt.Errorf("failed to creat base-memory error:%v", err)
	}

	ram, errmap := unix.Mmap(fd, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANONYMOUS)
	if errmap != nil {
		unix.Close(fd)
		unix.Munmap(ram)
	}
	uspaddr := uint64(uintptr(unsafe.Pointer(&ram[0])))
	usrmemr := &UserMemoryRegion{
		Slot:          0,
		Flags:         0,
		GuestPhysAddr: 0x0,
		MemorySize:    uint64(size),
		UserspaceAddr: uspaddr,
	}
	if _, errio := ioctl(uintptr(k.vmfd), KVM_SET_USER_MEMORY_REGION, uintptr(unsafe.Pointer(usrmemr))); errio != nil {
		unix.Close(fd)
		unix.Munmap(ram)
		return fmt.Errorf("failed to creat base-memory error:%v", errio)
	}
	memoryinfo := kvmbaseMemory{
		size:          size,
		memfd:         fd,
		gestphysaddr:  0x0,
		userspaceAddr: uspaddr,
	}
	k.memory = memoryinfo
	return nil
}
