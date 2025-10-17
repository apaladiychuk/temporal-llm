package contracts

import "time"

// JobInput описує payload, що надходить з UI у gateway та прокидається у workflow/активність.
type JobInput struct {
	UserID    string            `json:"user_id"`
	RequestID string            `json:"request_id"`
	Model     string            `json:"model"`
	Prompt    string            `json:"prompt"`
	Params    map[string]string `json:"params"`
}

// JobProgress фіксує останній прогрес GPU-активності.
type JobProgress struct {
	Percent int32  `json:"percent"`
	Stage   string `json:"stage"`
	Message string `json:"message"`
	// UpdatedAt зручний для UI, щоб розуміти, наскільки свіжа інформація.
	UpdatedAt time.Time `json:"updated_at"`
}

// JobResult повертається після успішного виконання пайплайну.
type JobResult struct {
	Output string            `json:"output"`
	Meta   map[string]string `json:"meta"`
}

// JobStatus використовується у Temporal Query для повернення стану workflow.
type JobStatus struct {
	State      string       `json:"state"`
	Progress   *JobProgress `json:"progress,omitempty"`
	StartedAt  time.Time    `json:"started_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
	RunID      string       `json:"run_id"`
	WorkflowID string       `json:"workflow_id"`
}

const (
	StateRunning   = "Running"
	StateCompleted = "Completed"
	StateFailed    = "Failed"
	StateCanceled  = "Canceled"
)

// QueryNames використовуються для Temporal Query API.
const (
	QueryStatus = "GetStatus"
)

// SignalNames використовуються для Temporal Signal API.
const (
	SignalCancel = "Cancel"
)

// Імена workflow та task queue, що використовуються gateway та Temporal worker.
const (
	WorkflowTypeLLMJob   = "LLMJobWorkflow"
	WorkflowTaskQueue    = "go-gateway-workflows"
	GPUActivityTaskQueue = "llm-gpu-activities"
	NotifyTaskQueue      = "notifications-activities"
)

// Helper для побудови deterministic workflowId.
func WorkflowID(userID, requestID string) string {
	return "llmjob-" + userID + "-" + requestID
}
