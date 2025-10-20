package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

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

	errCh := make(chan error, 1)

	go func() {
		errCh <- workflowWorker.Run(worker.InterruptCh())
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatalf("temporal workflow worker exited with error: %v", err)
		}
		log.Println("temporal workflow worker stopped")
	case sig := <-sigCh:
		log.Printf("received signal %s, shutting down workflow worker", sig)
	}

	workflowWorker.Stop()

	if err := <-errCh; err != nil {
		log.Printf("workflow worker exit error: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
