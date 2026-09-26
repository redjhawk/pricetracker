package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"pricefollower.local/config"
	"pricefollower.local/internal/httpapi"
	"pricefollower.local/internal/service"
	"pricefollower.local/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	database, err := store.Open(cfg)
	if err != nil {
		return err
	}
	defer database.Close()
	items := service.New(cfg, database)
	defer items.Close()
	if err := database.SeedDevelopment(context.Background(), items.NextCheckAt); err != nil {
		return fmt.Errorf("seed development data: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	schedulerDone := make(chan struct{})
	go func() { defer close(schedulerDone); items.RunScheduler(ctx) }()
	defer func() { stop(); <-schedulerDone }()

	api := httpapi.New(cfg, items)
	server := &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port)),
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("PriceFollower listening on http://%s", server.Addr)
		if !cfg.Development {
			log.Printf("Frontend is embedded in this binary; database directory: %s", cfg.DataDirectory)
		}
		serverErrors <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		return nil
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
