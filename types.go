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
	"errors"
	"mime"
	"os"
	"path/filepath"
)

// Config types

const (
	DefaultModel               = "gemini-3.5-flash"
	DefaultImageGenerationModel = "gemini-3.1-flash-image-preview"
)

type ThinkingLevel string

const (
	ThinkingLevelMinimal ThinkingLevel = "minimal"
	ThinkingLevelLow     ThinkingLevel = "low"
	ThinkingLevelMedium  ThinkingLevel = "medium"
	ThinkingLevelHigh    ThinkingLevel = "high"
)

type GenerationConfig struct {
	ThinkingLevel *ThinkingLevel `json:"thinking_level,omitempty"`
}

type ModelEntry struct {
	Name       string           `json:"name"`
	APIKey     *string          `json:"api_key,omitempty"`
	Generation GenerationConfig `json:"generation"`
}

type ModelConfig struct {
	Default         ModelEntry `json:"default"`
	ImageGeneration ModelEntry `json:"image_generation"`
}

type GeminiConfig struct {
	APIKey *string     `json:"api_key,omitempty"`
	Models ModelConfig `json:"models"`
}

type SystemInstructionSection struct {
	Content string `json:"content"`
	Title   string `json:"title"`
}

type CustomSystemInstructions struct {
	Text string `json:"text"`
}

type TemplatedSystemInstructions struct {
	Identity *string                    `json:"identity,omitempty"`
	Sections []SystemInstructionSection `json:"sections,omitempty"`
}

// BuiltinTools
type BuiltinTool string

const (
	BuiltinToolListDir       BuiltinTool = "list_directory"
	BuiltinToolSearchDir     BuiltinTool = "search_directory"
	BuiltinToolFindFile      BuiltinTool = "find_file"
	BuiltinToolViewFile      BuiltinTool = "view_file"
	BuiltinToolCreateFile    BuiltinTool = "create_file"
	BuiltinToolEditFile      BuiltinTool = "edit_file"
	BuiltinToolRunCommand    BuiltinTool = "run_command"
	BuiltinToolAskQuestion   BuiltinTool = "ask_question"
	BuiltinToolStartSubagent BuiltinTool = "start_subagent"
	BuiltinToolGenerateImage BuiltinTool = "generate_image"
	BuiltinToolFinish        BuiltinTool = "finish"
)

type CapabilitiesConfig struct {
	EnableSubagents       bool          `json:"enable_subagents"`
	EnabledTools          []BuiltinTool `json:"enabled_tools,omitempty"`
	DisabledTools         []BuiltinTool `json:"disabled_tools,omitempty"`
	CompactionThreshold   *int          `json:"compaction_threshold,omitempty"`
	ImageModel            string        `json:"image_model"`
	FinishToolSchemaJSON  *string       `json:"finish_tool_schema_json,omitempty"`
}

type McpStdioServer struct {
	Command string   `json:"command"`
	Type    string   `json:"type"` // always "stdio"
	Args    []string `json:"args,omitempty"`
}

type McpSseServer struct {
	URL     string            `json:"url"`
	Type    string            `json:"type"` // always "sse"
	Headers map[string]string `json:"headers,omitempty"`
}

type McpStreamableHttpServer struct {
	URL              string            `json:"url"`
	Type             string            `json:"type"` // always "http"
	Headers          map[string]string `json:"headers,omitempty"`
	Timeout          float64           `json:"timeout"`
	SSEReadTimeout   float64           `json:"sse_read_timeout"`
	TerminateOnClose bool              `json:"terminate_on_close"`
}

type McpServerConfig interface {
	IsMcpServerConfig()
}

func (McpStdioServer) IsMcpServerConfig() {}
func (McpSseServer) IsMcpServerConfig() {}
func (McpStreamableHttpServer) IsMcpServerConfig() {}

// Tool types

type ToolCall struct {
	Name          string         `json:"name"`
	Args          map[string]any `json:"args,omitempty"`
	ID            *string        `json:"id,omitempty"`
	CanonicalPath *string        `json:"canonical_path,omitempty"`
}

type ToolResult struct {
	Name      string  `json:"name"`
	ID        *string `json:"id,omitempty"`
	Result    any     `json:"result,omitempty"`
	Error     *string `json:"error,omitempty"`
	Exception error   `json:"-"`
}

// Step types

type UsageMetadata struct {
	PromptTokenCount        *int `json:"prompt_token_count,omitempty"`
	CachedContentTokenCount *int `json:"cached_content_token_count,omitempty"`
	CandidatesTokenCount    *int `json:"candidates_token_count,omitempty"`
	ThoughtsTokenCount      *int `json:"thoughts_token_count,omitempty"`
	TotalTokenCount         *int `json:"total_token_count,omitempty"`
}

type StepType string

