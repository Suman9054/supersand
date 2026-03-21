package main

import (
	"context"

	"fmt"

	"encoding/json"

	"log"
	"log/slog"
	"net/http"
	"os"

	"os/signal"
	"syscall"
	"time"

	"github.com/suman9054/supersand/process"
)

func main() {
	app := http.NewServeMux()

	app.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello from supersand"))
	})

	app.HandleFunc("GET /make",func(w http.ResponseWriter, r *http.Request) {
		err:= process.Sandbox().CreateNewContainer()
		if err != nil{
			errrespons :=map[string]interface{}{
				"message":"something went wrong",
				"err":err.Error(),
				"status":400,
			} 

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(errrespons)
		}

		respons:= map[string]interface{}{
			"message":"succesfull",
			"status":200,
		}
		json.NewEncoder(w).Encode(respons)

	})

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
