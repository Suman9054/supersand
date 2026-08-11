package process

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/suman9054/supersand/healper"
	"golang.org/x/sys/unix"
)

var (
	ansiRegx    = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)              // strips ANSI escape codes
	promptRegx  = regexp.MustCompile(`(?m)^[^\s]*[\$#]\s`)                 // strips shell prompts
	readBufPool = sync.Pool{New: func() any { return make([]byte, 4096) }} // 4 KB read buffers
)

type ContanerConf struct {
	Id     string
	PID    int
	Status healper.Status
}

// CreateNewContainer spawns the container child process inside a new set of
// Linux namespaces and attaches a pseudo-terminal to it. It blocks until the
// child either signals readiness (fd 3 closed with no message) or reports
// an error / dies during setup.
func (s *Process) CreateNewContainer() (error, ContanerConf) {
	contanerid := healper.GenrateRandomUUid()
	var v ContanerConf
	if err := SetupFilesystem(contanerid); err != nil {
		return err, v
	}
	rootfs := fmt.Sprintf("/etc/sandin/croot/%s_rootfs", contanerid)

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

	if err := setupCgroup(cmd.Process.Pid); err != nil {
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
	s.cmd = cmd
	s.f = ptx
	s.id = contanerid
	s.status = healper.Active
	s.mu.Unlock()
	return nil, ContanerConf{
		Id:     contanerid,
		PID:    cmd.Process.Pid,
		Status: healper.Active,
	}
}

// RunCommand writes a command to the container's PTY, reads back all output
// until a unique sentinel string appears, and returns the cleaned result.
func (s *Process) RunCommand(command string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != healper.Active {
		return "", fmt.Errorf("container is not running")
	}
	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	// Unique sentinel that marks the end of this command's output.
	sentinel := fmt.Sprintf("__done__%d__", time.Now().UnixNano())
	fullCmd := command + "; echo " + sentinel + "\n"

	if _, err := s.f.Write([]byte(fullCmd)); err != nil {
		return "", fmt.Errorf("write command: %w", err)
	}

	done := make(chan response, 1)

	go func() {
		buf := readBufPool.Get().([]byte)
		defer readBufPool.Put(buf)

		var output strings.Builder
		sentinelBytes := []byte(sentinel)

		for {

			time.Sleep(100 * time.Millisecond)
			n, err := s.f.Read(buf)

			if n > 0 {
				output.Write(buf[:n])
				// Check whether we have seen the sentinel in the accumulated output.
				if bytes.Contains([]byte(output.String()), sentinelBytes) {
					break
				}
			}

			if err != nil {
				if errors.Is(err, syscall.EIO) {
					break
				}
				// Any other error (e.g. read deadline exceeded) — bubble up.
				done <- response{Error: fmt.Errorf("read error: %w", err)}
				return
			}
		}

		done <- response{Output: cleanOutput(output.String(), sentinel)}
	}()

	select {
	case res := <-done:
		return res.Output, res.Error
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("command timed out")
	}
}

func (s *Process) SetUser(w string) error {
	return unix.Mount(w, s.workdir, "", unix.MS_BIND, "")
}

// StopContainer suspends the container with SIGSTOP.
func (s *Process) StopContainer() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status != healper.Active {
		return fmt.Errorf("container is not running")
	}
	if err := s.cmd.Process.Signal(syscall.SIGSTOP); err != nil {
		return fmt.Errorf("SIGSTOP: %w", err)
	}
	s.status = healper.Stopped
	return nil
}

// ResumeContainer continues a previously stopped container.
func (s *Process) ResumeContainer() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status != healper.Stopped {
		return fmt.Errorf("container is already running")
	}
	if err := s.cmd.Process.Signal(syscall.SIGCONT); err != nil {
		return fmt.Errorf("SIGCONT: %w", err)
	}
	s.status = healper.Active
	return nil
}

// KillContainer forcibly kills the container process.
func (s *Process) KillContainer() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status != healper.Active {
		return fmt.Errorf("container is not running")
	}
	if err := s.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("kill: %w", err)
	}
	s.status = healper.Stopped
	return nil
}

// cleanOutput strips ANSI codes, the echoed command line, the sentinel string,
// and any residual shell prompts, then trims surrounding whitespace.
func cleanOutput(s, sentinel string) string {
	// Strip ANSI escape sequences.
	s = ansiRegx.ReplaceAllString(s, "")
	// Drop the first line (the echoed command + sentinel suffix).
	if i := strings.Index(s, "\n"); i != -1 {
		s = s[i+1:]
	}
	// Drop everything from the sentinel onward.
	if i := strings.Index(s, sentinel); i != -1 {
		s = s[:i]
	}
	// Remove leftover shell prompts.
	s = promptRegx.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
