package process

import (
	"bytes"
	"fmt"
	"regexp"
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
	asniregx = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	promptregx = regexp.MustCompile(`(?m)^[^\s]*[\$#]\s`)

	redbuf = sync.Pool{New: func() any {return make([]byte,460)}}
)

type Snadbox interface{
	CreateNewContainer() error
	Runcomand(command string)(string,error)
	StopContainer() error
	ResumeContainer() error
	KillContainer() error
}


func Sandbox()Snadbox{
	return &Process{}
}

func Mkdv(major,minor int) uint64{
	return uint64((major<<8) | minor)
}

func (s *Process) CreateNewContainer() error   {
  s.mu.Lock()
  defer s.mu.Unlock()
	cmd := exec.Command("/proc/self/exe","child")

	

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
	
	

    
    ptx,err := pty.Start(cmd);

	if err != nil {
		slog.Error("error in starting container",err)
		return err
	}
	
  go func() {
	if err := cmd.Wait(); err != nil {
		slog.Error("error contaner crashed", err)
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		ptx.Close()
	}
   
  }()
 
 
    s.cmd = cmd
	s.f = ptx
   
	s.running = true
	
    return nil
}

func RunContainer() error{
	
	
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
        fmt.Fprintf(os.Stderr, "Chroot error: %w\n", err)
        os.Exit(1)
    }
    os.Chdir("/")
    if err:= syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
	   return fmt.Errorf("error in mounting proc: %w", err)
   }

   
	
   
    err = syscall.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ())
    return err
}




func (s *Process) Runcomand(command string)(string,error){
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
  
	buf:= redbuf.Get().([]byte)
	defer redbuf.Put(buf)
	
	 
	 var output strings.Builder
	 sentbytes := []byte(sentinal)
	 overlap:= len(sentbytes)-1
	 var tale []byte
    for {

	
	 n,err := s.f.Read(buf)
     time.Sleep(30*time.Millisecond)
	 if err != nil {
		 done <- response{Error: fmt.Errorf("error in reading output: %w", err)}
		 return 
	 }
	 
	 
	 if n> 0{
	 output.Write(buf[:n])
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
	 done <- response{Output: cleanasky(output.String(),sentinal)}

 }()

 
 
 select {
	case res := <-done:
		return res.Output,res.Error
	case <-time.After(400*time.Millisecond):
		return "",fmt.Errorf("command timed out")	
 }

}

func (s *Process) StopContainer() error{
	s.mu.Lock()
	defer s.mu.Unlock()
 if !s.running{
	return fmt.Errorf("container is not running")
 }
  s.cmd.Process.Signal(syscall.SIGSTOP)
  s.running = false
  return nil
}

func (s *Process) ResumeContainer() error{
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

func (s *Process) KillContainer() error{
	s.mu.Lock()
	defer s.mu.Unlock()
	 if !s.running{	
		return fmt.Errorf("container is not running")
	 }
	s.cmd.Process.Kill()
	s.running = false
	return nil 
}

func cleanasky(s,sentinal string)string{
  s = asniregx.ReplaceAllString(s, "")
    if i := strings.Index(s, "\n"); i != -1 { s = s[i+1:] }
    if i := strings.Index(s, sentinal); i != -1 { s = s[:i] }
    s = promptregx.ReplaceAllString(s, "")
    return strings.TrimSpace(s)
}