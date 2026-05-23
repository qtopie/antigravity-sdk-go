// Copyright 2026 qtopie
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of antigravity-sdk-go, a Go port of the Google Antigravity
// Python SDK (https://github.com/google/antigravity-sdk-python), which is
// licensed under the Apache License 2.0. This port is an independent community
// contribution and is NOT affiliated with or endorsed by Google LLC.
//
// Original Python SDK: Copyright 2026 Google LLC, Apache-2.0 License.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/qtopie/antigravity-sdk-go"
)

func main() {
	ctx := context.Background()

	// 1. Define configuration
	config := antigravity.AgentConfig{
		SystemInstructions: "You are a helpful Go assistant.",
		CreateStrategy: func(tools []any, hooks []any) (antigravity.ConnectionStrategy, error) {
			return antigravity.NewLocalConnectionStrategy(antigravity.AgentConfig{
				SystemInstructions: "You are a helpful Go assistant.",
			}), nil
		},
	}

	// 2. Create agent
	agent := antigravity.NewAgent(config)

	// 3. Start session (Skeleton check)
	err := agent.Start(ctx)
	if err != nil {
		log.Printf("Agent start failed (as expected in skeleton): %v", err)
		return
	}
	defer agent.Stop(ctx)

	// 4. Chat
	response, err := agent.Chat(ctx, antigravity.TextContent("Hello!"))
	if err != nil {
		log.Fatal(err)
	}

	for chunk := range response.Chunks() {
		fmt.Printf("Chunk: %v\n", chunk)
	}
}
