package menager

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/dgryski/go-farm"
	"github.com/suman9054/supersand/healper"
	"github.com/suman9054/supersand/process"
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
			if v.Processstatus != store.Active {
				continue
			}

			if now.Sub(v.Lastacces) <= 5*time.Minute {
				continue
			}

			slog.Info("killing container", "user", v.Useuniqename)

			if err := v.Process.StopContainer(); err != nil {
				slog.Error("failed to kill container",
					"user", v.Useuniqename,
					"err", err,
				)
				continue
			}
			key := farm.Fingerprint64([]byte(v.Useuniqename))
			keystr := fmt.Sprintf("%d", key)
			_, ok := s.Chash.Update(keystr, func(u store.Userdata) store.Userdata {
				u.Processstatus = store.Stopped
				return u
			})

			if !ok {
				slog.Error("failed to update user state", "user", v.Useuniqename)
			}
		}
	}
}

func Worker(v chan Processchannel, s *store.Store) { // Worker is a goroutine that processes incoming tasks from the channel and interacts with the store to manage user sessions and execute commands in containers
	for tasks := range v {
		switch tasks.Tasks {
		case store.Startnewsesion:
			slog.Info(fmt.Sprintf("starting new sesion for user %s", tasks.Prioritytaskvalue.User))
			p := process.NewSandbox()
			err := p.CreateNewContainer()
			if err != nil {
				slog.Error("error in creating container", err)

				tasks.Prioritytaskvalue.Respons <- store.Responschannel{
					Msg:    fmt.Errorf("one err happen ", err),
					Status: 500,
				}
				continue
			}
			key := farm.Fingerprint64([]byte(tasks.Prioritytaskvalue.User))
			keystr := fmt.Sprintf("%d", key)
			id := healper.GenrateRandomUUid()
			s.Chash.Set(keystr, store.Userdata{
				Id:            id,
				Useuniqename:  tasks.Prioritytaskvalue.User,
				Lastacces:     time.Now(),
				Processstatus: store.Active,
				Process:       p,
			})

			tasks.Prioritytaskvalue.Respons <- store.Responschannel{
				Msg:    "sesion started",
				Id:     id,
				Status: 200,
			}

		case store.Runcomand:
			key := farm.Fingerprint64([]byte(tasks.Unprioritytasks.Sesioninfo.User))
			keystr := fmt.Sprintf("%d", key)
			user, ok := s.Chash.Get(keystr)
			if !ok {
				slog.Error("not an valid user", tasks.Unprioritytasks.Sesioninfo.User)

				tasks.Unprioritytasks.Respons <- store.Responschannel{
					Msg:    "not an valid user",
					Status: 500,
				}
				continue
			}
			if user.Processstatus != store.Active {
				slog.Error(fmt.Sprintf("process is not active for user %s", user.Useuniqename))

				if err := user.Process.ResumeContainer(); err != nil {
					slog.Error(fmt.Sprintf("err in resuming container for user%s err:", user.Useuniqename, err))

					tasks.Unprioritytasks.Respons <- store.Responschannel{
						Msg:    "err in resuming container",
						Status: 500,
					}
					continue
				}
				err, ok := s.Chash.Update(keystr, func(u store.Userdata) store.Userdata {
					u.Lastacces = time.Now()
					u.Processstatus = store.Active
					return u
				})
				if err != nil || !ok {
					slog.Error(fmt.Sprintf("err in updating user status for user%s err:", user.Useuniqename, err))

					tasks.Unprioritytasks.Respons <- store.Responschannel{
						Msg:    "err in updating user status",
						Status: 500,
					}
					if err := user.Process.StopContainer(); err != nil {
						slog.Error(fmt.Sprintf("err in stopping container for user%s err:", user.Useuniqename, err))
					}
					continue
				}

			}

			data, err := user.Process.RunCommand(tasks.Unprioritytasks.Comand)
			if err != nil {
				slog.Error(fmt.Sprintf("err in runing comand for user %s err:", tasks.Unprioritytasks.Sesioninfo.User, err))

				tasks.Unprioritytasks.Respons <- store.Responschannel{
					Msg:    "err on runing comand",
					Status: 500,
				}
				continue
			}

			tasks.Unprioritytasks.Respons <- store.Responschannel{
				Msg:    data,
				Status: 200,
			}
			continue
		}
	}
}

