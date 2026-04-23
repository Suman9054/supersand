// Package process give the interface to creat an new snadbox
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
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/suman9054/supersand/healper"
	"golang.org/x/sys/unix"
)

// Process holds the state of a running sandboxed container.
type Process struct {
	id         string
	cmd        *exec.Cmd
	f          *os.File // master PTY fd
	running    bool
	uperdir    string
	workdir    string
	meargeddir string
	mu         sync.Mutex
}

type response struct {
	Output string
	Error  error
}

var (
	ansiRegx    = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)              // strips ANSI escape codes
	promptRegx  = regexp.MustCompile(`(?m)^[^\s]*[\$#]\s`)                 // strips shell prompts
	readBufPool = sync.Pool{New: func() any { return make([]byte, 4096) }} // 4 KB read buffers
)

// Sandbox defines the interface for managing containerized processes.
type Sandbox interface {
	CreateNewContainer() error
	RunCommand(command string) (string, error)
	StopContainer() error
	ResumeContainer() error
	KillContainer() error
	SetupNetwork(ip string) error
}

// NewSandbox returns a new Process that implements the Sandbox interface.
func NewSandbox() Sandbox {
	return &Process{}
}

// ──────────────────────────────────────────────
// CreateNewContainer
// ──────────────────────────────────────────────

// CreateNewContainer spawns the container child process inside a new set of
// Linux namespaces and attaches a pseudo-terminal to it.
func (s *Process) CreateNewContainer() error {
	contanerid := healper.GenrateRandomUUid()
	workdir := fmt.Sprintf("snadinternal/v1_supersand/template/work/%s_workdir", contanerid)
	meargeddir := fmt.Sprintf("snadinternal/v1_supersand/template/merarged/%s_meargeddir", contanerid)
	uperdir := fmt.Sprintf("snadinternal/v1_supersand/template/uperdirectory/%s_uperdir", contanerid)
	lowerdir, erro := filepath.Abs("./template/base/rootfs-busy/")
	if erro != nil {
		slog.Error("err in lowe", "error", erro)
	}
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return fmt.Errorf("error in creating workingdir", "error", err)
	}
	if err := os.MkdirAll(meargeddir, 0o755); err != nil {
		return fmt.Errorf("error in creating mearg", "error", err)
	}
	if err := os.MkdirAll(uperdir, 0o755); err != nil {
		return fmt.Errorf("error in creating uperdirectory", "error", err)
	}
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerdir, uperdir, workdir)
	if err := unix.Mount("overlay", meargeddir, "overlay", 0, opts); err != nil {
		return fmt.Errorf("error in creating overlay", "error", err)
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
		return fmt.Errorf("error starting container: %w", err)
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
		unix.Unmount(s.meargeddir, syscall.MNT_DETACH)
		os.RemoveAll(s.uperdir)
		os.RemoveAll(s.workdir)
		os.RemoveAll(s.meargeddir)
		s.mu.Lock()
		s.running = false
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
	s.running = true
	s.mu.Unlock()
	return nil
}

// ──────────────────────────────────────────────
// RunContainer  (called as the "child" re-exec)
// ──────────────────────────────────────────────

// ──────────────────────────────────────────────
// cgroups
// ──────────────────────────────────────────────

func setupCgroup(pid int) error {
	base := "/sys/fs/cgroup/ctr-" + strconv.Itoa(pid)

	if err := os.Mkdir(base, 0o755); err != nil {
		return fmt.Errorf("mkdir cgroup: %w", err)
	}

	limits := []struct{ file, value string }{
		{"memory.max", "67108864"},  // 64 MB RAM
		{"cpu.max", "20000 100000"}, // 20 % CPU (20 ms per 100 ms period)
		{"pids.max", "64"},          // max 64 processes
	}
	for _, l := range limits {
		if err := os.WriteFile(base+"/"+l.file, []byte(l.value), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", l.file, err)
		}
	}

	return os.WriteFile(base+"/cgroup.procs", []byte(strconv.Itoa(pid)), 0o644)
}

// ──────────────────────────────────────────────
// Network
// ──────────────────────────────────────────────

