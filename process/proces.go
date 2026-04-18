package process

import (
	"bytes"
	
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"sync"

	"log/slog"

	"os"
	"os/exec"

	"path/filepath"
	"syscall"

	"github.com/creack/pty"
)

type Process struct{
	cmd *exec.Cmd
	f *os.File
	
	running bool
	mu sync.Mutex
}
type response struct{
	Output string 
	Error error 
}

var (
	asniregx = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`) // Matches ANSI escape codes
	promptregx = regexp.MustCompile(`(?m)^[^\s]*[\$#]\s`) // Matches shell prompts (e.g., "user@host:~$ ")

	redbuf = sync.Pool{New: func() any {return make([]byte,460)}} // Buffer pool for reading command output
)

// Snadbox defines the interface for managing containerized processes
type Snadbox interface{
	CreateNewContainer() error
	Runcomand(command string)(string,error)
	StopContainer() error
	ResumeContainer() error
	KillContainer() error
	Setupnetwork(ip string) error
}


// Sandbox returns a new instance of the Process struct that implements the Snadbox interface
func Sandbox()Snadbox{
	return &Process{}
}

// convert major and minor device numbers to a single uint64 value for cgroup device access control {custom implementation}
func Mkdv(major,minor int) uint64{
	return uint64((major<<8) | minor)
}

func (s *Process) CreateNewContainer() error   {
  // Create command to run the container
	cmd := exec.Command("/proc/self/exe","child")

	
// Set up namespaces and user mappings
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS  |
			syscall.CLONE_NEWUSER,
			
		UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
        GidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
		GidMappingsEnableSetgroups: false,
	
		
	}
	
	

    // Start the container process with a pseudo-terminal
    ptx,err := pty.Start(cmd);


	if err != nil {
		slog.Error(fmt.Sprintf("error in starting container: %v", err))
		return err
	}

	// Set up cgroups for resource limits
  if err := setupCgroup(cmd.Process.Pid); err != nil {
	 cmd.Process.Kill()
	  slog.Error(fmt.Sprintf("error in setting up cgroup: %v", err))
	  return err
  }
	
  // Wait for the container process to exit in a separate goroutine
  go func() {
	if err := cmd.Wait(); err != nil {
		slog.Error(fmt.Sprintf("error container crashed: %v", err))
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		ptx.Close()
	}
   
  }()
 
  // Store the command and pseudo-terminal for later use
  s.mu.Lock()
    s.cmd = cmd
	s.f = ptx
   
	s.running = true
	s.mu.Unlock()
    return nil
}

func RunContainer() error{
	
	// Set up the container environment
 rootfs, err:= filepath.Abs("./rootfs")
 if err != nil {
	return fmt.Errorf("error in getting rootfs path: %w", err)
 }
    
   if err:=syscall.Sethostname([]byte("supersand")); err != nil {
	   return fmt.Errorf("error in setting hostname: %w", err)
   }

  if err:=  syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
	   return fmt.Errorf("error in mounting root: %w", err)
   }
   if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
	   return fmt.Errorf("error in mounting rootfs: %w", err)
   }

    if err:= syscall.Chroot(rootfs); err != nil {
        fmt.Fprintf(os.Stderr, "Chroot error: %v\n", err)
        os.Exit(1)
    }
    os.Chdir("/")
    if err:= syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
	   return fmt.Errorf("error in mounting proc: %w", err)
   }

   
	
   
    err = syscall.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ())
    return err
}

func setupCgroup(pid int) error {
	base := "/sys/fs/cgroup/ctr-" + strconv.Itoa(pid)

	if err := os.Mkdir(base, 0755); err != nil {
		return err
	}

	// 10MB RAM
	if err := os.WriteFile(base+"/memory.max", []byte("10485760"), 0644); err != nil {
		return err
	}

	// 5% CPU
	if err := os.WriteFile(base+"/cpu.max", []byte("5000 100000"), 0644); err != nil {
		return err
	}

	// max 1 processes is allowed to be created
	if err := os.WriteFile(base+"/pids.max", []byte("1"), 0644); err != nil {
		return err
	}

	return os.WriteFile(base+"/cgroup.procs", []byte(strconv.Itoa(pid)), 0644)
}

