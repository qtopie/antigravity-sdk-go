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
	"sync"
)

// ConversationState manages the stateful session wrapping a single conversation.
type ConversationState struct {
	mu                  sync.RWMutex
	conn                Connection
	steps               []Step
	turnStartIndices    []int
	compactionIndices   []int
	maxHistorySize      int
	cumulativeUsage     UsageMetadata
	turnUsage           *UsageMetadata
}

const DefaultMaxHistorySize = 10000

func NewConversationState(conn Connection) *ConversationState {
	return &ConversationState{
		conn:             conn,
		maxHistorySize:   DefaultMaxHistorySize,
		cumulativeUsage:  UsageMetadata{},
	}
}
// ChatResponseGo provides access to the turn response, similar to types.ChatResponse.
type ChatResponseGo struct {
	chunks <-chan StreamChunk // Thought, Text, or ToolCall
	conv   *ConversationState
}

func (r *ChatResponseGo) Chunks() <-chan StreamChunk {
	return r.chunks
}

// Additional helpers for ChatResponseGo could go here (Text(), Resolve(), etc.)

func (c *ConversationState) Send(ctx context.Context, prompt ContentPrimitive) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.turnStartIndices = append(c.turnStartIndices, len(c.steps))
	c.turnUsage = nil

	return c.conn.Send(ctx, prompt, nil)
}

func (c *ConversationState) ReceiveChunks(ctx context.Context) <-chan StreamChunk {
	out := make(chan StreamChunk)
	steps, err := c.conn.ReceiveSteps(ctx)
	if err != nil {
		close(out)
		return out
	}

	go func() {
		defer close(out)
		for step := range steps {
			c.mu.Lock()
			c.steps = append(c.steps, step)
			// Token usage tracking could be added here
			c.mu.Unlock()

			// Simple mapping of Step to Chunks (matching Python logic)
			if step.Source == StepSourceModel && step.Target == StepTargetUser {
				if step.ThinkingDelta != "" {
					out <- Thought{Content: step.ThinkingDelta}
				}
				if step.ContentDelta != "" {
					out <- Text{Content: step.ContentDelta}
				}
			}
			// Mapping ToolCalls
			for _, tc := range step.ToolCalls {
				out <- tc
			}
		}
	}()
	return out
}

func (c *ConversationState) History() []Step {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]Step{}, c.steps...)
}

func (c *ConversationState) Chat(ctx context.Context, prompt ContentPrimitive) (*ChatResponseGo, error) {
	if err := c.Send(ctx, prompt); err != nil {
		return nil, err
	}
	return &ChatResponseGo{
		chunks: c.ReceiveChunks(ctx),
		conv:   c,
	}, nil
}

func (c *ConversationState) Close(ctx context.Context) error {
	return c.conn.Close(ctx)
}
