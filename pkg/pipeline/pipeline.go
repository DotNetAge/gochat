package pipeline

import (
	"context"
	"errors"
	"fmt"
)

// Pipeline manages a sequence of steps that execute in order.
// It provides hooks for observing execution and supports graceful early exit.
//
// Pipeline is generic over the state type, allowing type-safe data sharing
// between steps.
//
// Example:
//
//	pipeline := pipeline.New[pipeline.State]().
//	    AddStep(step1).
//	    AddStep(step2).
//	    AddHook(observer)
//
//	if err := pipeline.Execute(ctx, state); err != nil {
//	    log.Fatal(err)
//	}
type Pipeline[T any] struct {
	steps []Step[T]
	hooks []Hook[T]
}

// New creates a new, empty Pipeline ready to have steps added.
func New[T any]() *Pipeline[T] {
	return &Pipeline[T]{
		steps: make([]Step[T], 0),
		hooks: make([]Hook[T], 0),
	}
}

// AddStep appends a step to the pipeline and returns the pipeline
// for method chaining.
//
// Parameters:
//   - step: The step to add
//
// Returns the pipeline for chaining
func (p *Pipeline[T]) AddStep(step Step[T]) *Pipeline[T] {
	p.steps = append(p.steps, step)
	return p
}

// AddSteps appends multiple steps to the pipeline and returns the pipeline
// for method chaining.
//
// Parameters:
//   - steps: The steps to add
//
// Returns the pipeline for chaining
func (p *Pipeline[T]) AddSteps(steps ...Step[T]) *Pipeline[T] {
	for _, step := range steps {
		p.AddStep(step)
	}
	return p
}

// AddHook appends an observer hook to the pipeline and returns the pipeline
// for method chaining. Hooks are called at various points during execution
// to allow monitoring or logging.
//
// Parameters:
//   - hook: The hook to add
//
// Returns the pipeline for chaining
func (p *Pipeline[T]) AddHook(hook Hook[T]) *Pipeline[T] {
	p.hooks = append(p.hooks, hook)
	return p
}

// Execute runs all steps sequentially. If any step fails, execution stops
// and the error is returned. If the context is canceled, execution stops
// immediately with the context error.
//
// Hooks are called before and after each step execution to allow monitoring.
//
// Parameters:
//   - ctx: Context for cancellation
//   - state: The state object passed to each step
//
// Returns an error if any step fails or context is canceled
func (p *Pipeline[T]) Execute(ctx context.Context, state T) error {
	for _, step := range p.steps {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("pipeline aborted before step %q: %w", step.Name(), err)
		}

		for _, hook := range p.hooks {
			hook.OnStepStart(ctx, step, state)
		}

		err := step.Execute(ctx, state)

		if err != nil {
			if errors.Is(err, ErrReturn) {
				return nil
			}

			for _, hook := range p.hooks {
				hook.OnStepError(ctx, step, state, err)
			}
			return fmt.Errorf("pipeline step %q failed: %w", step.Name(), err)
		}

		for _, hook := range p.hooks {
			hook.OnStepComplete(ctx, step, state)
		}
	}

	return nil
}
