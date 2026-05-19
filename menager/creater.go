package menager

import (
	"log/slog"
	"sync"

	"github.com/suman9054/supersand/healper"
	"github.com/suman9054/supersand/process"
	"github.com/suman9054/supersand/store"
)

func SetupContaners(s *store.Store) {
	total := healper.DecideLimits()
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for i := 1; i <= total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			p := process.NewSandbox()
			err, v := p.CreateNewContainer()
			if err != nil {
				slog.Error("Failed to create an contaner", "error", err)
				<-sem
				return
			}
			err1 := p.SetupNetwork()
			if err1 != nil {
				<-sem
				slog.Error("failed to setup network of an contaner ", "error", err1)
				if err2 := p.KillContainer(); err2 != nil {
					slog.Error("failed tp kill contaner", "error", err2)
					return
				}
				return
			}
			s.ProcessPool.Enqueue(p)
			s.ProcessMap.Set(v.Id, store.Processdata{
				PID:           v.PID,
				ProcessStatus: v.Status,
			})
			<-sem
		}()
	}
}
