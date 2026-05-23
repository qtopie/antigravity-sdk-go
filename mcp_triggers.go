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
)

// MCPBridge represents the bridge to MCP servers.
type MCPBridge struct {
	// Implementation would manage connections to stdio/sse/http servers.
}

func NewMCPBridge() *MCPBridge {
	return &MCPBridge{}
}

func (b *MCPBridge) Connect(ctx context.Context, config McpServerConfig) error {
	return nil
}

func (b *MCPBridge) Tools() []any {
	return nil
}

func (b *MCPBridge) Stop(ctx context.Context) error {
	return nil
}

// Trigger represents a condition that can trigger an agent action.
type Trigger interface {
	Run(ctx context.Context, delivery func(ContentPrimitive) error) error
}

// TriggerRunner manages the lifecycle of triggers.
type TriggerRunner struct {
	triggers []Trigger
}

func NewTriggerRunner(triggers []Trigger) *TriggerRunner {
	return &TriggerRunner{triggers: triggers}
}

func (r *TriggerRunner) Start(ctx context.Context) error {
	return nil
}

func (r *TriggerRunner) Stop(ctx context.Context) error {
	return nil
}
