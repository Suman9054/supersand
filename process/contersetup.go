package process

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// fail writes a diagnostic message to the readiness pipe (fd 3) and exits.
func fail(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if f := os.NewFile(3, "ready"); f != nil {
		fmt.Fprintf(f, "ERR:%s", msg)
		f.Close()
	}
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func Runcontaner(root string) error {
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		fail("error making / private: %v", err)
	}

	// resolving the acutal file path.
	absRoot, err := filepath.Abs(root)
	if err != nil {
		fail("error resolving absolute path for %s: %v", root, err)
	}

	if err := syscall.Mount(absRoot, absRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		fail("error bind-mounting rootfs onto itself: %v", err)
	}
	if err := os.Chdir(absRoot); err != nil {
		fail("error changing directory to %s: %v", absRoot, err)
	}

	if err := os.Mkdir(".oldroot", 0o700); err != nil {
		fail("error creating .oldroot: %v", err)
	}
	if err := unix.PivotRoot(".", ".oldroot"); err != nil {
		fail("error in pivot_root: %v", err)
	}
	if err := os.Chdir("/"); err != nil {
		fail("error chdir to new /: %v", err)
	}
	if err := syscall.Unmount(".oldroot", syscall.MNT_DETACH); err != nil {
		fail("error unmounting .oldroot: %v", err)
	}
	os.Remove(".oldroot")

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		fail("error mounting /proc: %v", err)
	}

	if readyFd := os.NewFile(3, "ready"); readyFd != nil {
		readyFd.Close()
	}

	if err := syscall.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "EXEC FAILED: %v\n", err)
		os.Exit(1)
	}
	return nil
}
