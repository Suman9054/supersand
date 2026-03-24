package menager

import (
	"fmt"
	

	"github.com/suman9054/supersand/store"
)


func Menager(v chan store.Prioritytaskvalue,s *store.Store){
  for {
	tasks,err:=s.Querys.Dqueue()
	if err != nil {
		continue
	}
	v<-tasks
  }
}


func Worker(v chan store.Prioritytaskvalue){
  for tasks := range v{
     fmt.Print("procassing",tasks.User)
	 fmt.Print("process ",tasks.Process)
  }
}