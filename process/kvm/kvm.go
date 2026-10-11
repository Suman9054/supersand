package kvm

import (
	"fmt"
	"log/slog"

	"golang.org/x/sys/unix"
)

// lets move to kvm based containerization with samll footprint and better performance

// ioctl constants
const (
	KVMIO = 0xAE

	KVM_GET_API_VERSION = 0xAE00
	KVM_CREATE_VM       = 0xAE01
	KVM_CREATE_VCPU     = 0xAE41
	KVM_SET_USER_MEMORY_REGION = 0x4020AE46
	KVM_RUN             = 0xAE80

	// x86 KVM exit reasons
	KVM_EXIT_HLT = 5
)

type UserMemoryRegion struct {
    Slot          uint32
    Flags         uint32
    GuestPhysAddr uint64
    MemorySize    uint64
    UserspaceAddr uint64
}

func ioctl(fd uintptr, request uint, arg uintptr) (uintptr, error) {
	ret, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(request), arg)
	if errno != 0 {
		return 0, errno
	}
	return ret, nil
}


type KVM struct {
	fd int
}

type memoryRegion struct {
	slot          uint32
	flags         uint32
	guestPhysAddr uint64
	memorySize    uint64
	userspaceAddr uint64
}


func NewKVM() (*KVM, error) {
	fd, err := unix.Open("/dev/kvm", unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open /dev/kvm: %v", err)
	}
	return &KVM{fd: fd}, nil
}


func (k *KVM) CreateVM() (int, error) {
	vmFd, err := ioctl(uintptr(k.fd), KVM_CREATE_VM, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to create VM: %v", err)
	}
	slog.Info("VM created successfully", "vmFd", vmFd)
	return int(vmFd), nil
}