// Package process give the interface to creat an new snadbox
package process

import (
	"os"
	"os/exec"
	"sync"

	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/google/uuid"
	"github.com/suman9054/supersand/healper"
)

// Process holds the state of a running sandboxed container.
type Process struct {
	id              uuid.UUID
	pid             int
	cmd             *exec.Cmd
	f               *os.File // master PTY fd
	status          healper.Status
	mu              sync.Mutex
	veth            string
	peername        string
	cgroupmanager   *cgroup2.Manager
	cgroupeventchan cgroup2.Event
}

type response struct {
	Output string
	Error  error
}

// Sandbox defines the interface for managing containerized processes.
type Sandbox interface {
	CreateNewContainer() (error, ContanerConf)
	RunCommand(command string) (string, error)
	StopContainer() error
	ResumeContainer() error
	KillContainer() error
	SetupNetwork() error
	CreatEventlistner() error
}

// NewSandbox returns a new Process that implements the Sandbox interface.
func NewSandbox() Sandbox {
	return &Process{}
}
