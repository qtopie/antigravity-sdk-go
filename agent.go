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
	"errors"
	"log"
)

// AgentConfig represents the configuration for an Agent.
// In Go, since we don't have abstract base classes in the same way,
// we can use a struct with a Strategy factory function, or an interface.
// For simplicity, we define a concrete struct that users can extend or populate.
type AgentConfig struct {
	SystemInstructions  any                   // Can be string, CustomSystemInstructions, or TemplatedSystemInstructions
	Capabilities        CapabilitiesConfig
	Tools               []any                 // Custom tools
	MCPServers          []McpServerConfig
	ResponseSchema      *string               // JSON schema string
	Hooks               []any                 // Hooks
	Triggers            []any                 // Triggers
	Policies            []any                 // Policies
	
	// CreateStrategy is a factory function that creates the connection strategy.
	CreateStrategy func(tools []any, hooks []any) (ConnectionStrategy, error)
}

// ConnectionStrategy is responsible for establishing a Connection.
type ConnectionStrategy interface {
	Connect(ctx context.Context) (Connection, error)
	Close(ctx context.Context) error
}

// Connection represents a connection to the agent backend.
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

// Agent represents the High-level Agent API.
type Agent struct {
	config       *AgentConfig
	conversation Conversation
	started      bool
}

// NewAgent initializes a new Agent.
func NewAgent(config AgentConfig) *Agent {
	// Deep copy or clone logic would go here if necessary
	cfg := config

	if cfg.ResponseSchema != nil {
		cfg.Capabilities.FinishToolSchemaJSON = cfg.ResponseSchema
	}

	return &Agent{
		config: &cfg,
	}
}

// Start starts the agent session.
func (a *Agent) Start(ctx context.Context) error {
	log.Println("Starting Agent session")

	if a.config.CreateStrategy == nil {
		return errors.New("AgentConfig.CreateStrategy is required")
	}

	// Strategy creation
	strategy, err := a.config.CreateStrategy(a.config.Tools, a.config.Hooks)
	if err != nil {
		return err
	}

	// Connection establishment
	conn, err := strategy.Connect(ctx)
	if err != nil {
		return err
	}

	// Conversation initialization
	a.conversation = NewConversationState(conn)
	a.started = true

	return nil
}

// Stop stops the agent session.
func (a *Agent) Stop(ctx context.Context) error {
	log.Println("Stopping Agent session")
	if !a.started || a.conversation == nil {
		return nil
	}
	err := a.conversation.Close(ctx)
	a.conversation = nil
	a.started = false
	return err
}

// Chat sends a prompt and returns the stream of chunks.
func (a *Agent) Chat(ctx context.Context, prompt ContentPrimitive) (*ChatResponseGo, error) {
	if !a.started || a.conversation == nil {
		return nil, errors.New("Agent session not started. Call Start() first")
	}
	return a.conversation.Chat(ctx, prompt)
}

// IsStarted returns whether the agent session has been started.
func (a *Agent) IsStarted() bool {
	return a.started
}

// StreamChunk is an interface that represents a chunk from the chat stream.
type StreamChunk interface {
	IsStreamChunk()
}

type Thought struct {
	Content string
}

func (Thought) IsStreamChunk() {}

func (t Thought) String() string { return t.Content }

type Text struct {
	Content string
}

func (Text) IsStreamChunk() {}

func (t Text) String() string { return t.Content }

func (ToolCall) IsStreamChunk() {}
