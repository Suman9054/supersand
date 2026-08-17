package store

import "github.com/suman9054/supersand/process"

type Store struct {
	Chash       stable[uint64, UserObject]
	Querys      queys[Prioritytaskvalue]
	Tasks       queys[Unprioritytasks]
	ProcessPool queys[process.Sandbox]
	ProcessMap  stable[string, Processdata]
}

func Newstore() *Store {
	return &Store{
		Chash:       NewUserCash(),
		Querys:      NewprorityTasks(),
		Tasks:       Newunproritytsks(),
		ProcessPool: NewProcessPool(),
		ProcessMap:  Newprocesscash(),
	}
}
