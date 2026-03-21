package process

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
)

type process struct{
	cmd *exec.Cmd
	stdin io.WriteCloser
	stdout io.ReadCloser
	running bool
}

type snadbox interface{
	CreateNewContainer() error
	RunContainer() error
	Runcomand(command string)(string,error)
}


func Sandbox()snadbox{
	return &process{}
}

func (s *process) CreateNewContainer() error   {

	s.cmd = exec.Command("/proc/self/exe")

	

	s.cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS  |
			syscall.CLONE_NEWUSER,
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

func (s *process)RunContainer() error{
 fmt.Println("inside container")
  var err error
	err=syscall.Sethostname([]byte("supersand"))

	if err != nil {
	slog.Error("eror in sethost",err.Error())
	}

	rootfs := "./rootfs"

	
	if err = syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err !=nil{
		slog.Error("err: ",err)
	}

	
	if err = syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		slog.Error("err:",err)
	}

	// chroot
	if err = syscall.Chroot(rootfs); err != nil {
		slog.Error("err",err)
	}
	if err = os.Chdir("/"); err !=nil {
		slog.Error("err",err)
	}

	// mount proc
	if err = syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		slog.Error("err",err)
	}

	// run command inside container
	s.cmd = exec.Command("/bin/sh")
    stdin,_:=s.cmd.StdinPipe()
	stdout,_:=s.cmd.StdoutPipe()
	s.stdin = stdin
	s.stdout = stdout
	err =s.cmd.Run()
	s.running = true
	if err !=nil{
		return err
	}
	return nil
}

func (s *process) Runcomand(command string)(string,error){
	
 if !s.running{
	
	return "",fmt.Errorf("server is not running")
 }

 if command == ""{
  return "",fmt.Errorf("comand requard")
 }

 _,err:=s.stdin.Write([]byte(command + "\n"))
 if err != nil {
	return "", err
 }

 buffer := make([]byte,4096)
 n,errs := s.stdout.Read(buffer)

 if errs !=nil {
	return "",errs
 }

 return string(buffer[:n]),nil

}
