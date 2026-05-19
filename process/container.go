package process

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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
// Linux namespaces and attaches a pseudo-terminal to it.
func (s *Process) CreateNewContainer() (error, ContanerConf) {
	contanerid := healper.GenrateRandomUUid()
	var v ContanerConf

	workdir := fmt.Sprintf("sandinternal/v1_supersand/template/work/%s_workdir", contanerid)
	meargeddir := fmt.Sprintf("sandinternal/v1_supersand/template/merarged/%s_meargeddir", contanerid)
	uperdir := fmt.Sprintf("sandinternal/v1_supersand/template/uperdirectory/%s_uperdir", contanerid)
	lowerdir, erro := filepath.Abs("./template/base/rootfs-busy/")
	if erro != nil {
		slog.Error("err in lowe", "error", erro)
	}
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return fmt.Errorf("error in creating workingdir %w", err), v
	}
	if err := os.MkdirAll(meargeddir, 0o755); err != nil {
		return fmt.Errorf("error in creating mearg %w", err), v
	}
	if err := os.MkdirAll(uperdir, 0o755); err != nil {
		return fmt.Errorf("error in creating uperdirectory %w", err), v
	}
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerdir, uperdir, workdir)
	if err := unix.Mount("overlay", meargeddir, "overlay", 0, opts); err != nil {
		return fmt.Errorf("error in creating overlay %w", err), v
	}

	cmd := exec.Command("/proc/self/exe", "child", meargeddir)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
	}

	ptx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("error starting container: %w", err), v
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
		unix.Unmount(meargeddir, syscall.MNT_DETACH)
		os.RemoveAll(uperdir)
		os.RemoveAll(workdir)
		os.RemoveAll(meargeddir)
		s.mu.Lock()
		s.status = healper.Stopped
		_ = ptx.Close()
		s.mu.Unlock()
	}()

	s.mu.Lock()
	s.cmd = cmd
	s.f = ptx
	s.uperdir = uperdir
	s.workdir = workdir
	s.meargeddir = meargeddir
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
