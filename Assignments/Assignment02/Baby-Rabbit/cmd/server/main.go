package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	httpDelivery "Baby-Rabbit/internal/delivery/http"
	"Baby-Rabbit/internal/pkg/clock"
	"Baby-Rabbit/internal/pkg/idgen"
	"Baby-Rabbit/internal/pkg/logger"
	"Baby-Rabbit/internal/repository"
	"Baby-Rabbit/internal/service"
	"Baby-Rabbit/internal/usecase"
)

const (
	listenAddr       = ":8080"
	ttlSweepInterval = time.Second
	shutdownTimeout  = 5 * time.Second
)

// main is the composition root: it instantiates concrete adapters and
// injects them into the inner layers. Inner layers know nothing about
// gin, uuid, zap, etc.
func main() {
	zlog, err := logger.NewZap()
	if err != nil {
		log.Fatalf("logger init: %v", err)
	}
	defer zlog.Sync()

	manager := repository.NewQueueManager(repository.RingBufferFactory{})
	svc := usecase.NewQueueUseCase(manager, idgen.UUID{}, clock.Real{})
	handler := httpDelivery.NewHandler(svc)
	router := httpDelivery.NewRouter(handler)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cleaner := service.NewTTLCleaner(manager, zlog, ttlSweepInterval)
	go cleaner.Run(ctx)

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		zlog.Infof("Baby-Rabbit listening on %s", listenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zlog.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	zlog.Infof("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		zlog.Errorf("graceful shutdown failed: %v", err)
	}
}
