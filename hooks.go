// Copyright 2026 qtopie
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of antigravity-sdk-go, a Go port of the Google Antigravity
// Python SDK (https://github.com/google-antigravity/antigravity-sdk-python), which is
// licensed under the Apache License 2.0. This port is an independent community
// contribution and is NOT affiliated with or endorsed by Google LLC.
//
// Original Python SDK: Copyright 2026 Google LLC, Apache-2.0 License.
package antigravity

import (
	"context"
	"sync"
)

// HookContext provides a way for hooks to share state.
type HookContext struct {
	mu     sync.RWMutex
	parent *HookContext
	store  map[string]any
}

func NewHookContext(parent *HookContext) *HookContext {
	return &HookContext{
		parent: parent,
		store:  make(map[string]any),
	}
}

func (c *HookContext) Get(key string, defaultValue any) any {
	c.mu.RLock()
	val, ok := c.store[key]
	c.mu.RUnlock()
	if ok {
		return val
	}
	if c.parent != nil {
		return c.parent.Get(key, defaultValue)
	}
	return defaultValue
}

func (c *HookContext) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = value
}

// Hook interfaces

type InspectHook[T any] interface {
	Run(ctx context.Context, hctx *HookContext, data T) error
}

type DecideHook[T any] interface {
	Run(ctx context.Context, hctx *HookContext, data T) (HookResult, error)
}

type TransformHook[T any, R any] interface {
	Run(ctx context.Context, hctx *HookContext, data T) (R, error)
}

// Concrete hook types

type PreTurnHook DecideHook[ContentPrimitive]
type PostTurnHook InspectHook[string]
type PreToolCallDecideHook DecideHook[ToolCall]
type PostToolCallHook InspectHook[ToolResult]
type OnToolErrorHook TransformHook[error, any]
type OnInteractionHook TransformHook[AskQuestionInteractionSpec, QuestionHookResult]
type OnCompactionHook InspectHook[any]
type OnSessionStartHook InspectHook[any]
type OnSessionEndHook InspectHook[any]

// Hook is a marker interface for all hooks
type Hook any

// Functional hook helpers

type InspectFunc[T any] func(ctx context.Context, hctx *HookContext, data T) error
func (f InspectFunc[T]) Run(ctx context.Context, hctx *HookContext, data T) error {
	return f(ctx, hctx, data)
}

type DecideFunc[T any] func(ctx context.Context, hctx *HookContext, data T) (HookResult, error)
func (f DecideFunc[T]) Run(ctx context.Context, hctx *HookContext, data T) (HookResult, error) {
	return f(ctx, hctx, data)
}

type TransformFunc[T any, R any] func(ctx context.Context, hctx *HookContext, data T) (R, error)
func (f TransformFunc[T, R]) Run(ctx context.Context, hctx *HookContext, data T) (R, error) {
	return f(ctx, hctx, data)
}
