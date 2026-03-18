package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

type process struct{
	cmd *exec.Cmd
	stdin io.WriteCloser
	stdout io.ReadCloser
}

type snadbox interface{
	CreateNewContainer() error
	RunContainer() error
}


func Sandbox()snadbox{
	return &process{}
}

func (s *process) CreateNewContainer() error   {

	s.cmd = exec.Command("/proc/self/exe")

	

	s.cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS  |
			syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{
              ContainerID: 0,
			  HostID: 100000,
			  Size: 65536,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID: 100000,
				Size: 1,
			},
		},
		
		GidMappingsEnableSetgroups: false,

	}

	return s.RunContainer()
}

func (s *process)RunContainer() error{
 fmt.Println("inside container")

	syscall.Sethostname([]byte("supersand"))

	rootfs := "./rootfs"

	// make mount private
	syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")

	// bind mount rootfs
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		panic(err)
	}

	// chroot
	if err := syscall.Chroot(rootfs); err != nil {
		panic(err)
	}
	os.Chdir("/")

	// mount proc
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		panic(err)
	}

	// run command inside container
	s.cmd = exec.Command("/bin/sh")
    s.cmd.Stdin = os.Stdin
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr
	err :=s.cmd.Run()
	if err !=nil{
		return err
	}
	return nil
}