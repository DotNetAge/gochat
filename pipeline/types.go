package pipeline

import "context"

// Step defines an individual operation in the pipeline.
type Step[T any] interface {
	// Name returns the name of the step for logging and debugging.
	Name() string

	// Execute performs the step's operation, reading from and writing to the State.
	Execute(ctx context.Context, state T) error
}

// Hook allows observing pipeline execution.
type Hook[T any] interface {
	// OnStepStart is called before a step executes.
	OnStepStart(ctx context.Context, step Step[T], state T)

	// OnStepError is called if a step returns an error.
	OnStepError(ctx context.Context, step Step[T], state T, err error)

	// OnStepComplete is called after a step successfully executes.
	OnStepComplete(ctx context.Context, step Step[T], state T)
}
