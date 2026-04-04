package menager

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/suman9054/supersand/process"
	"github.com/suman9054/supersand/store"
)

type Processchannel struct{
  store.Prioritytaskvalue
  store.Unprioritytasks
}

func Menager(v chan Processchannel,s *store.Store){
  for {
	  if s.Querys.Isempty() {
      value,err:=s.Tasks.Dqueue()
     if err!=nil{
      slog.Error("err in unpririty task",fmt.Errorf(err.Error()))
       continue
      }   
     v<- Processchannel{
       Unprioritytasks: value,
     }
    }
   ver,err:=s.Querys.Dqueue()
   if err!=nil{
    slog.Error("err in pririty task",fmt.Errorf(err.Error()))
     continue
    }   
   v<- Processchannel{
     Prioritytaskvalue: ver,
   } 
 }
}

func Worker(v chan Processchannel,s *store.Store){
  for tasks := range v{
    switch tasks.Tasktype{
    case store.Startnewsesion:
      slog.Info("starting new sesion for user ",tasks.Prioritytaskvalue.User)
      p:=process.Sandbox()
      err:=p.CreateNewContainer()
      ok:= s.Chash.Set(tasks.Prioritytaskvalue.User,store.Userdata{
         Id: tasks.Prioritytaskvalue.User,
          Useuniqename: tasks.Prioritytaskvalue.User, 
          Lastacces:  time.Now(),
          Processstatus: store.Active,
          Process: p,
      })
      if err!=nil{
        slog.Error("error in creating container",err)
        tasks.Prioritytaskvalue.Respons<- store.Responschannel{
          Msg: fmt.Errorf("one err happen ",err.Error()),
          Status: 500,
        }
        continue
      }
    }
    
  }
}