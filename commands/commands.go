package comands

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/suman9054/supersand/healper"
	"github.com/suman9054/supersand/store"
)

/* i have to creat init,status,config,pull,run,stop and delete commnad. */

type Commnadstype interface{
	Init() (string,error)
  Status()(store.Servicedata,error)
  Pull(path string) error
  Run() error
	Stop() error
	Delete() error
}

type comands struct{
	s					*store.Store
	userkey		uint64
	serviceid	uuid.UUID
}

func Newcommands(s *store.Store,userkey uint64,serviceid uuid.UUID) Commnadstype{
	return &comands{
   s: s,
	 userkey: userkey,
	 serviceid: serviceid,
	}
} 

func (c *comands) Init()(string,error){
	 id:=uuid.New()

	if err,_:= c.s.Chash.Update(c.userkey, func(v store.UserObject) store.UserObject{
		v.Survices = append(v.Survices,store.Servicedata{
												Id: 						id,
												Lastacces: 			time.Now(),
												Processtatus:		healper.Pending,
												ServiceUptime:  time.Now().Sub(time.Now()),
                        Ramusage: 0,
												})
		return v
	}); err != nil{

   
		return "",fmt.Errorf("error in service init :%v",err)
	}

	return id.String(),nil
} 

func (c *comands) Status()(store.Servicedata,error){

	return
}

func(c *comands) Pull(path string)error{
	
	return nil
}

func(c *comands) Run() error{

return nil
}

func(c *comands) Stop() error {

	return nil
}

func(c *comands) Delete() error {

	return nil
}
