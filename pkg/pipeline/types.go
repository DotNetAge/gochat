package pipeline

import "context"

// Step defines an individual operation in the pipeline.
type Step interface {
	// Name returns the name of the step for logging and debugging.
	Name() string
	
	// Execute performs the step's operation, reading from and writing to the State.
	Execute(ctx context.Context, state *State) error
}

// Hook allows observing pipeline execution.
type Hook interface {
	// OnStepStart is called before a step executes.
	OnStepStart(ctx context.Context, step Step, state *State)
	
	// OnStepError is called if a step returns an error.
	OnStepError(ctx context.Context, step Step, state *State, err error)
	
	// OnStepComplete is called after a step successfully executes.
	OnStepComplete(ctx context.Context, step Step, state *State)
}
