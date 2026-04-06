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
      if s.Tasks.Isempty(){
        time.Sleep(100*time.Millisecond)
        continue
      }
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
   time.Sleep(10*time.Millisecond)
 }
}

func Worker(v chan Processchannel,s *store.Store){
  for tasks := range v{
    switch tasks.Tasktype{
    case store.Startnewsesion:
      slog.Info("starting new sesion for user ",tasks.Prioritytaskvalue.Sesioninfo.User)
      p:=process.Sandbox()
      err:=p.CreateNewContainer()
       s.Chash.Set(tasks.Prioritytaskvalue.User,store.Userdata{
         Id: tasks.Prioritytaskvalue.User,
          Useuniqename: tasks.Prioritytaskvalue.User, 
          Lastacces:  time.Now(),
          Processstatus: store.Active,
          Process:p,
      })
      if err!=nil {
        slog.Error("error in creating container",err)
        tasks.Prioritytaskvalue.Respons<- store.Responschannel{
          Msg: fmt.Errorf("one err happen ",err.Error()),
          Status: 500,
        }
        continue
      }
      
      tasks.Prioritytaskvalue.Respons<- store.Responschannel{
        Msg: "sesion started",
        Status: 200,
      }
    case store.Stopesesion:
       slog.Info("stoping te contaner of user",tasks.Prioritytaskvalue.Sesioninfo.User)
        user,ok:=s.Chash.Get(tasks.Prioritytaskvalue.Sesioninfo.User)
        if !ok{
          slog.Error("not an valid user",tasks.Prioritytaskvalue.Sesioninfo.User)
          tasks.Prioritytaskvalue.Respons<- store.Responschannel{
            Msg: "not an valid user",
            Status: 500,
          }
          continue
        }
       err:=user.Process.StopContainer()
       if err != nil{
        slog.Error("err in stoping contaner for user%s err:",tasks.Prioritytaskvalue.User,fmt.Errorf(err.Error()))
        tasks.Prioritytaskvalue.Respons<- store.Responschannel{
          Msg: "err on stoping contaner",
          Status: 500,
        }
        continue
       }
        s.Chash.Set(tasks.Prioritytaskvalue.User,store.Userdata{
        Processstatus: store.Stopped,
       })
       if err != nil{
        slog.Error("err in stoping contaner for user%s err:",tasks.Prioritytaskvalue.User,fmt.Errorf(err.Error()))
        tasks.Prioritytaskvalue.Respons<- store.Responschannel{
          Msg: "err on stoping contaner",
          Status: 500,
        }
       }
       tasks.Prioritytaskvalue.Respons <- store.Responschannel{
        Msg: "sucsess",
        Status: 200,
       }
       continue
    case store.Runcomand:
          user,ok:=s.Chash.Get(tasks.Unprioritytasks.Sesioninfo.User)
          if !ok{
            slog.Error("not an valid user",tasks.Unprioritytasks.Sesioninfo.User)
          }
        if user.Processstatus != store.Active{
          slog.Error("process is not active",user.Useuniqename)
          s.Tasks.Enqueue(store.Unprioritytasks{
            Comand: tasks.Unprioritytasks.Comand,
            Respons: tasks.Unprioritytasks.Respons,
            Sesioninfo:store.Sesioninfo{
              User: tasks.Unprioritytasks.Sesioninfo.User,
            },
          })
        }
       data,err:=user.Process.Runcomand(tasks.Unprioritytasks.Comand)
       if err != nil{
        slog.Error("err in runing comand for user%s err:",tasks.Unprioritytasks.Sesioninfo.User,fmt.Errorf(err.Error()))
        tasks.Unprioritytasks.Respons<- store.Responschannel{
          Msg: "err on runing comand",
          Status: 500,
        }
        continue
       }
       tasks.Unprioritytasks.Respons<- store.Responschannel{
        Msg: data,
        Status: 200,
       }
       continue 
    }
    
  }
}