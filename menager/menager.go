package menager

import (
	"fmt"
	"log/slog"

	"github.com/suman9054/supersand/store"
)


func Menager(){
  num:=store.NewTasks().Lenth()

  for num !=0 {
    data,err :=store.NewTasks().Dqueue()
	if err !=nil{
		slog.Info("one err happen",fmt.Errorf(err.Error()))
	}
	switch data.Tasktype{
	case 1:
		
	}
  }
}

func processrunner(){

}