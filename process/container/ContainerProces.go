// Package process give the interface to creat an new snadbox
package container

import (
	"sync"

	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/google/uuid"
	"github.com/suman9054/supersand/healper"
)

// ContainerProcess holds the state of a running sandboxed container.
type ContainerProcess struct {
	id              uuid.UUID
	pid             int
	status          healper.Status
	mu              sync.Mutex
	veth            string
	peername        string
	cgroupmanager   *cgroup2.Manager
	cgroupeventchan cgroup2.Event
}



// Sandbox defines the interface for managing containerized processes.
type Sandbox interface {
	CreateNewNamespaceContainer() (error, ContainerConf)
	SetupNetwork() error
	CreateEventlistener() error
}

// NewSandbox returns a new ContainerProcess that implements the Sandbox interface.
func NewSandbox() Sandbox {
	return &ContainerProcess{}
}
