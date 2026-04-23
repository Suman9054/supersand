package process

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func Runcontaner(root string) error {
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("error in mounting", "error", err)
	}
	if err := os.Chdir(root); err != nil {
		return fmt.Errorf("error in changing directory", "error", err)
	}
	os.Mkdir(".oldroot", 0o700)
	if err := unix.PivotRoot(".", ".oldroot"); err != nil {
		return fmt.Errorf("error in PivotRoot setup", "error", err)
	}
	os.Chdir("/")
	syscall.Unmount(".oldroot", syscall.MNT_DETACH)
	os.Remove(".oldroot")

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("error in prock mounting", "error", err)
	}

	if err := syscall.Mount("/dev", "/dev", "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("error in mounting dev", "error", err)
	}
	if err := syscall.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ()); err != nil {
		return fmt.Errorf("error in sycall bash call", "error", err)
	}
	return nil
}
