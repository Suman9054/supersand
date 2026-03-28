package store

type Store struct {
	Chash    stable[string, userdata]
	Querys   queys[Prioritytaskvalue]
	Tasks    queys[Unprioritytasks]
}

func Newstore() *Store {
	return &Store{
		Chash:    Newstoremap(),
		Querys:   NewprorityTasks(),
		Tasks:  Newunproritytsks(),
	}
}
