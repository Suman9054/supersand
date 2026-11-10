package kvm

import (
	"testing"

	"golang.org/x/sys/unix"
)

// newTestKVM opens /dev/kvm and registers cleanup automatically.
func newTestKVM(t *testing.T) *kvm {
	t.Helper()

	if err := unix.Access("/dev/kvm", unix.R_OK|unix.W_OK); err != nil {
		t.Skipf("/dev/kvm not usable: %v", err)
	}

	k, err := NewKVM()
	if err != nil {
		t.Fatalf("NewKVM: %v", err)
	}
	t.Cleanup(func() { k.Close() })
	return k
}

// newTestVM = newTestKVM + CreateVM.
func newTestVM(t *testing.T) *kvm {
	t.Helper()

	k := newTestKVM(t)
	if err := k.CreateVM(); err != nil {
		t.Fatalf("CreateVM: %v", err)
	}
	return k
}

func TestCreateVM(t *testing.T) {
	newTestVM(t)
}

func TestSetupBaseMemory(t *testing.T) {
	tests := []struct {
		name    string
		sizeMB  int
		wantErr bool
	}{
		{"1MB", 1, false},
		{"16MB", 16, false},
		{"100MB", 100, false},
		{"zero", 0, true},
		{"negative", -1, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k := newTestVM(t)

			err := k.SetupbaseMemory(tc.sizeMB)
			if (err != nil) != tc.wantErr {
				t.Fatalf("SetupbaseMemory(%d) err=%v, wantErr=%v", tc.sizeMB, err, tc.wantErr)
			}
			if err == nil && k.memory.size != tc.sizeMB {
				t.Fatalf("size = %d, want %d", k.memory.size, tc.sizeMB)
			}
		})
	}
}
