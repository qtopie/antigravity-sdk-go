// Copyright 2026 qtopie
// SPDX-License-Identifier: Apache-2.0

package antigravity

import (
	"bufio"
	"context"
	"embed"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/qtopie/antigravity-sdk-go/proto"
	"google.golang.org/protobuf/encoding/protojson"
	googleproto "google.golang.org/protobuf/proto"
)

//go:generate go run download.go
//go:embed bin/localharness
var embeddedHarness embed.FS

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
		tempDir := filepath.Join(os.TempDir(), "antigravity_bin")
		os.MkdirAll(tempDir, 0755)
		binaryPath = filepath.Join(tempDir, "localharness")
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			data, err := embeddedHarness.ReadFile("bin/localharness")
			if err == nil {
				os.WriteFile(binaryPath, data, 0755)
			} else {
				binaryPath = "localharness"
			}
		}
	}

	return &LocalConnectionStrategy{
		binaryPath:         binaryPath,
		geminiConfig:       GeminiConfig{},
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

	inputConfig := &proto.InputConfig{
		StorageDirectory: s.saveDir,
	}
	data, _ := googleproto.Marshal(inputConfig)

	binary.Write(stdin, binary.LittleEndian, uint32(len(data)))
	stdin.Write(data)

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

	wsURL := fmt.Sprintf("ws://localhost:%d/", outputConfig.Port)
	header := http.Header{}
	header.Add("x-goog-api-key", outputConfig.ApiKey)

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

	harnessConfig := &proto.HarnessConfig{
		GeminiConfig: &proto.GeminiConfig{
			ApiKey:    os.Getenv("GEMINI_API_KEY"),
			ModelName: "gemini-3.5-flash",
		},
	}

	if s.systemInstructions != nil {
		if si, ok := s.systemInstructions.(string); ok && si != "" {
			harnessConfig.SystemInstructions = &proto.SystemInstructions{
				Custom: &proto.CustomSystemInstructions{
					Part: []*proto.CustomSystemInstructions_Part{{Text: si}},
				},
			}
		}
	}

	initEvent := &proto.InitializeConversationEvent{
		Config: harnessConfig,
	}
	initJSON, _ := protojson.Marshal(initEvent)
	if err := ws.WriteMessage(websocket.TextMessage, initJSON); err != nil {
		return nil, fmt.Errorf("failed to send init event: %w", err)
	}

	conn := &LocalConnection{
		cmd:    cmd,
		ws:     ws,
		stderr: stderr,
		steps:  make(chan Step, 100),
	}

	go conn.readLoop()
	go conn.stderrLoop()

	return conn, nil
}

func (s *LocalConnectionStrategy) Close(ctx context.Context) error {
	return nil
}

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
	data, _ := protojson.Marshal(&inputEvent)
	return c.ws.WriteMessage(websocket.TextMessage, data)
}

func (c *LocalConnection) ReceiveSteps(ctx context.Context) (<-chan Step, error) {
	return c.steps, nil
}

func (c *LocalConnection) IsIdle() bool                          { return true }
func (c *LocalConnection) WaitForIdle(ctx context.Context) error { return nil }

func (c *LocalConnection) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	c.ws.Close()
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

func (c *LocalConnection) readLoop() {
	defer close(c.steps)
	for {
		messageType, message, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.TextMessage {
			continue
		}

		var event proto.OutputEvent
		if err := protojson.Unmarshal(message, &event); err != nil {
			continue
		}

		if event.StepUpdate != nil {
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
		s.Target = StepTargetEnvironment
	case proto.StepUpdate_TARGET_ENVIRONMENT:
		s.Target = StepTargetEnvironment
	default:
		s.Target = StepTargetUnknown
	}

	switch pb.State {
	case proto.StepUpdate_STATE_ACTIVE:
		s.Status = StepStatusActive
	case proto.StepUpdate_STATE_DONE:
		s.Status = StepStatusDone
		isComp := true
		s.IsCompleteResponse = &isComp
	case proto.StepUpdate_STATE_WAITING_FOR_USER:
		s.Status = StepStatusWaitingForUser
	case proto.StepUpdate_STATE_ERROR:
		s.Status = StepStatusError
	}
	return s
}

func (c *LocalConnection) stderrLoop() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			fmt.Printf("[Harness] %s\n", line)
		}
	}
}
