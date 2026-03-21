package process

import (
	"os"
	"os/exec"
	"syscall"
)

func Creatnewcontaner() *exec.Cmd {
	cmd := exec.Command("bin/bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.Clone_NEWUTS,
	}
	return cmd
}
