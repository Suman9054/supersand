package process

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
	"github.com/suman9054/supersand/healper"
)

type ContanerConf struct {
	Id     uuid.UUID
	PID    int
	Status healper.Status
}

// CreateNewContainer spawns the container child process inside a new set of
// Linux namespaces and attaches a pseudo-terminal to it. It blocks until the
// child either signals readiness (fd 3 closed with no message) or reports
// an error / dies during setup.
func (s *Process) CreateNewContainer() (error, ContanerConf) {
	s.id = uuid.New()
	var v ContanerConf
	rootfs, err := s.SetupFilesystem()
	if err != nil {
		return err, v
	}

	cmd := exec.Command("/proc/self/exe", "child", rootfs)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
	}

	// Readiness pipe: child closes fd 3 once namespace/mount setup succeeds
	// and it's about to exec. If setup fails first, it writes "ERR:<msg>"
	// to fd 3 before exiting, so we get a real diagnostic instead of a
	// bare EOF.
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("error creating readiness pipe: %w", err), v
	}
	cmd.ExtraFiles = []*os.File{w}
	ptx, err := pty.Start(cmd)
	w.Close()
	if err != nil {
		r.Close()
		return fmt.Errorf("error starting container: %w", err), v
	}

	msg, _ := io.ReadAll(r)
	r.Close()

	if len(msg) > 0 {
		// Child wrote an error before dying — surface it directly.
		_ = cmd.Wait()
		_ = ptx.Close()
		return fmt.Errorf("child setup failed: %s", string(msg)), v
	}

	// msg is empty and the pipe hit EOF: either the child closed fd 3
	// cleanly (success — proceed), or it died with no write at all (e.g.
	// killed by a signal, OOM, etc). Distinguish via Wait: if the process
	// has already exited, ProcessState will be non-nil after Wait returns.
	select {
	case <-time.After(0):
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		// Process is already gone and wrote nothing — reap it and report.
		_ = cmd.Wait()
		_ = ptx.Close()
		return fmt.Errorf("child exited during setup with no diagnostic (likely signaled)"), v
	}

	if err := s.setupCgroup(); err != nil {
		// Non-fatal: log and continue; container still runs without limits.
		slog.Warn("cgroup setup failed", "error", err)
	}

	// Watch for unexpected container death in the background.
	go func() {
		if err := cmd.Wait(); err != nil {
			slog.Error("container crashed", "error", err)
		}
		s.mu.Lock()
		s.status = healper.Stopped
		_ = ptx.Close()
		s.mu.Unlock()
	}()

	s.mu.Lock()
	s.status = healper.Active
	s.mu.Unlock()
	return nil, ContanerConf{
		Id:     s.id,
		PID:    cmd.Process.Pid,
		Status: healper.Active,
	}
}
