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
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/qtopie/antigravity-sdk-go/proto"
	"github.com/gorilla/websocket"
	googleproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

// LocalConnectionStrategy implements the connection strategy for the local harness.
type LocalConnectionStrategy struct {
	binaryPath         string
	toolRunner         *ToolRunner
	hookRunner         *HookRunner
	geminiConfig       GeminiConfig
	systemInstructions any
	capabilities       CapabilitiesConfig
	conversationID     string
	saveDir            string
	workspaces         []string
}

func NewLocalConnectionStrategy(config AgentConfig) *LocalConnectionStrategy {
	binaryPath := os.Getenv("ANTIGRAVITY_HARNESS_PATH")
	if binaryPath == "" {
		binaryPath = "localharness" // Fallback to PATH
	}

	return &LocalConnectionStrategy{
		binaryPath:         binaryPath,
		geminiConfig:       GeminiConfig{}, // In a real app, populate from config
		systemInstructions: config.SystemInstructions,
		capabilities:       config.Capabilities,
		workspaces:         []string{"."},
		saveDir:            filepath.Join(os.TempDir(), "antigravity"),
	}
}

func (s *LocalConnectionStrategy) Connect(ctx context.Context) (Connection, error) {
	cmd := exec.CommandContext(ctx, s.binaryPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start localharness: %w", err)
	}

	// 1. Handshake over Stdio
	inputConfig := &proto.InputConfig{
		StorageDirectory: s.saveDir,
	}
	data, _ := googleproto.Marshal(inputConfig)
	
	binary.Write(stdin, binary.LittleEndian, uint32(len(data)))
	stdin.Write(data)

	// Read OutputConfig
	var length uint32
	if err := binary.Read(stdout, binary.LittleEndian, &length); err != nil {
		return nil, fmt.Errorf("failed to read handshake length: %w", err)
	}
	respData := make([]byte, length)
	if _, err := io.ReadFull(stdout, respData); err != nil {
		return nil, fmt.Errorf("failed to read handshake response: %w", err)
	}

	outputConfig := &proto.OutputConfig{}
	if err := googleproto.Unmarshal(respData, outputConfig); err != nil {
		return nil, fmt.Errorf("failed to parse output config: %w", err)
	}

	// 2. Connect via WebSocket (local, no proxy needed)
	wsURL := fmt.Sprintf("ws://localhost:%d/", outputConfig.Port)
	header := http.Header{}
	header.Add("x-goog-api-key", outputConfig.ApiKey)

	// Use a dialer that respects HTTPS_PROXY/ALL_PROXY env vars for outbound
	// connections but not for localhost (the harness WS is local).
	dialer := &websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial(network, addr)
		},
		Proxy: http.ProxyFromEnvironment,
	}
	ws, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to websocket: %w", err)
	}

	// 3. Initialize Conversation
	harnessConfig := &proto.HarnessConfig{
		GeminiConfig: &proto.GeminiConfig{
			ApiKey:    os.Getenv("GEMINI_API_KEY"),
			ModelName: "gemini-2.5-flash",
		},
	}
	initEvent := &proto.InitializeConversationEvent{
		Config: harnessConfig,
	}
	log.Printf("Initializing conversation with model: %s", harnessConfig.GeminiConfig.ModelName)
	// Use protojson to correctly serialize proto message with camelCase field names
	initJSON, err := protojson.Marshal(initEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal init event: %w", err)
	}
	if err := ws.WriteMessage(websocket.TextMessage, initJSON); err != nil {
		return nil, fmt.Errorf("failed to send init event: %w", err)
	}

	conn := &LocalConnection{
		cmd:    cmd,
		ws:     ws,
		stderr: stderr,
		steps:  make(chan Step, 100),
	}


	// Start background readers
	go conn.readLoop()
	go conn.stderrLoop()

	return conn, nil
}

func (s *LocalConnectionStrategy) Close(ctx context.Context) error {
	return nil
}

// LocalConnection implements the Connection interface for the local harness.
type LocalConnection struct {
	cmd    *exec.Cmd
	ws     *websocket.Conn
	stderr io.ReadCloser
	steps  chan Step
	mu     sync.Mutex
	closed bool
}

func (c *LocalConnection) Send(ctx context.Context, prompt ContentPrimitive, options map[string]any) error {
	var inputEvent proto.InputEvent

	switch p := prompt.(type) {
	case TextContent:
		inputEvent.UserInput = string(p)
	}

	log.Printf("Sending InputEvent: user_input=%q", inputEvent.UserInput)
	data, err := protojson.Marshal(&inputEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal InputEvent: %w", err)
	}
	return c.ws.WriteMessage(websocket.TextMessage, data)
}

func (c *LocalConnection) ReceiveSteps(ctx context.Context) (<-chan Step, error) {
	return c.steps, nil
}

func (c *LocalConnection) IsIdle() bool {
	return true // Simplified
}

func (c *LocalConnection) WaitForIdle(ctx context.Context) error {
	return nil
}

func (c *LocalConnection) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	c.ws.Close()
	return c.cmd.Process.Kill()
}

func (c *LocalConnection) readLoop() {
	defer close(c.steps)
	for {
		messageType, message, err := c.ws.ReadMessage()
		if err != nil {
			log.Printf("WS read error: %v", err)
			return
		}
		if messageType != websocket.TextMessage {
			continue
		}

		var event proto.OutputEvent
		if err := protojson.Unmarshal(message, &event); err != nil {
			log.Printf("Failed to unmarshal OutputEvent: %v. Data: %s", err, string(message))
			continue
		}

		if event.StepUpdate != nil {
			log.Printf("Received StepUpdate: Text=%q, State=%v", event.StepUpdate.Text, event.StepUpdate.State)
			step := mapProtoStepToSDK(event.StepUpdate)
			c.steps <- step
		}
	}
}

func mapProtoStepToSDK(pb *proto.StepUpdate) Step {
	s := Step{
		ID:            pb.TrajectoryId,
		StepIndex:     int(pb.StepIndex),
		Content:       pb.Text,
		ContentDelta:  pb.TextDelta,
		Thinking:      pb.Thinking,
		ThinkingDelta: pb.ThinkingDelta,
	}

	switch pb.Source {
	case proto.StepUpdate_SOURCE_SYSTEM:
		s.Source = StepSourceSystem
	case proto.StepUpdate_SOURCE_USER:
		s.Source = StepSourceUser
	case proto.StepUpdate_SOURCE_MODEL:
		s.Source = StepSourceModel
	default:
		s.Source = StepSourceUnknown
	}

	switch pb.Target {
	case proto.StepUpdate_TARGET_USER:
		s.Target = StepTargetUser
	case proto.StepUpdate_TARGET_MODEL:
		s.Target = StepTargetEnvironment // Note: target mapping might vary
	case proto.StepUpdate_TARGET_ENVIRONMENT:
		s.Target = StepTargetEnvironment
	default:
		s.Target = StepTargetUnknown
	}

	return s
}

func (c *LocalConnection) stderrLoop() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		log.Printf("[Harness Stderr] %s", scanner.Text())
	}
}