const (
	StepTypeTextResponse StepType = "TEXT_RESPONSE"
	StepTypeToolCall     StepType = "TOOL_CALL"
	StepTypeSystemMessage StepType = "SYSTEM_MESSAGE"
	StepTypeCompaction    StepType = "COMPACTION"
	StepTypeFinish        StepType = "FINISH"
	StepTypeUnknown       StepType = "UNKNOWN"
)

type StepSource string

const (
	StepSourceSystem  StepSource = "SYSTEM"
	StepSourceUser    StepSource = "USER"
	StepSourceModel   StepSource = "MODEL"
	StepSourceUnknown StepSource = "UNKNOWN"
)

type StepTarget string

const (
	StepTargetUser        StepTarget = "TARGET_USER"
	StepTargetEnvironment StepTarget = "TARGET_ENVIRONMENT"
	StepTargetUnspecified StepTarget = "TARGET_UNSPECIFIED"
	StepTargetUnknown     StepTarget = "UNKNOWN"
)

type StepStatus string

const (
	StepStatusActive         StepStatus = "ACTIVE"
	StepStatusDone           StepStatus = "DONE"
	StepStatusWaitingForUser StepStatus = "WAITING_FOR_USER"
	StepStatusError          StepStatus = "ERROR"
	StepStatusCanceled       StepStatus = "CANCELED"
	StepStatusUnknown        StepStatus = "UNKNOWN"
)

type Step struct {
	ID                 string         `json:"id"`
	StepIndex          int            `json:"step_index"`
	Type               StepType       `json:"type"`
	Source             StepSource     `json:"source"`
	Target             StepTarget     `json:"target"`
	Status             StepStatus     `json:"status"`
	Content            string         `json:"content"`
	ContentDelta       string         `json:"content_delta"`
	Thinking           string         `json:"thinking"`
	ThinkingDelta      string         `json:"thinking_delta"`
	ToolCalls          []ToolCall     `json:"tool_calls"`
	Error              string         `json:"error"`
	IsCompleteResponse *bool          `json:"is_complete_response,omitempty"`
	StructuredOutput   any            `json:"structured_output,omitempty"`
	UsageMetadata      *UsageMetadata `json:"usage_metadata,omitempty"`
}

// Hook types

type HookResult struct {
	Allow   bool   `json:"allow"`
	Message string `json:"message"`
}

type QuestionResponse struct {
	SelectedOptionIDs []string `json:"selected_option_ids,omitempty"`
	FreeformResponse  string   `json:"freeform_response"`
	Skipped           bool     `json:"skipped"`
}

type QuestionHookResult struct {
	Responses []QuestionResponse `json:"responses"`
	Cancelled bool               `json:"cancelled"`
}

type AskQuestionOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type AskQuestionEntry struct {
	Question      string              `json:"question"`
	Options       []AskQuestionOption `json:"options"`
	IsMultiSelect bool                `json:"is_multi_select"`
}

type AskQuestionInteractionSpec struct {
	Questions []AskQuestionEntry `json:"questions"`
}

// Trigger types

type TriggerDelivery string

const (
	TriggerDeliverySendImmediately TriggerDelivery = "send_immediately"
	TriggerDeliveryWaitIdle        TriggerDelivery = "wait_idle"
)

type FileChangeKind string

const (
	FileChangeKindAdded    FileChangeKind = "added"
	FileChangeKindModified FileChangeKind = "modified"
	FileChangeKindDeleted  FileChangeKind = "deleted"
)

type FileChange struct {
	Kind FileChangeKind `json:"kind"`
	Path string         `json:"path"`
}

// Content types

type ContentPrimitive interface {
	IsContentPrimitive()
}

type TextContent string

func (TextContent) IsContentPrimitive() {}

type BaseMedia struct {
	Data        []byte  `json:"data"`
	MimeType    string  `json:"mime_type"`
	Description *string `json:"description,omitempty"`
}

type Image struct {
	BaseMedia
}

func (Image) IsContentPrimitive() {}

type Document struct {
	BaseMedia
}

func (Document) IsContentPrimitive() {}

type Audio struct {
	BaseMedia
}

func (Audio) IsContentPrimitive() {}

type Video struct {
	BaseMedia
}

func (Video) IsContentPrimitive() {}

func MediaFromFile(path string, description *string) (ContentPrimitive, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		return nil, errors.New("could not infer a valid MIME type for extension")
	}

	base := BaseMedia{
		Data:        data,
		MimeType:    mimeType,
		Description: description,
	}

	switch {
	case mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/webp" || mimeType == "image/bmp":
		return Image{base}, nil
	case mimeType == "application/pdf" || mimeType == "text/plain" || mimeType == "application/json" || mimeType == "text/html":
		return Document{base}, nil
	case mimeType == "audio/wav" || mimeType == "audio/mp3":
		return Audio{base}, nil
	case mimeType == "video/mp4" || mimeType == "video/mpeg":
		return Video{base}, nil
	default:
		return nil, errors.New("unsupported MIME type")
	}
}
