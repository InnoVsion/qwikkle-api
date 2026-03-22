package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"qwikkle-api/internal/config"
	"qwikkle-api/internal/db"
	"qwikkle-api/internal/logger"
	"qwikkle-api/internal/server"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool := db.New(ctx)
	logg := logger.New()
	defer logg.Sync()

	startupCtx, startupCancel := context.WithTimeout(ctx, 5*time.Second)
	defer startupCancel()
	if err := pool.Ping(startupCtx); err != nil {
		logg.Fatal("postgres ping failed", zap.Error(err))
	}
	var databaseName string
	_ = pool.QueryRow(startupCtx, "SELECT current_database()").Scan(&databaseName)
	logg.Info("postgres connected", zap.String("database", databaseName))

	if db.MigrationsEnabled() {
		logg.Info("running migrations")
		if err := db.RunMigrations(ctx, os.Getenv("POSTGRES_DSN"), "internal/db/migrations"); err != nil {
			log.Fatalf("migrations failed: %v", err)
		}
		logg.Info("migrations complete")
	}

	srv := server.New(cfg, pool, logg)
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
