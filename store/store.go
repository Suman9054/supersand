package store

import (
	"time"

	"github.com/suman9054/supersand/process"

	"github.com/google/uuid"

	"github.com/suman9054/supersand/healper"
)

type Servicedata struct { // implimenting lfu based cash updation the apis for lfu
	Id            uuid.UUID
	Lastacces     time.Time
	Processtatus  healper.Status
	ServiceUptime time.Duration
	Ramusage      int8
	WorkingDir    string
}

type UserObject struct {
	Id       uuid.UUID
	Survices []Servicedata
}

type Processdata struct {
	PID           int
	ProcessStatus healper.Status
}

type Store struct {
	Chash       stable[uint64, UserObject]
	Querys      queys[Prioritytaskvalue]
	Tasks       queys[Unprioritytasks]
	ProcessPool queys[process.Sandbox]
	ProcessMap  stable[string, Processdata]
}

func Newstore() *Store {
	return &Store{
		Chash:       NewChach(1024),
		Querys:      NewprorityTasks(),
		Tasks:       Newunproritytsks(),
		ProcessPool: NewProcessPool(),
	}
}
