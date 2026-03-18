package pipeline

import (
	"context"
	"errors"
	"fmt"
)

// Pipeline manages a sequence of steps.
type Pipeline[T any] struct {
	steps []Step[T]
	hooks []Hook[T]
}

// New creates a new, empty pipeline.
func New[T any]() *Pipeline[T] {
	return &Pipeline[T]{
		steps: make([]Step[T], 0),
		hooks: make([]Hook[T], 0),
	}
}

// AddStep appends a step to the pipeline.
func (p *Pipeline[T]) AddStep(step Step[T]) *Pipeline[T] {
	p.steps = append(p.steps, step)
	return p
}

func (p *Pipeline[T]) AddSteps(steps ...Step[T]) *Pipeline[T] {
	for _, step := range steps {
		p.AddStep(step)
	}
	return p
}

// AddHook appends an observer hook to the pipeline.
func (p *Pipeline[T]) AddHook(hook Hook[T]) *Pipeline[T] {
	p.hooks = append(p.hooks, hook)
	return p
}

// Execute runs all steps sequentially. If any step fails or the context is canceled,
// execution stops and the error is returned.
func (p *Pipeline[T]) Execute(ctx context.Context, state T) error {
	for _, step := range p.steps {
		// Check for context cancellation before starting next step
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("pipeline aborted before step %q: %w", step.Name(), err)
		}

		// Trigger OnStepStart hooks
		for _, hook := range p.hooks {
			hook.OnStepStart(ctx, step, state)
		}

		// Execute step
		err := step.Execute(ctx, state)

		if err != nil {
			if errors.Is(err, ErrReturn) {
				return nil // 优雅退出
			}

			// Trigger OnStepError hooks
			for _, hook := range p.hooks {
				hook.OnStepError(ctx, step, state, err)
			}
			return fmt.Errorf("pipeline step %q failed: %w", step.Name(), err)
		}

		// Trigger OnStepComplete hooks
		for _, hook := range p.hooks {
			hook.OnStepComplete(ctx, step, state)
		}
	}

	return nil
}
