package menager

import (
	"fmt"
	

	"github.com/suman9054/supersand/store"
)


func Menager(v chan store.Prioritytaskvalue,s *store.Store){
  for {
	
 }
}


func Worker(v chan store.Prioritytaskvalue){
  for tasks := range v{
     fmt.Print("procassing",tasks.User)
	 fmt.Print("process ",tasks.Process)
  }
}