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
)

type HookRunner struct {
	SessionContext *HookContext

	OnSessionStartHooks      []OnSessionStartHook
	OnSessionEndHooks        []OnSessionEndHook
	PreTurnHooks             []PreTurnHook
	PostTurnHooks            []PostTurnHook
	PreToolCallDecideHooks   []PreToolCallDecideHook
	PostToolCallHooks        []PostToolCallHook
	OnToolErrorHooks         []OnToolErrorHook
	OnInteractionHooks       []OnInteractionHook
	OnCompactionHooks        []OnCompactionHook
}

func NewHookRunner() *HookRunner {
	return &HookRunner{
		SessionContext: NewHookContext(nil),
	}
}

func (r *HookRunner) RegisterHook(hook Hook) error {
	switch h := hook.(type) {
	case OnSessionStartHook:
		r.OnSessionStartHooks = append(r.OnSessionStartHooks, h)
	case OnSessionEndHook:
		r.OnSessionEndHooks = append(r.OnSessionEndHooks, h)
	case PreTurnHook:
		r.PreTurnHooks = append(r.PreTurnHooks, h)
	case PostTurnHook:
		r.PostTurnHooks = append(r.PostTurnHooks, h)
	case PreToolCallDecideHook:
		r.PreToolCallDecideHooks = append(r.PreToolCallDecideHooks, h)
	case PostToolCallHook:
		r.PostToolCallHooks = append(r.PostToolCallHooks, h)
	case OnToolErrorHook:
		r.OnToolErrorHooks = append(r.OnToolErrorHooks, h)
	case OnInteractionHook:
		r.OnInteractionHooks = append(r.OnInteractionHooks, h)
	case OnCompactionHook:
		r.OnCompactionHooks = append(r.OnCompactionHooks, h)
	default:
		return fmt.Errorf("unknown hook type: %T", hook)
	}
	return nil
}

func (r *HookRunner) DispatchSessionStart(ctx context.Context) error {
	for _, h := range r.OnSessionStartHooks {
		if err := h.Run(ctx, r.SessionContext, nil); err != nil {
			return err
		}
	}
	return nil
}

func (r *HookRunner) DispatchPreTurn(ctx context.Context, prompt ContentPrimitive) (HookResult, *HookContext, error) {
	turnContext := NewHookContext(r.SessionContext)
	for _, h := range r.PreTurnHooks {
		res, err := h.Run(ctx, turnContext, prompt)
		if err != nil {
			return HookResult{}, nil, err
		}
		if !res.Allow {
			return res, turnContext, nil
		}
	}
	return HookResult{Allow: true}, turnContext, nil
}

func (r *HookRunner) DispatchPreToolCall(ctx context.Context, turnCtx *HookContext, toolCall ToolCall) (HookResult, ToolCall, *HookContext, error) {
	opCtx := NewHookContext(turnCtx)
	for _, h := range r.PreToolCallDecideHooks {
		res, err := h.Run(ctx, opCtx, toolCall)
		if err != nil {
			return HookResult{}, toolCall, nil, err
		}
		if !res.Allow {
			return res, toolCall, opCtx, nil
		}
	}
	return HookResult{Allow: true}, toolCall, opCtx, nil
}
