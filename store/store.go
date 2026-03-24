package store

type Store struct {
	Chash    stable[string, userdata]
	Querys   queys
	Faildque Stack
}

func Newstore() *Store {
	return &Store{
		Chash:    Newstoremap(),
		Querys:   NewprorityTasks(),
		Faildque: Newstack(),
	}
}
