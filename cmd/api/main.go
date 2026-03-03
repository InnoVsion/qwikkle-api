package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qwikkle-api/internal/config"
	"qwikkle-api/internal/server"
)

func main() {
	cfg := config.Load()

	srv := server.New(cfg)
	httpServer := srv.HTTPServer()

	go func() {
		log.Printf("qwikkle-api listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	} else {
		log.Println("server stopped")
	}
}
