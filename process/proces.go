package process

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
)

type Process struct{
	cmd *exec.Cmd
	stdin io.WriteCloser
	stdout io.ReadCloser
	running bool
}

type Snadbox interface{
	CreateNewContainer() error
	RunContainer() error
	Runcomand(command string)(string,error)
	StopContainer() error
	KillContainer() error
}


func Sandbox()Snadbox{
	return &Process{}
}

func Mkdv(major,minor int) uint64{
	return uint64((major<<8) | minor)
}

func (s *Process) CreateNewContainer() error   {

	s.cmd = exec.Command("/proc/self/exe")

	

	s.cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS  ,
			
		UidMappings: []syscall.SysProcIDMap{
			{
              ContainerID: 0,
			  HostID: 100000,
			  Size: 65536,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID: 100000,
				Size: 1,
			},
		},
		
		GidMappingsEnableSetgroups: false,

	}

	return s.RunContainer()
}

func (s *Process)RunContainer() error{
 fmt.Println("inside container")
  var err error
	err=syscall.Sethostname([]byte("supersand"))

	if err != nil {
	slog.Error("eror in sethost",err.Error())
	}

	rootfs := "./rootfs"

	
	if err = syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err !=nil{
		slog.Error("err: ",err.Error())
	}

	if err := os.MkdirAll(rootfs, 0755); err != nil {
    slog.Error("mkdir error:", err.Error())
    
   }
	if err = syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		slog.Error("err on mounting rootfs",err.Error())
	}

	
	if err = syscall.Chroot(rootfs); err != nil {
		slog.Error("err",err.Error())
	}
	if err = os.Chdir("/"); err !=nil {
		slog.Error("err",err.Error())
	}
  
	

	
	s.cmd = exec.Command("/bin/sh")
    stdin,_:=s.cmd.StdinPipe()
	stdout,_:=s.cmd.StdoutPipe()
	s.stdin = stdin
	s.stdout = stdout
	err =s.cmd.Start()
	s.running = true
	if err !=nil{
		return err
	}
	return nil
}

func (s *Process) Runcomand(command string)(string,error){
	
 if !s.running{
	
	return "",fmt.Errorf("server is not running")
 }

 if command == ""{
  return "",fmt.Errorf("comand requard")
 }

 command = fmt.Sprintf("%s & echo PID:$!",command)

 _,err:=s.stdin.Write([]byte(command + "\n"))
 if err != nil {
	return "", err
 }

 buffer := make([]byte,4096)
 n,errs := s.stdout.Read(buffer)

 if errs !=nil {
	return "",errs
 }
fmt.Println("comand output:",string(buffer[:n]))
 return string(buffer[:n]),nil

}

func (s *Process) StopContainer() error{
 if !s.running{
	return fmt.Errorf("container is not running")
 }
  s.cmd.Process.Signal(syscall.SIGTERM)
  s.running = false
  return nil
}

func (s *Process) KillContainer() error{
	 if !s.running{	
		return fmt.Errorf("container is not running")
	 }
	s.cmd.Process.Kill()
	s.running = false
	return nil 
}