package kvm

import (
	
	"testing"
)

func TestCreateVM(t *testing.T) {
	kvm, err := NewKVM()
	if err != nil {
		t.Fatalf("Failed to create KVM instance: %v", err)
	}

	 err = kvm.CreateVM()
	if err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}
	if err := kvm.Getsuportedcpuid();err != nil {
		t.Fatalf("fild to get supoorted cpuid %v",err)
	}

	
	
}
