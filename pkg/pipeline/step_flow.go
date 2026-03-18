package pipeline

import (
	"context"
	"errors"
)

// 定义控制流哨兵错误
var (
	ErrBreak  = errors.New("pipeline loop break")
	ErrReturn = errors.New("pipeline execution return")
)

// IfStep 提供管线中的条件分支逻辑。
type IfStep[T any] struct {
	StepName  string
	Condition func(ctx context.Context, state T) bool
	Then      Step[T]
	Else      Step[T]
}

func NewIf[T any](name string, cond func(context.Context, T) bool, thenStep, elseStep Step[T]) *IfStep[T] {
	return &IfStep[T]{
		StepName:  name,
		Condition: cond,
		Then:      thenStep,
		Else:      elseStep,
	}
}

func (s *IfStep[T]) Name() string {
	return s.StepName
}

func (s *IfStep[T]) Execute(ctx context.Context, state T) error {
	if s.Condition(ctx, state) {
		if s.Then != nil {
			return s.Then.Execute(ctx, state)
		}
	} else {
		if s.Else != nil {
			return s.Else.Execute(ctx, state)
		}
	}
	return nil
}

// LoopStep 提供管线中的重复执行逻辑，支持 ErrBreak。
type LoopStep[T any] struct {
	StepName  string
	Condition func(context.Context, T) bool
	Body      Step[T]
	MaxLoops  int
}

func NewLoop[T any](name string, cond func(context.Context, T) bool, body Step[T], maxLoops int) *LoopStep[T] {
	return &LoopStep[T]{
		StepName:  name,
		Condition: cond,
		Body:      body,
		MaxLoops:  maxLoops,
	}
}

func (s *LoopStep[T]) Name() string {
	return s.StepName
}

func (s *LoopStep[T]) Execute(ctx context.Context, state T) error {
	for i := 0; i < s.MaxLoops; i++ {
		if s.Condition != nil && !s.Condition(ctx, state) {
			break
		}

		err := s.Body.Execute(ctx, state)
		if err != nil {
			if errors.Is(err, ErrBreak) {
				return nil // 正常跳出
			}
			return err
		}
	}
	return nil
}

// ReturnStep 终止整个 Pipeline 执行。
type ReturnStep[T any] struct {
	StepName string
}

func (s *ReturnStep[T]) Name() string {
	return s.StepName
}

func (s *ReturnStep[T]) Execute(ctx context.Context, state T) error {
	return ErrReturn
}

// BreakStep 跳出当前 Loop。
type BreakStep[T any] struct {
	StepName string
}

func (s *BreakStep[T]) Name() string {
	return s.StepName
}

func (s *BreakStep[T]) Execute(ctx context.Context, state T) error {
	return ErrBreak
}
