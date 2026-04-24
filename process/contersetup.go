package process

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func Runcontaner(root string) error {
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("error in mounting %w", err)
	}
	if err := os.Chdir(root); err != nil {
		return fmt.Errorf("error in changing directory %w", err)
	}
	os.Mkdir(".oldroot", 0o700)
	if err := unix.PivotRoot(".", ".oldroot"); err != nil {
		return fmt.Errorf("error in PivotRoot setup %w", err)
	}
	os.Chdir("/")
	syscall.Unmount(".oldroot", syscall.MNT_DETACH)
	os.Remove(".oldroot")

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("error in prock mounting %w", err)
	}

	if err := syscall.Mount("/dev", "/dev", "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("error in mounting dev %w", err)
	}
	if err := syscall.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ()); err != nil {
		return fmt.Errorf("error in sycall bash call %w", err)
	}
	return nil
}
