package kvm

import "golang.org/x/sys/unix"

// ioctl constants
const (
	KVMIO                      = 0xAE
	KVM_GET_API_VERSION        = 0xAE00
	KVM_CREATE_VM              = 0xAE01
	KVM_CREATE_VCPU            = 0xAE41
	KVM_SET_USER_MEMORY_REGION = 0x4020AE46
	KVM_RUN                    = 0xAE80
	KVM_GET_VCPU_MMAP_SIZ      = 0xAE04
	KVM_GET_SUPPORTED_CPUID    = 0xC008AE05 // on /dev/kvm fd
	KVM_SET_CPUID2             = 0x4008AE90 // on each vCPU fd
	KVM_SET_CLOCK              = 0x4030AE7B
	KVM_GET_CLOCK              = 0x8030AE7C
	KVM_CREATE_IRQCHIP         = 0xAE60
	KVM_IRQ_LINE               = 0x4008AE61
	KVM_GET_IRQCHIP            = 0xC208AE62
	KVM_SET_IRQCHIP            = 0x8008AE63
	KVM_GET_REGS               = 0x8090AE81
	KVM_SET_REGS               = 0x4090AE82
	KVM_GET_SREGS              = 0x8138AE83
	KVM_SET_SREGS              = 0x4138AE84
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
