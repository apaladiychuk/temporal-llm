package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/example/temporal-llm/internal/activities"
	"github.com/example/temporal-llm/internal/contracts"
	"github.com/example/temporal-llm/internal/workflows"
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

	workflowWorker := worker.New(c, workflows.WorkflowTaskQueue(), worker.Options{})
	workflowWorker.RegisterWorkflowWithOptions(
		workflows.LLMJobWorkflow,
		workflow.RegisterOptions{Name: contracts.WorkflowTypeLLMJob},
	)

	notifyWorker := worker.New(c, contracts.NotifyTaskQueue, worker.Options{})
	notifyWorker.RegisterActivity(activities.NotifyUI)

	errCh := make(chan error, 2)

	go func() {
		errCh <- workflowWorker.Run(worker.InterruptCh())
	}()

	go func() {
		errCh <- notifyWorker.Run(worker.InterruptCh())
	}()

	remaining := 2

	select {
	case err := <-errCh:
		remaining--
		if err != nil {
			log.Fatalf("temporal worker exited with error: %v", err)
		}
		log.Println("temporal worker stopped")
	case sig := <-sigCh:
		log.Printf("received signal %s, shutting down workers", sig)
	}

	workflowWorker.Stop()
	notifyWorker.Stop()

	for i := 0; i < remaining; i++ {
		if err := <-errCh; err != nil {
			log.Printf("worker exit error: %v", err)
		}
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