func (s *Process) Setupnetwork(ip string) error { // this is the most important part of the code as it sets up the veth pair and the network namespace for the container to allow it to communicate with the host and other containers
	// Create veth pair
	if err := exec.Command("ip", "link", "add", fmt.Sprintf("veth0-%d", s.cmd.Process.Pid), "type", "veth", "peer", "name", fmt.Sprintf("veth1-%d", s.cmd.Process.Pid)).Run(); err != nil {
		return fmt.Errorf("error in creating veth pair: %w", err)
	}
	
  if err := exec.Command("ip", "link", "set", fmt.Sprintf("veth1-%d", s.cmd.Process.Pid), "netns", strconv.Itoa(s.cmd.Process.Pid)).Run(); err != nil {
		return fmt.Errorf("error in moving veth to container namespace: %w", err)
	}

	if err := exec.Command("ip", "link", "set", fmt.Sprintf("veth0-%d", s.cmd.Process.Pid), "up").Run(); err != nil {
		return fmt.Errorf("error in setting veth0 up: %w", err)
	}

	if err := exec.Command("ip", "netns", "exec", strconv.Itoa(s.cmd.Process.Pid), "ip", "link", "set", fmt.Sprintf("veth1-%d", s.cmd.Process.Pid), "up").Run(); err != nil {
		return fmt.Errorf("error in setting veth1 up: %w", err)
	}

	if err := exec.Command("ip", "netns", "exec", strconv.Itoa(s.cmd.Process.Pid), "ip", "addr", "add", ip+"/24", "dev", fmt.Sprintf("veth1-%d", s.cmd.Process.Pid)).Run(); err != nil {
		return fmt.Errorf("error in assigning IP address to veth1: %w", err)
	}

	if err:= exec.Command("ip", "netns", "exec", strconv.Itoa(s.cmd.Process.Pid), "ip", "route", "add", "default", "via", ip).Run(); err != nil {
		return fmt.Errorf("error in setting default route in container: %w", err)
	}

	if err := exec.Command("ip", "tables", "-t", "nat", "-A", "POSTROUTING", "-s", ip+"/24", "!", "-o", fmt.Sprintf("veth0-%d", s.cmd.Process.Pid), "-j", "MASQUERADE").Run(); err != nil {
		return fmt.Errorf("error in setting up iptables for NAT: %w", err)
	}


	

	return nil
}

func (s *Process) Runcomand(command string)(string,error){ // Run a command in the container and return its output
	s.mu.Lock()
	defer s.mu.Unlock()
	
 if !s.running{
	
	return "",fmt.Errorf("server is not running")
 }

 if command == ""{
	
  return "",fmt.Errorf("comand requard")
 }
 sentinal:= fmt.Sprintf("_done_ %d", time.Now().UnixNano())
 command = command + "; echo " + sentinal + "\n"

  if _, err := s.f.Write([]byte(command)); err != nil {
	return "",fmt.Errorf("error in writing command: %w", err)
  }
 
 done := make(chan response, 1)
 

 
 
 go func () {
  
	buf:= redbuf.Get().([]byte) // Ensure the buffer is returned to the pool when done
	defer redbuf.Put(buf)
	
	 
	 var output strings.Builder
	 sentbytes := []byte(sentinal)
	 overlap:= len(sentbytes)-1
	 var tale []byte
    for {

	
	 n,err := s.f.Read(buf)
     time.Sleep(30*time.Millisecond) // Small delay to allow more output to accumulate
	 if err != nil {
		 done <- response{Error: fmt.Errorf("error in reading output: %w", err)}
		 return 
	 }
	 
	 
	 if n> 0{
	 output.Write(buf[:n]) // Append the new output to the existing output
	  full:= []byte(output.String())
	  start:= len(full)-n-overlap
	  if start < 0 {
		  start = 0
	  }
	  if bytes.Contains(full[start:], sentbytes) {
		  break
	  }
     if len(full) > overlap{
		tale = full[len(full)-overlap:]
	 }else{
		tale = full
	 }
	 _ = tale

	}
	
	}
	
	 done <- response{Output: cleanasky(output.String(),sentinal)} // Send the cleaned output back through the channel

 }()

 
 
 select {
	case res := <-done:
		return res.Output,res.Error
	case <-time.After(400*time.Millisecond): // Timeout after 400ms
		return "",fmt.Errorf("command timed out")	
 }

}

func (s *Process) StopContainer() error{  // Stop the container process
	s.mu.Lock()
	defer s.mu.Unlock()
 if !s.running{
	return fmt.Errorf("container is not running")
 }
  s.cmd.Process.Signal(syscall.SIGSTOP)
  s.running = false
  return nil
}

func (s *Process) ResumeContainer() error{  // Resume the container process if it was stopped
	s.mu.Lock()
	defer s.mu.Unlock()
	 if s.running{
		return fmt.Errorf("container is already running")
	 }
	if err := s.cmd.Process.Signal(syscall.SIGCONT); err != nil {
		return fmt.Errorf("error in resuming container: %w", err)
	}
	s.running = true
	return nil 
}

func (s *Process) KillContainer() error{  // Kill the container process
	s.mu.Lock()
	defer s.mu.Unlock()
	 if !s.running{	
		return fmt.Errorf("container is not running")
	 }
	s.cmd.Process.Kill()
	s.running = false
	return nil 
}

func cleanasky(s,sentinal string)string{ // Clean the output by removing ANSI escape codes and prompts
  s = asniregx.ReplaceAllString(s, "")
    if i := strings.Index(s, "\n"); i != -1 { s = s[i+1:] }
    if i := strings.Index(s, sentinal); i != -1 { s = s[:i] }
    s = promptregx.ReplaceAllString(s, "")
    return strings.TrimSpace(s)
}