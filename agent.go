// Copyright 2026 qtopie
// SPDX-License-Identifier: Apache-2.0

package antigravity

import (
	"context"
	"errors"
)

type AgentConfig struct {
	SystemInstructions any
	Capabilities       CapabilitiesConfig
	Tools              []any
	MCPServers         []McpServerConfig
	ResponseSchema     *string
	Hooks              []any
	Triggers           []any
	Policies           []any
	CreateStrategy     func(tools []any, hooks []any) (ConnectionStrategy, error)
}

type ConnectionStrategy interface {
	Connect(ctx context.Context) (Connection, error)
	Close(ctx context.Context) error
}

type Connection interface {
	Send(ctx context.Context, prompt ContentPrimitive, options map[string]any) error
	ReceiveSteps(ctx context.Context) (<-chan Step, error)
	IsIdle() bool
	WaitForIdle(ctx context.Context) error
	Close(ctx context.Context) error
}

type Conversation interface {
	Chat(ctx context.Context, prompt ContentPrimitive) (*ChatResponseGo, error)
	Close(ctx context.Context) error
}

type Agent struct {
	config       *AgentConfig
	conversation Conversation
	started      bool
}

func NewAgent(config AgentConfig) *Agent {
	cfg := config
	if cfg.ResponseSchema != nil {
		cfg.Capabilities.FinishToolSchemaJSON = cfg.ResponseSchema
	}
	return &Agent{config: &cfg}
}

func (a *Agent) Start(ctx context.Context) error {
	if a.config.CreateStrategy == nil {
		return errors.New("AgentConfig.CreateStrategy is required")
	}
	strategy, err := a.config.CreateStrategy(a.config.Tools, a.config.Hooks)
	if err != nil {
		return err
	}
	conn, err := strategy.Connect(ctx)
	if err != nil {
		return err
	}
	a.conversation = NewConversationState(conn)
	a.started = true
	return nil
}

func (a *Agent) Stop(ctx context.Context) error {
	if !a.started || a.conversation == nil {
		return nil
	}
	err := a.conversation.Close(ctx)
	a.conversation = nil
	a.started = false
	return err
}

func (a *Agent) Chat(ctx context.Context, prompt ContentPrimitive) (*ChatResponseGo, error) {
	if !a.started || a.conversation == nil {
		return nil, errors.New("Agent session not started. Call Start() first")
	}
	return a.conversation.Chat(ctx, prompt)
}

func (a *Agent) IsStarted() bool { return a.started }

type StreamChunk interface{ IsStreamChunk() }
type Thought struct{ Content string }

func (Thought) IsStreamChunk()   {}
func (t Thought) String() string { return t.Content }

type Text struct{ Content string }

func (Text) IsStreamChunk()   {}
func (t Text) String() string { return t.Content }

// ToolCall implements StreamChunk
func (ToolCall) IsStreamChunk() {}
