package kvm

import (
	"fmt"
	"unsafe"
)

const (
	KVM_NR_INTERRUPTS = 256
)

// for x-86
type kvm_regs struct {
	rax, rbx, rcx, rdx uint64
	rsi, rdi, rsp, rbp uint64
	r8, r9, r10, r11   uint64
	r12, r13, r14, r15 uint64
	rip, rflags        uint64
}

type kvm_sregs struct {
	cs, ds, es, fs, gs, ss  kvm_segment
	tr, ldt                 kvm_segment
	gdt, idt                kvm_dtable
	cr0, cr2, cr3, cr4, cr8 uint64
	efer                    uint64
	apic_base               uint64
	interrupt_bitmap        [(KVM_NR_INTERRUPTS + 63) / 64]uint64
}

type kvm_segment struct {
	Base     uint64
	Limit    uint32
	Selector uint16
	Type     uint8
	Present  uint8
	DPL      uint8
	DB       uint8
	S        uint8
	L        uint8
	G        uint8
	AVL      uint8
	Unusable uint8
	Padding  uint8
}

type kvm_dtable struct {
	Base    uint64
	Limit   uint16
	Padding [3]uint16
}

func (k *kvm) set_regs(regs []kvm_regs) error {
	for i, v := range k.vcpus {
		if _, err := ioctl(uintptr(v.vcpufd), KVM_SET_REGS, uintptr(unsafe.Pointer(&regs[i]))); err != nil {
			return fmt.Errorf(" some thing went wrong on setting up register for %d vcpu  error:%w", v.vcpufd, err)
		}
	}
	return nil
}

func (k *kvm) get_regs() ([]kvm_regs, error) {
	var kvm_regs_arr []kvm_regs
	var kvm_regs_em []kvm_regs
	for _, v := range k.vcpus {
		var kvm_regsi kvm_regs
		if _, err := ioctl(uintptr(v.vcpufd), KVM_GET_REGS, uintptr(unsafe.Pointer(&kvm_regsi))); err != nil {
			return kvm_regs_em, fmt.Errorf("not able to get register error:%w", err)
		}
		kvm_regs_arr = append(kvm_regs_arr, kvm_regsi)
	}
	return kvm_regs_arr, nil
}

func (k *kvm) setSregs(sregs []kvm_sregs) error {

	for i, v := range k.vcpus {
		if _, err := ioctl(uintptr(v.vcpufd), KVM_SET_SREGS, uintptr(unsafe.Pointer(&sregs[i]))); err != nil {
			return fmt.Errorf("not able to set sregisters error:%w", err)
		}
	}
	return nil
}

func (k *kvm) getSregs() ([]kvm_sregs, error) {
	var def_kvm_sregs []kvm_sregs
	var emt_kvm_sregs []kvm_sregs

	for _, v := range k.vcpus {
		var kvm_sragsi kvm_sregs

		if _, err := ioctl(uintptr(v.vcpufd), KVM_GET_SREGS, uintptr(unsafe.Pointer(&kvm_sragsi))); err != nil {
			return emt_kvm_sregs, fmt.Errorf("not able to get the s-regs error:%w", err)
		}
		def_kvm_sregs = append(def_kvm_sregs, kvm_sragsi)
	}
	return def_kvm_sregs, nil
}
