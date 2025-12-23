package claudecode

import (
	"context"

	"github.com/severity1/claude-code-sdk-go/internal/shared"
)

// Message represents any message type in the conversation.
type Message = shared.Message

// ContentBlock represents a content block within a message.
type ContentBlock = shared.ContentBlock

// UserMessage represents a message from the user.
type UserMessage = shared.UserMessage

// AssistantMessage represents a message from the assistant.
type AssistantMessage = shared.AssistantMessage

// AssistantMessageError represents error types in assistant messages.
type AssistantMessageError = shared.AssistantMessageError

// SystemMessage represents a system prompt message.
type SystemMessage = shared.SystemMessage

// ResultMessage represents a result or status message.
type ResultMessage = shared.ResultMessage

// TextBlock represents a text content block.
type TextBlock = shared.TextBlock

// ThinkingBlock represents a thinking content block.
type ThinkingBlock = shared.ThinkingBlock

// ToolUseBlock represents a tool usage content block.
type ToolUseBlock = shared.ToolUseBlock

// ToolResultBlock represents a tool result content block.
type ToolResultBlock = shared.ToolResultBlock

// StreamMessage represents a message in the streaming protocol.
type StreamMessage = shared.StreamMessage

// MessageIterator provides iteration over messages.
type MessageIterator = shared.MessageIterator

// StreamValidator tracks tool requests and results to detect incomplete streams.
type StreamValidator = shared.StreamValidator

// StreamIssue represents a validation issue found in the stream.
type StreamIssue = shared.StreamIssue

// StreamStats provides statistics about the message stream.
type StreamStats = shared.StreamStats

// Re-export message type constants
const (
	MessageTypeUser      = shared.MessageTypeUser
	MessageTypeAssistant = shared.MessageTypeAssistant
	MessageTypeSystem    = shared.MessageTypeSystem
	MessageTypeResult    = shared.MessageTypeResult
)

// Re-export content block type constants
const (
	ContentBlockTypeText       = shared.ContentBlockTypeText
	ContentBlockTypeThinking   = shared.ContentBlockTypeThinking
	ContentBlockTypeToolUse    = shared.ContentBlockTypeToolUse
	ContentBlockTypeToolResult = shared.ContentBlockTypeToolResult
)

// Re-export AssistantMessageError constants
const (
	AssistantMessageErrorAuthFailed     = shared.AssistantMessageErrorAuthFailed
	AssistantMessageErrorBilling        = shared.AssistantMessageErrorBilling
	AssistantMessageErrorRateLimit      = shared.AssistantMessageErrorRateLimit
	AssistantMessageErrorInvalidRequest = shared.AssistantMessageErrorInvalidRequest
	AssistantMessageErrorServer         = shared.AssistantMessageErrorServer
	AssistantMessageErrorUnknown        = shared.AssistantMessageErrorUnknown
)

// AgentModel represents the model to use for an agent.
type AgentModel = shared.AgentModel

// AgentDefinition defines a programmatic subagent.
type AgentDefinition = shared.AgentDefinition

// Re-export agent model constants
const (
	AgentModelSonnet  = shared.AgentModelSonnet
	AgentModelOpus    = shared.AgentModelOpus
	AgentModelHaiku   = shared.AgentModelHaiku
	AgentModelInherit = shared.AgentModelInherit
)

// Transport abstracts the communication layer with Claude Code CLI.
// This interface stays in main package because it's used by client code.
type Transport interface {
	Connect(ctx context.Context) error
	SendMessage(ctx context.Context, message StreamMessage) error
	ReceiveMessages(ctx context.Context) (<-chan Message, <-chan error)
	Interrupt(ctx context.Context) error
	Close() error
	GetValidator() *StreamValidator
}

// Hook System Types

// HookEventName represents the type of hook event.
type HookEventName = shared.HookEventName

// HookCallback is the function signature for hook callbacks.
type HookCallback = shared.HookCallback

// HookMatcher defines pattern matching and callbacks for a hook event.
type HookMatcher = shared.HookMatcher

// HookInput represents the input data for hook callbacks.
type HookInput = shared.HookInput

// HookContext provides additional context to hook callbacks.
type HookContext = shared.HookContext

// HookRegistry manages hook registration and callback invocation.
type HookRegistry = shared.HookRegistry

// HookSystem defines the interface for hook management.
type HookSystem = shared.HookSystem

// Re-export HookEventName constants
const (
	HookEventPreToolUse       = shared.HookEventPreToolUse
	HookEventPostToolUse      = shared.HookEventPostToolUse
	HookEventUserPromptSubmit = shared.HookEventUserPromptSubmit
	HookEventStop             = shared.HookEventStop
	HookEventSubagentStop     = shared.HookEventSubagentStop
	HookEventPreCompact       = shared.HookEventPreCompact
)

// Re-export hook system constants
const (
	DefaultHookTimeout = shared.DefaultHookTimeout
)

// NewHookSystem creates a new HookRegistry instance.
func NewHookSystem() HookSystem {
	return shared.NewHookSystem()
}
