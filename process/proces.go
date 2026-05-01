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
	"github.com/vishvananda/netlink"
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
	veth       string
	peername   string
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
	SetupNetwork() error
}

// NewSandbox returns a new Process that implements the Sandbox interface.
func NewSandbox() Sandbox {
	return &Process{}
}

// CreateNewContainer spawns the container child process inside a new set of
// Linux namespaces and attaches a pseudo-terminal to it.
func (s *Process) CreateNewContainer() error {
	contanerid := healper.GenrateRandomUUid()
	workdir := fmt.Sprintf("sandinternal/v1_supersand/template/work/%s_workdir", contanerid)
	meargeddir := fmt.Sprintf("sandinternal/v1_supersand/template/merarged/%s_meargeddir", contanerid)
	uperdir := fmt.Sprintf("sandinternal/v1_supersand/template/uperdirectory/%s_uperdir", contanerid)
	lowerdir, erro := filepath.Abs("./template/base/rootfs-busy/")
	if erro != nil {
		slog.Error("err in lowe", "error", erro)
	}
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return fmt.Errorf("error in creating workingdir %w", err)
	}
	if err := os.MkdirAll(meargeddir, 0o755); err != nil {
		return fmt.Errorf("error in creating mearg %w", err)
	}
	if err := os.MkdirAll(uperdir, 0o755); err != nil {
		return fmt.Errorf("error in creating uperdirectory %w", err)
	}
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerdir, uperdir, workdir)
	if err := unix.Mount("overlay", meargeddir, "overlay", 0, opts); err != nil {
		return fmt.Errorf("error in creating overlay %w", err)
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
		unix.Unmount(meargeddir, syscall.MNT_DETACH)
		os.RemoveAll(uperdir)
		os.RemoveAll(workdir)
		os.RemoveAll(meargeddir)
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

// SetupNetwork creates a veth pair, moves one end into the container's network
// namespace, and configures IP addresses + NAT on the host side.
func (s *Process) SetupNetwork() error {
	pid := s.cmd.Process.Pid
	id := healper.GenrateNetworkid()
	host := fmt.Sprintf("veth-host-%s", id)
	peername := fmt.Sprintf("eth-%s", id)

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: host,
		},
		PeerName: peername,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("error in veth setup %w", err)
	}

	link, erro := netlink.LinkByName(peername)

	if erro != nil {
		return fmt.Errorf("error in conect veth by name %w", erro)
	}

	if err := netlink.LinkSetNsPid(link, pid); err != nil {
		return fmt.Errorf("eror in namespace veth setup for %s error is %w ", id, err)
	}
	hostl, errh := netlink.LinkByName(host)
	if errh != nil {
		return fmt.Errorf("error in seting up vet on host %w", errh)
	}

	netlink.LinkSetUp(hostl)
	adder, erroradder := netlink.ParseAddr("10.0.0.1/24")
	if erroradder != nil {
		return fmt.Errorf("eeror in netlink setup %w", erroradder)
	}
	netlink.AddrAdd(hostl, adder)
	s.mu.Lock()
	s.veth = host
	s.peername = peername
	s.mu.Unlock()
	return nil
}

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
