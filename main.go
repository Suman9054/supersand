package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	fmt.Println("hallow from super sand")

	app := http.NewServeMux()
	app.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hallow from supersand"))
	})

	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: app,
	}

	stope := make(chan os.Signal, 1)

	slog.Info("starting up surver at localhost:8080")
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal("falid to start the server:", err)
		}
	}()
	<-stope
	slog.Info("shutiing down server....")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if shtdownerr := server.Shutdown(ctx); shtdownerr != nil {
		slog.Info("server frosed shut down:", slog.String("err", shtdownerr.Error()))
	}
}
