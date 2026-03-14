package store

type Store struct {
	chash    stable[string, userdata]
	querys   queys
	faildque Stack
}

func Newstore() *Store {
	return &Store{
		chash:    Newstoremap(),
		querys:   NewTasks(),
		faildque: Newstack(),
	}
}
