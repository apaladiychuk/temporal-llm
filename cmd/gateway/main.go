package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/example/temporal-llm/internal/activities"
	"github.com/example/temporal-llm/internal/server"
	"github.com/example/temporal-llm/internal/workflows"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	temporalAddress := getenv("TEMPORAL_ADDRESS", "temporal:7233")
	c, err := client.NewClient(client.Options{HostPort: temporalAddress})
	if err != nil {
		log.Fatalf("failed to create Temporal client: %v", err)
	}
	defer c.Close()

	gwWorker := worker.New(c, workflows.WorkflowTaskQueue(), worker.Options{})
	gwWorker.RegisterWorkflow(workflows.LLMJobWorkflow)

	notifyWorker := worker.New(c, workflows.NotifyTaskQueue(), worker.Options{})
	notifyWorker.RegisterActivity(activities.NotifyUI)

	go func() {
		if err := gwWorker.Run(worker.InterruptCh()); err != nil {
			log.Fatalf("workflow worker exited: %v", err)
		}
	}()
	go func() {
		if err := notifyWorker.Run(worker.InterruptCh()); err != nil {
			log.Fatalf("notification worker exited: %v", err)
		}
	}()

	srv := server.New(c)
	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           srv.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("gateway listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
