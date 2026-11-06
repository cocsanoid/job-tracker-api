package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cocsanoid/job-tracker-api/internal/job"
)

func main() {
	port := getEnv("PORT", "8080")
	dataFile := getEnv("DATA_FILE", "data.json")

	store, err := job.NewStore(dataFile)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           job.NewHandler(store).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Сервер работает в отдельной горутине, main ждёт сигнал остановки (Ctrl+C).
	go func() {
		log.Printf("server listening on :%s (data: %s)", port, dataFile)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
