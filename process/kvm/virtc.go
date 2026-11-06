package kvm

import (
	"fmt"
	"unsafe"
)

// kvm intrup controller
type kvm_irq_chip struct {
	chip_id uint32
	pad     uint32
	chip    []byte
}

// kvm intrupt controller
func (k *kvm) intrupcontroller() error {
	if _, err := ioctl(uintptr(k.vmfd), KVM_CREATE_IRQCHIP, 0); err != nil {
		return fmt.Errorf("some thing bad happend on crating intrupcontroler error:%w", err)
	}
	return nil
}

// irtc mean intrupt controller
// it give the kurnal vIntrupcontroler state
func (k *kvm) getIrtcState() (kvm_irq_chip, error) {
	var def_irq_chip kvm_irq_chip
	if _, err := ioctl(uintptr(k.vmfd), KVM_GET_IRQCHIP, uintptr(unsafe.Pointer(&def_irq_chip))); err != nil {
		return def_irq_chip, fmt.Errorf(" some thing is wrong in the kvm dose not get the state of intrupt controller error: %w", err)
	}

	return def_irq_chip, nil
}

func (k *kvm) setIrtcState(state *kvm_irq_chip) error {

	if _, err := ioctl(uintptr(k.vmfd), KVM_SET_IRQCHIP, uintptr(unsafe.Pointer(state))); err != nil {
		return fmt.Errorf(" some thing went erong in kvm, not able to set kurnal Intrupt contorller state error: %w", err)
	}
	return nil
}
