package kvm

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

// lets move to kvm based containerization with samll footprint and better performance

type kvm struct {
	id     uuid.UUID
	kvmfd  int
	vmfd   int
	vcpus  []*vcpu
	memory kvmbaseMemory
}

type KVM interface {
	CreateVM() error
	Close() error
	Getsuportedcpuid() error
	CreateVCPU(vcpunumber int) error
}

func NewKVM() (KVM, error) {

	fd, err := unix.Open("~/dev/kvm", unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open /dev/kvm: %v", err)
	}
	return &kvm{kvmfd: fd}, nil
}

func (k *kvm) CreateVM() error {
	vmFd, err := ioctl(uintptr(k.kvmfd), KVM_CREATE_VM, 0)
	if err != nil {
		return fmt.Errorf("failed to create VM: %v", err)
	}
	k.vmfd = int(vmFd)
	k.id = uuid.New()
	slog.Info("VM created successfully", "vmFd", vmFd)
	return nil
}

func (k *kvm) Close() error {
	if err := unix.Close(k.kvmfd); err != nil {
		return fmt.Errorf("failed to close KVM fd: %v", err)
	}
	return nil
}
