package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/example/temporal-llm/internal/activities"
	"github.com/example/temporal-llm/internal/contracts"
)

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	temporalAddress := getenv("TEMPORAL_ADDRESS", "temporal:7233")
	c, err := client.NewClient(client.Options{HostPort: temporalAddress})
	if err != nil {
		log.Fatalf("failed to create Temporal client: %v", err)
	}
	defer c.Close()

	notifyWorker := worker.New(c, contracts.NotifyTaskQueue, worker.Options{})
	notifyWorker.RegisterActivity(activities.NotifyUI)

	errCh := make(chan error, 1)

	go func() {
		errCh <- notifyWorker.Run(worker.InterruptCh())
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatalf("notification worker exited with error: %v", err)
		}
		log.Println("notification worker stopped")
	case sig := <-sigCh:
		log.Printf("received signal %s, shutting down notification worker", sig)
	}

	notifyWorker.Stop()

	if err := <-errCh; err != nil {
		log.Printf("notification worker exit error: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
