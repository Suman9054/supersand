package kvm

import (
	"fmt"
	"log/slog"

	"golang.org/x/sys/unix"
)


type vcpu struct{
 id     int 
 vcpufd int 
 run    []byte
} 

func (k *kvm) CreateVCPU(vcpunumber int)error  {
 size,err:=ioctl(uintptr(k.vmfd),KVM_GET_VCPU_MMAP_SIZ,0)
 if err != nil {
	 return fmt.Errorf("get kvm memory size o error:%w",err)
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
