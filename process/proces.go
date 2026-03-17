package process

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func CreateNewContainer() *exec.Cmd {
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
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

	return cmd
}

func runContainer() {
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
	os.Chdir("/user")

	// mount proc
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		panic(err)
	}

	// run command inside container
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
}