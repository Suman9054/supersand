package main

import (
	"context"
	

	"encoding/json"

	"log"
	"log/slog"
	"net/http"
	"os"

	"os/signal"
	"syscall"
	"time"

	"github.com/suman9054/supersand/menager"
	"github.com/suman9054/supersand/process"

	"github.com/suman9054/supersand/store"
)

func main() {

   if len(os.Args)>1 && os.Args[1]=="child"{
	
	 if err:=process.RunContainer();err!=nil{
	  slog.Error("error in running container",err)
	 }
	 return
   }

	app := http.NewServeMux()
    Jobs:=make(chan menager.Processchannel,100)
    s:=store.Newstore()
	
    go menager.Killer(s)
	
	app.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello from supersand"))
	})

	app.HandleFunc("POST /make",func(w http.ResponseWriter, r *http.Request) {

		type request struct {
			User string `json:"user"`
		}

		var req request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "invalid request body",
			})
			return
		}

		res:=make(chan store.Responschannel)

		tsk:= store.Prioritytaskvalue{
			Tasktype: store.Startnewsesion,
			Sesioninfo: store.Sesioninfo{
				User: req.User,
			},
			Respons: res,
		}
		  s.Querys.Enqueue(tsk)
		

			w.Header().Set("Content-Type", "application/json")
			val:=<-res
			if val.Status != 200{
				w.WriteHeader(http.StatusInternalServerError)
			} 

		respons:= map[string]interface{}{
			"message":val.Msg,
			"status": val.Status,
		}
		json.NewEncoder(w).Encode(respons)

	})

	app.HandleFunc("POST /run", func(w http.ResponseWriter, r *http.Request) {
		type request struct {
			User    string `json:"user"`
			Comand  string `json:"comand"`
		}
		var req request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "invalid request body",
			})
			return
		}
		res:=make(chan store.Responschannel)

		tsk:= store.Unprioritytasks{
			Tasktype: store.Runcomand,
			Comand: req.Comand,
			Respons: res,
			Sesioninfo: store.Sesioninfo{
				User: req.User,
			},
		}
		  s.Tasks.Enqueue(tsk)
		w.Header().Set("Content-Type", "application/json")
		val:=<-res
		if val.Status != 200{
			w.WriteHeader(http.StatusInternalServerError)
		}
		respons:= map[string]interface{}{
			"message":val.Msg,
			"status": val.Status,
		}
		json.NewEncoder(w).Encode(respons)
	})  
	

	for i:=1;i<=5;i++{
       go menager.Worker(Jobs,s)
	}

	go menager.Menager(Jobs,s)

	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: app,
	}


	
	
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	slog.Info("starting server at http://127.0.0.1:8080")

	
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	
	<-stop

	slog.Info("shutting down server...")

	
	ctx, cancel := context.WithTimeout(context.Background(),5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced shutdown", slog.String("err", err.Error()))
	}

	slog.Info("server exited properly")

}
