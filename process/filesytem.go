package process

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func SetupFilesystem(UUid string) error {
	rootfs := fmt.Sprintf("sandinternal/v1_supersand/%s_rootfs", UUid)

	if err := os.MkdirAll(rootfs, 0o755); err != nil {
		return fmt.Errorf("faild to creat rootfs:%v", err)
	}
	folders := [8]string{"bin", "dev", "etc", "proc", "sys", "tmp", "usr", "var"}
	for _, fl := range folders {
		fls := fmt.Sprintf("%s/%s", rootfs, fl)
		if err := os.MkdirAll(fls, 0o755); err != nil {
			os.RemoveAll(rootfs)
			return fmt.Errorf("faild to creat %s : %v", fl, err)
		}
	}
	bin := fmt.Sprintf("%s/bin", rootfs)
	base := "../template/base/rootfs-busy"
	basebin := fmt.Sprintf("%s/bin", base)
	baseusr := fmt.Sprintf("%s/usr", base)
	flags := unix.MS_RDONLY
	usr := fmt.Sprintf("%s/usr", rootfs)
	if err := unix.Mount(basebin, bin, "", uintptr(flags), ""); err != nil {
		unix.Unmount(bin, 0)
		os.RemoveAll(rootfs)
		return fmt.Errorf("faild to mount rootfs/bin: %v", err)
	}
	if err := unix.Mount(baseusr, usr, "", uintptr(flags), ""); err != nil {
		unix.Unmount(bin, 0)
		unix.Unmount(usr, 0)
		os.RemoveAll(rootfs)
		return fmt.Errorf("faild to mount rootfs/usr: %v", err)
	}

	return nil
}
