package kvm

import "golang.org/x/sys/unix"

// ioctl constants
const (
	KVMIO = 0xAE

	KVM_GET_API_VERSION        = 0xAE00
	KVM_CREATE_VM              = 0xAE01
	KVM_CREATE_VCPU            = 0xAE41
	KVM_SET_USER_MEMORY_REGION = 0x4020AE46
	KVM_RUN                    = 0xAE80
  KVM_GET_VCPU_MMAP_SIZ      =  0xAE04 

	// x86 KVM exit reasons
	KVM_EXIT_HLT = 5
)

func ioctl(fd uintptr, request uint, arg uintptr) (uintptr, error) {
	ret, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(request), arg)
	if errno != 0 {
		return 0, errno
	}
	return ret, nil
}
