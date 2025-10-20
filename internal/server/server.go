package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.temporal.io/sdk/client"

	"github.com/example/temporal-llm/internal/contracts"
)

type Server struct {
	temporalClient client.Client
}

func New(c client.Client) *Server {
	return &Server{temporalClient: c}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Post("/jobs", s.handleStartJob)
	r.Get("/jobs/{workflowID}/status", s.handleGetStatus)
	r.Post("/jobs/{workflowID}/cancel", s.handleCancel)
	return r
}

func (s *Server) handleStartJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var input contracts.JobInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	workflowID := contracts.WorkflowID(input.UserID, input.RequestID)
	options := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                contracts.WorkflowTaskQueue,
		WorkflowExecutionTimeout: 8 * time.Hour,
	}

	we, err := s.temporalClient.ExecuteWorkflow(ctx, options, contracts.WorkflowTypeLLMJob, input)
	if err != nil {
		log.Printf("failed to start workflow: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"workflow_id": we.GetID(),
		"run_id":      we.GetRunID(),
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func (s *Server) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "workflowID")
	runID := r.URL.Query().Get("runId")

	resp, err := s.temporalClient.QueryWorkflow(ctx, workflowID, runID, contracts.QueryStatus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var status contracts.JobStatus
	if err := resp.Get(&status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "workflowID")
	runID := r.URL.Query().Get("runId")

	if err := s.temporalClient.SignalWorkflow(ctx, workflowID, runID, contracts.SignalCancel, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}
