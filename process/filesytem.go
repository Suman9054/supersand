package process

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suman9054/supersand/healper"
	"golang.org/x/sys/unix"
)

func (s *Process) SetupFilesystem() (string, error) {
	rootfs := fmt.Sprintf("%s/croot/%s_rootfs", healper.FOlderpath, s.id.String())

	if err := os.MkdirAll(rootfs, 0o755); err != nil {
		return "", fmt.Errorf("faild to creat rootfs:%v", err)
	}
	folders := [9]string{"bin", "dev", "etc", "proc", "sys", "tmp", "usr", "var", "work"}
	for _, fl := range folders {
		fls := fmt.Sprintf("%s/%s", rootfs, fl)
		if err := os.MkdirAll(fls, 0o755); err != nil {
			os.RemoveAll(rootfs)
			return "", fmt.Errorf("faild to creat %s : %v", fl, err)
		}
	}
	null := filepath.Join(rootfs, "dev", "null")
	zero := filepath.Join(rootfs, "dev", "zero")
	random := filepath.Join(rootfs, "dev", "random")
	urandom := filepath.Join(rootfs, "dev", "urandom")

	if err := unix.Mknod(null, unix.S_IFCHR|0o666, int(unix.Mkdev(1, 3))); err != nil {
		return "", fmt.Errorf("faild to creat rootfs/dev/null:%v", err)
	}
	if err := unix.Mknod(zero, unix.S_IFCHR|0o666, int(unix.Mkdev(1, 5))); err != nil {
		return "", fmt.Errorf("faild to creat rootfs/dev/zero:%v", err)
	}
	if err := unix.Mknod(random, unix.S_IFCHR|0o666, int(unix.Mkdev(1, 8))); err != nil {
		return "", fmt.Errorf("faild to creat rootfs:%v", err)
	}
	if err := unix.Mknod(urandom, unix.S_IFCHR|0o666, int(unix.Mkdev(1, 9))); err != nil {
		return "", fmt.Errorf("faild to creat rootfs:%v", err)
	}

	bin := filepath.Join(rootfs, "bin")
	base := "/home/suman/supersand/template/base/rootfs-busy"
	basebin := filepath.Join(base, "bin")
	baseusr := filepath.Join(base, "usr")

	usr := filepath.Join(rootfs, "usr")
	if err := unix.Mount(basebin, bin, "", uintptr(unix.MS_BIND|unix.MS_RDONLY), ""); err != nil {
		unix.Unmount(bin, 0)
		os.RemoveAll(rootfs)
		return "", fmt.Errorf("faild to mount rootfs/bin: %v", err)
	}
	if err := unix.Mount(baseusr, usr, "", uintptr(unix.MS_BIND|unix.MS_RDONLY), ""); err != nil {
		unix.Unmount(bin, 0)
		unix.Unmount(usr, 0)
		os.RemoveAll(rootfs)
		return "", fmt.Errorf("faild to mount rootfs/usr: %v", err)
	}

	return rootfs, nil
}
