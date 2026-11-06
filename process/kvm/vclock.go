package kvm

import (
	"fmt"
	"unsafe"
)

type kvm_clock_data struct {
	clocks   uint64
	flags    uint32
	pad0     uint32
	realtime uint64
	host_tsc uint64
	pad      [4]uint32
}

func (k *kvm) setupclock(clock *kvm_clock_data) error {
	if _, err := ioctl(uintptr(k.vmfd), KVM_SET_CLOCK, uintptr(unsafe.Pointer(clock))); err != nil {

		return fmt.Errorf("i dont know some thing went wrong in kvm clock setup error:%w", err)
	}
	return nil
}

func (k *kvm) getclock() (kvm_clock_data, error) {
	var defclock kvm_clock_data

	if _, err := ioctl(uintptr(k.vmfd), KVM_GET_CLOCK, uintptr(unsafe.Pointer(&defclock))); err != nil {
		return defclock, fmt.Errorf("i dint know some thing went wrong in kvm clock get error %w", err)
	}
	return defclock, nil
}
