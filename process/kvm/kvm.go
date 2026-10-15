package kvm

import (
	"fmt"
	"log/slog"

	"golang.org/x/sys/unix"
)

// lets move to kvm based containerization with samll footprint and better performance





type kvm struct {
	fd int
}

type KVM interface{
  CreateVM() error 
	Close() error 
}



func NewKVM() (KVM, error) {
	fd, err := unix.Open("/dev/kvm", unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open /dev/kvm: %v", err)
	}
	return &kvm{fd: fd}, nil
}


func (k *kvm) CreateVM() error {
	vmFd, err := ioctl(uintptr(k.fd), KVM_CREATE_VM, 0)
	if err != nil {
		return  fmt.Errorf("failed to create VM: %v", err)
	}
	slog.Info("VM created successfully", "vmFd", vmFd)
	return nil
}


func (k *kvm) Close() error {
	if err := unix.Close(k.fd); err != nil {
		return fmt.Errorf("failed to close KVM fd: %v", err)
	}
	return nil
}