// SetupNetwork creates a veth pair, moves one end into the container's network
// namespace, and configures IP addresses + NAT on the host side.
func (s *Process) SetupNetwork(ip string) error {
	pid := s.cmd.Process.Pid
	pidStr := strconv.Itoa(pid)
	veth0 := fmt.Sprintf("veth0-%d", pid)
	veth1 := fmt.Sprintf("veth1-%d", pid)
	const gatewayIP = "10.0.0.1"

	// Bind-mount the container's netns to a named path so `ip netns exec` works.
	if err := os.MkdirAll("/var/run/netns", 0o755); err != nil {
		return fmt.Errorf("mkdir /var/run/netns: %w", err)
	}
	netnsPath := "/var/run/netns/" + pidStr
	procNetns := fmt.Sprintf("/proc/%s/ns/net", pidStr)

	f, err := os.Create(netnsPath)
	if err != nil {
		return fmt.Errorf("create netns file: %w", err)
	}
	f.Close()

	if _, err := os.Stat(procNetns); err != nil {
		if state := s.cmd.ProcessState; state != nil {
			return fmt.Errorf("container already exited (code %d): %w", state.ExitCode(), err)
		}
		return fmt.Errorf("container not in /proc (pid %d): %w", pid, err)
	}
	nsExec := func(args ...string) error {
		out, err := exec.Command("ip", append([]string{"netns", "exec", pidStr}, args...)...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, out)
		}
		return nil
	}

	steps := []struct {
		desc string
		run  func() error
	}{
		{"create veth pair", func() error {
			return exec.Command("ip", "link", "add", veth0, "type", "veth", "peer", "name", veth1).Run()
		}},
		{"move veth1 into container netns", func() error {
			return exec.Command("ip", "link", "set", veth1, "netns", pidStr).Run()
		}},
		{"assign IP to host veth0", func() error {
			return exec.Command("ip", "addr", "add", gatewayIP+"/24", "dev", veth0).Run()
		}},
		{"bring host veth0 up", func() error {
			return exec.Command("ip", "link", "set", veth0, "up").Run()
		}},
		{"bring container veth1 up", func() error {
			return nsExec("ip", "link", "set", veth1, "up")
		}},
		{"assign IP to container veth1", func() error {
			return nsExec("ip", "addr", "add", ip+"/24", "dev", veth1)
		}},
		{"set default route in container", func() error {
			return nsExec("ip", "route", "add", "default", "via", gatewayIP)
		}},
		{"iptables NAT on host", func() error {
			return exec.Command(
				"iptables", "-t", "nat", "-A", "POSTROUTING",
				"-s", ip+"/24", "!", "-o", veth0, "-j", "MASQUERADE",
			).Run()
		}},
	}

	for _, step := range steps {
		if err := step.run(); err != nil {
			return fmt.Errorf("%s: %w", step.desc, err)
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// RunCommand
// ──────────────────────────────────────────────

// RunCommand writes a command to the container's PTY, reads back all output
// until a unique sentinel string appears, and returns the cleaned result.
func (s *Process) RunCommand(command string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
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

// ──────────────────────────────────────────────
// Lifecycle helpers
// ──────────────────────────────────────────────

// StopContainer suspends the container with SIGSTOP.
func (s *Process) StopContainer() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return fmt.Errorf("container is not running")
	}
	if err := s.cmd.Process.Signal(syscall.SIGSTOP); err != nil {
		return fmt.Errorf("SIGSTOP: %w", err)
	}
	s.running = false
	return nil
}

// ResumeContainer continues a previously stopped container.
func (s *Process) ResumeContainer() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return fmt.Errorf("container is already running")
	}
	if err := s.cmd.Process.Signal(syscall.SIGCONT); err != nil {
		return fmt.Errorf("SIGCONT: %w", err)
	}
	s.running = true
	return nil
}

// KillContainer forcibly kills the container process.
func (s *Process) KillContainer() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return fmt.Errorf("container is not running")
	}
	if err := s.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("kill: %w", err)
	}
	s.running = false
	return nil
}

// ──────────────────────────────────────────────
// Output cleaning
// ──────────────────────────────────────────────

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
