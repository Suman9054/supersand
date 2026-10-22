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

type vcpu struct{
 id     int 
 vcpufd int 
 run    []byte
} 


type kvm_cpuid2 struct {
	net        uint32
	padding    uint32
	entries[max_etry_size] kvm_cpuid_entry2
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

 func (k *kvm)Getsuportedcpuid()error{
    c:=&kvm_cpuid2{net:max_etry_size}
		x,err:= ioctl(uintptr(k.kvmfd),KVM_GET_SUPPORTED_CPUID,uintptr(unsafe.Pointer(c)))
		if err != nil {
		  return fmt.Errorf("i dont why this happed error %v",err)
		}

		fmt.Print(x)
		return nil
 }

func (k *kvm) CreateVCPU(vcpunumber int)error  {
 size,err:=ioctl(uintptr(k.kvmfd),KVM_GET_VCPU_MMAP_SIZ,0)
 if err != nil {
	 return fmt.Errorf("get kvm memory size o error:%v",err)
 } 
	for i:= range vcpunumber{
		 vcpufd,err:= ioctl(uintptr(k.vmfd),KVM_CREATE_VCPU,uintptr(i))
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
    k.vcpus = append(k.vcpus, &vcpu{id: i,vcpufd: int(vcpufd),run: run})
	}

	return nil
}

func (k *kvm)closevcpu(){
 for _,v :=range k.vcpus{
   if err := unix.Munmap(v.run);err != nil{
		 slog.Error("failed to unmap vpu memory")
	 }

   if err:=unix.Close(v.vcpufd);err != nil {
		 slog.Error("failed to clsoe vcpu","error:%w",err)
	 }
	 }
}
