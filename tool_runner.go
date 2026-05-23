// Copyright 2026 qtopie
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of antigravity-sdk-go, a Go port of the Google Antigravity
// Python SDK (https://github.com/google/antigravity-sdk-python), which is
// licensed under the Apache License 2.0. This port is an independent community
// contribution and is NOT affiliated with or endorsed by Google LLC.
//
// Original Python SDK: Copyright 2026 Google LLC, Apache-2.0 License.
package antigravity

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// ToolFunc is a function that can be executed as a tool.
// In Go, we can represent tools as functions that take a context and some arguments.
type ToolFunc any

type ToolRunner struct {
	mu            sync.RWMutex
	tools         map[string]ToolFunc
	contextParams map[string]int // maps tool name to the index of the ToolContext parameter
	ctx           *ToolContext
}

func NewToolRunner() *ToolRunner {
	return &ToolRunner{
		tools:         make(map[string]ToolFunc),
		contextParams: make(map[string]int),
	}
}

func (r *ToolRunner) Register(name string, tool ToolFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tools[name]; ok {
		return fmt.Errorf("tool %q already registered", name)
	}

	r.tools[name] = tool
	
	// Reflection to find ToolContext parameter index
	v := reflect.ValueOf(tool)
	t := v.Type()
	if t.Kind() != reflect.Func {
		return fmt.Errorf("tool %q is not a function", name)
	}
	
	for i := 0; i < t.NumIn(); i++ {
		paramType := t.In(i)
		// Check if it's *ToolContext
		if paramType == reflect.TypeOf(&ToolContext{}) {
			r.contextParams[name] = i
			break
		}
	}

	return nil
}

func (r *ToolRunner) SetContext(ctx *ToolContext) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ctx = ctx
}

func (r *ToolRunner) Execute(ctx context.Context, name string, args map[string]any) (any, error) {
	r.mu.RLock()
	fn, ok := r.tools[name]
	ctxIdx, hasCtx := r.contextParams[name]
	toolCtx := r.ctx
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}

	v := reflect.ValueOf(fn)
	t := v.Type()
	
	in := make([]reflect.Value, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		if hasCtx && i == ctxIdx {
			in[i] = reflect.ValueOf(toolCtx)
			continue
		}
		
		// Map args from map[string]any to parameters
		// This is a simplified implementation. Real implementation would use JSON tags or parameter names.
		// For now, let's assume arguments are passed in order if not named, or use a map.
		// In a real Go SDK, tools often take a single "Args" struct.
	}
	
	results := v.Call(in)
	
	if len(results) == 0 {
		return nil, nil
	}
	
	// Handle (any, error) return pattern
	if len(results) == 2 {
		errVal := results[1].Interface()
		if errVal != nil {
			return results[0].Interface(), errVal.(error)
		}
		return results[0].Interface(), nil
	}
	
	return results[0].Interface(), nil
}

// ToolContext provides workspace and session information to tools.
type ToolContext struct {
	conn Connection
}

func NewToolContext(conn Connection) *ToolContext {
	return &ToolContext{conn: conn}
}
