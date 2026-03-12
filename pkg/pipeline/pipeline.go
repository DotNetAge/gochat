package pipeline

import (
	"context"
	"fmt"
)

// Pipeline manages a sequence of steps.
type Pipeline struct {
	steps []Step
	hooks []Hook
}

// New creates a new, empty pipeline.
func New() *Pipeline {
	return &Pipeline{
		steps: make([]Step, 0),
		hooks: make([]Hook, 0),
	}
}

// AddStep appends a step to the pipeline.
func (p *Pipeline) AddStep(step Step) *Pipeline {
	p.steps = append(p.steps, step)
	return p
}

// AddHook appends an observer hook to the pipeline.
func (p *Pipeline) AddHook(hook Hook) *Pipeline {
	p.hooks = append(p.hooks, hook)
	return p
}

// Execute runs all steps sequentially. If any step fails or the context is canceled,
// execution stops and the error is returned.
func (p *Pipeline) Execute(ctx context.Context, state *State) error {
	if state == nil {
		return fmt.Errorf("pipeline state cannot be nil")
	}

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
