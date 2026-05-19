package menager

import (
	"log/slog"
	"time"

	"github.com/suman9054/supersand/store"
)

type Processchannel struct { // Processchannel is a struct that represents a message sent to the worker goroutine for processing tasks
	store.Prioritytaskvalue
	store.Unprioritytasks
	store.Tasks
}

func Manager(v chan Processchannel, s *store.Store) { // Manager is the main loop that listens for incoming tasks and dispatches them to worker goroutines
	for {
		store.Sheardcond.L.Lock()

		// Wait only when both queues are empty
		for s.Querys.Isempty() && s.Tasks.Isempty() {
			store.Sheardcond.Wait()
		}

		var msg Processchannel

		if !s.Querys.Isempty() {
			ver, err := s.Querys.Dqueue()
			if err != nil {
				slog.Error("err in priority task", "error", err)
				store.Sheardcond.L.Unlock()
				continue
			}
			msg = Processchannel{
				Tasks:             ver.Tasktype,
				Prioritytaskvalue: ver,
			}
		} else {
			value, err := s.Tasks.Dqueue()
			if err != nil {
				slog.Error("err in unpriority task", "error", err)
				store.Sheardcond.L.Unlock()
				continue
			}
			msg = Processchannel{
				Tasks:           value.Tasktype,
				Unprioritytasks: value,
			}
		}

		store.Sheardcond.L.Unlock() // Release BEFORE blocking on channel send

		v <- msg
	}
}

func Killer(s *store.Store) { // it use a ticker to periodically check for inactive containers and kill them if they have been inactive for more than 5 minutes
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		for _, v := range s.Chash.Allitems() {
		}
	}
}

func worker(v chan Processchannel, s *store.Store) {
}
