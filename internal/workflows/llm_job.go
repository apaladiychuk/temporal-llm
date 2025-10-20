package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/example/temporal-llm/internal/activities"
	"github.com/example/temporal-llm/internal/contracts"
)

// StatusState містить mutable state workflow та використовується у Query handler.
type StatusState struct {
	contracts.JobStatus
}

// LLMJobWorkflow orchestration.
func LLMJobWorkflow(ctx workflow.Context, input contracts.JobInput) (*contracts.JobResult, error) {
	state := &StatusState{
		JobStatus: contracts.JobStatus{
			State:      contracts.StateRunning,
			StartedAt:  workflow.Now(ctx).UTC(),
			UpdatedAt:  workflow.Now(ctx).UTC(),
			WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID,
			RunID:      workflow.GetInfo(ctx).WorkflowExecution.RunID,
		},
	}

	if err := workflow.SetQueryHandler(ctx, contracts.QueryStatus, func() (contracts.JobStatus, error) {
		return state.JobStatus, nil
	}); err != nil {
		return nil, err
	}

	ctx, cancel := workflow.WithCancel(ctx)
	cancelSignal := workflow.GetSignalChannel(ctx, contracts.SignalCancel)
	workflow.Go(ctx, func(gctx workflow.Context) {
		var payload interface{}
		cancelSignal.Receive(gctx, &payload)
		state.State = contracts.StateCanceled
		state.UpdatedAt = workflow.Now(gctx).UTC()
		cancel()
	})

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout:    2 * time.Hour,
		ScheduleToStartTimeout: 5 * time.Minute,
		HeartbeatTimeout:       30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
		TaskQueue: contracts.GPUActivityTaskQueue,
	}
	actCtx := workflow.WithActivityOptions(ctx, activityOpts)

	var result contracts.JobResult
	future := workflow.ExecuteActivity(actCtx, "RunLLMOnGPU", input)
	if err := future.Get(actCtx, &result); err != nil {
		if temporal.IsCanceledError(err) {
			state.State = contracts.StateCanceled
		} else {
			state.State = contracts.StateFailed
		}
		state.UpdatedAt = workflow.Now(ctx).UTC()
		return nil, err
	}

	state.State = contracts.StateCompleted
	state.JobStatus.Progress = &contracts.JobProgress{
		Percent:   100,
		Stage:     "completed",
		Message:   "Workflow finished successfully",
		UpdatedAt: workflow.Now(ctx).UTC(),
	}
	state.UpdatedAt = workflow.Now(ctx).UTC()

	notifyOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
		TaskQueue: contracts.NotifyTaskQueue,
	}
	notifyCtx := workflow.WithActivityOptions(ctx, notifyOpts)
	notifyPayload := activities.NotificationPayload{}
	notifyPayload.Input.UserID = input.UserID
	notifyPayload.Input.RequestID = input.RequestID
	notifyPayload.Input.Model = input.Model
	notifyPayload.Input.Prompt = input.Prompt
	notifyPayload.Input.Params = input.Params
	notifyPayload.Result.Output = result.Output
	notifyPayload.Result.Meta = result.Meta
	_ = workflow.ExecuteActivity(notifyCtx, "NotifyUI", notifyPayload).Get(notifyCtx, nil)

	return &result, nil
}

// WorkflowTaskQueue повертає queue.
func WorkflowTaskQueue() string { return contracts.WorkflowTaskQueue }

// NotifyTaskQueue повертає queue для нотифікацій.
func NotifyTaskQueue() string { return contracts.NotifyTaskQueue }
