// Copyright 2026 qtopie
// SPDX-License-Identifier: Apache-2.0

package antigravity

import (
	"context"
	"sync"
)

type ConversationState struct {
	mu                sync.RWMutex
	conn              Connection
	steps             []Step
	turnStartIndices  []int
	compactionIndices []int
	maxHistorySize    int
	cumulativeUsage   UsageMetadata
	turnUsage         *UsageMetadata
}

const DefaultMaxHistorySize = 10000

func NewConversationState(conn Connection) *ConversationState {
	return &ConversationState{
		conn:           conn,
		maxHistorySize: DefaultMaxHistorySize,
	}
}

type ChatResponseGo struct {
	chunks <-chan StreamChunk
	conv   *ConversationState
}

func (r *ChatResponseGo) Chunks() <-chan StreamChunk {
	return r.chunks
}

func (c *ConversationState) Send(ctx context.Context, prompt ContentPrimitive) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.turnStartIndices = append(c.turnStartIndices, len(c.steps))
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
			c.mu.Unlock()

			// Check for model response
			if step.Source == StepSourceModel && step.Target == StepTargetUser {
				if step.ThinkingDelta != "" {
					out <- Thought{Content: step.ThinkingDelta}
				} else if step.Thinking != "" && step.Status == StepStatusDone {
					out <- Thought{Content: step.Thinking}
				}

				if step.ContentDelta != "" {
					out <- Text{Content: step.ContentDelta}
				} else if step.Content != "" && step.Status == StepStatusDone {
					out <- Text{Content: step.Content}
				}
			}

			for _, tc := range step.ToolCalls {
				out <- tc
			}

			// Terminate ONLY if it's a completed MODEL response, or if the stream ends
			if step.Source == StepSourceModel && step.Status == StepStatusDone {
				break
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
