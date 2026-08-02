package comands

import (
	"fmt"

	"github.com/suman9054/supersand/healper"
	"github.com/suman9054/supersand/store"
)

type Statusres struct {
	Serviceuptime  int32
	Resourcesusage int32
}

func Statuse(sesionID string, s *store.Store) (Statusres, error) {
	var def Statusres
	val, ok := s.Chash.Get(sesionID)
	if !ok {
		return def, fmt.Errorf("sesionId:%s dose not exiest", sesionID)
	}
	return Statusres{
		Serviceuptime:  val.ServiceUptime,
		Resourcesusage: val.Rameusage,
	}, nil
}

func Init(s *store.Store) string {
	id := healper.GenrateRandomUUid()
	work := fmt.Sprintf("sandinternal/services/%s_work", id)
	s.Chash.Set(id, store.Servicedata{
		Id:            id,
		Processstatus: healper.Pending,
		ServiceUptime: 0,
		Rameusage:     0,
		WorkingDir:    work,
	})

	return id
}
