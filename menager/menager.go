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
  store.Tasks
}

func Manager(v chan Processchannel, s *store.Store) {
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
                Tasks:              ver.Tasktype,
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

func Killer(s *store.Store){
  for {
    time.Sleep(100*time.Millisecond)
    for _,v:= range s.Chash.Allitems(){
      if v.Processstatus == store.Active && time.Since(v.Lastacces)>5*time.Minute{
        slog.Info(fmt.Sprintf("killing the contaner of user %s",v.Useuniqename))
        err:= v.Process.StopContainer()
        if err != nil{
          slog.Error("err in killing contaner for user%s err:",v.Useuniqename,err)
          continue
         }
         err,ok:= s.Chash.Update(v.Id,func(u store.Userdata) store.Userdata {
          u.Processstatus = store.Stopped
          return u
         })
            if err != nil || !ok{ 
            slog.Error(fmt.Sprintf("err in updating user status for user%s err:",v.Useuniqename, err))
            continue
           }
      }
    }
  }
}

func Worker(v chan Processchannel,s *store.Store){
  for tasks := range v{
    

    

    switch tasks.Tasks{
    case store.Startnewsesion:
      slog.Info(fmt.Sprintf("starting new sesion for user %s",tasks.Prioritytaskvalue.User))
      p:=process.Sandbox()
      err:=p.CreateNewContainer()
      
      
      if err!=nil {
        slog.Error("error in creating container",err)
        
        tasks.Prioritytaskvalue.Respons<- store.Responschannel{
          Msg: fmt.Errorf("one err happen ",err),
          Status: 500,
        }
        continue
      }
       s.Chash.Set(tasks.Prioritytaskvalue.User,store.Userdata{
         Id: tasks.Prioritytaskvalue.User,
          Useuniqename: tasks.Prioritytaskvalue.User, 
          Lastacces:  time.Now(),
          Processstatus: store.Active,
          Process:p,
      })
      
      tasks.Prioritytaskvalue.Respons<- store.Responschannel{
        Msg: "sesion started",
        Status: 200,
      }
    
       
    case store.Runcomand:
          user,ok:=s.Chash.Get(tasks.Unprioritytasks.Sesioninfo.User)
          if !ok{
            slog.Error("not an valid user",tasks.Unprioritytasks.Sesioninfo.User)
             
            tasks.Unprioritytasks.Respons<- store.Responschannel{
              Msg: "not an valid user",
              Status: 500,
            }
            continue
          }
        if user.Processstatus != store.Active{
          slog.Error(fmt.Sprintf("process is not active for user %s", user.Useuniqename))
        
          
         if err:= user.Process.ResumeContainer(); err != nil{
          slog.Error(fmt.Sprintf("err in resuming container for user%s err:",user.Useuniqename,err))
          
          tasks.Unprioritytasks.Respons<- store.Responschannel{
            Msg: "err in resuming container",
            Status: 500,
          }
          continue
         }
        err,ok:= s.Chash.Update(user.Id,func(u store.Userdata) store.Userdata {
          u.Lastacces = time.Now()
          u.Processstatus = store.Active
          return u
        })
           if err != nil || !ok{
            slog.Error(fmt.Sprintf("err in updating user status for user%s err:",user.Useuniqename, err))
            
            tasks.Unprioritytasks.Respons<- store.Responschannel{
              Msg: "err in updating user status",
              Status: 500,
            }
           if  err:=user.Process.StopContainer(); err != nil{
            slog.Error(fmt.Sprintf("err in stopping container for user%s err:",user.Useuniqename,err))
            }
            continue
           }
         
        }
        
        
       data,err:=user.Process.Runcomand(tasks.Unprioritytasks.Comand)
       if err != nil{
        slog.Error(fmt.Sprintf("err in runing comand for user %s err:",tasks.Unprioritytasks.Sesioninfo.User, err))
       
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