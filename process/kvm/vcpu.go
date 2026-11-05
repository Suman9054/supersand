package kvm

import (
	"fmt"
	"log/slog"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	max_etry_size = 256
)

type vcpu struct {
	id     int
	vcpufd int
	run    []byte
}

type kvm_cpuid2 struct {
	net     uint32
	padding uint32
	entries [max_etry_size]kvm_cpuid_entry2
}

type kvm_cpuid_entry2 struct {
	function uint32
	index    uint32
	flags    uint32
	eax      uint32
	ebx      uint32
	ecx      uint32
	edx      uint32
	padding  [3]uint32
}

type kvm_clock_data struct {
	clocks   uint64
	flags    uint32
	pad0     uint32
	realtime uint64
	host_tsc uint64
	pad      [4]uint32
}

type kvm_irq_chip struct {
	chip_id uint32
	pad     uint32
	chip    []byte
}

func (k *kvm) setupclock(clock *kvm_clock_data) error {
	if _, err := ioctl(uintptr(k.vmfd), KVM_SET_CLOCK, uintptr(unsafe.Pointer(clock))); err != nil {

		return fmt.Errorf("i dont know some thing went wrong in kvm clock setup error:%v", err)
	}
	return nil
}

func (k *kvm) getclock() (kvm_clock_data, error) {
	var defclock kvm_clock_data

	if _, err := ioctl(uintptr(k.vmfd), KVM_GET_CLOCK, uintptr(unsafe.Pointer(&defclock))); err != nil {
		return defclock, fmt.Errorf("i dint know some thing went wrong in kvm clock get error %v", err)
	}
	return defclock, nil
}

func (k *kvm) getsuportedcpuid() (kvm_cpuid2, error) {
	var defval kvm_cpuid2
	c := &kvm_cpuid2{net: max_etry_size}
	x, err := ioctl(uintptr(k.kvmfd), KVM_GET_SUPPORTED_CPUID, uintptr(unsafe.Pointer(c)))
	if err != nil {
		return defval, fmt.Errorf("i dont why this happed error %v", err)
	}

	fmt.Printf("this the value for x %#x\n", x)
	fmt.Printf("Total CPUID entries: %d\n", c.net)

	for i := uint32(0); i < c.net; i++ {
		entry := c.entries[i]

		fmt.Printf(
			"Function: %#x EAX: %#x EBX: %#x ECX: %#x EDX: %#x\n",
			entry.function,
			entry.eax,
			entry.ebx,
			entry.ecx,
			entry.edx,
		)
	}
	defval = *c
	return defval, nil
}

func (k *kvm) setcpuid2(c *kvm_cpuid2) error {
	for _, v := range k.vcpus {
		if _, err := ioctl(uintptr(v.vcpufd), KVM_SET_CPUID2, uintptr(unsafe.Pointer(c))); err != nil {
			return fmt.Errorf("some thing went wrong in the cpuid setup error:", err)
		}
	}
	return nil
}

func (k *kvm) createVCPU(vcpunumber int) error {
	size, err := ioctl(uintptr(k.kvmfd), KVM_GET_VCPU_MMAP_SIZ, 0)
	if err != nil {
		return fmt.Errorf("get kvm memory size o error:%v", err)
	}
	for i := range vcpunumber {
		vcpufd, err := ioctl(uintptr(k.vmfd), KVM_CREATE_VCPU, uintptr(i))
		if err != nil {
			k.closevcpu()
			return fmt.Errorf(" some thing went worg in vcpu creation")
		}
		run, err := unix.Mmap(int(vcpufd), 0, int(size),
			unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
		if err != nil {
			unix.Close(int(vcpufd))
			k.closevcpu()
			return fmt.Errorf("mmap kvm_run %d: %w", i, err)
		}
		k.vcpus = append(k.vcpus, vcpu{id: i, vcpufd: int(vcpufd), run: run})
	}

	return nil
}

func (k *kvm) closevcpu() {
	for _, v := range k.vcpus {
		if err := unix.Munmap(v.run); err != nil {
			slog.Error("failed to unmap vpu memory")
		}

		if err := unix.Close(v.vcpufd); err != nil {
			slog.Error("failed to clsoe vcpu", "error:%w", err)
		}
	}
}
