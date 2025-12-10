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

// Control Protocol Types

// ControlRequest represents any control protocol request type (discriminated union).
type ControlRequest = shared.ControlRequest

// InitializeRequest registers hooks and callbacks with the CLI.
type InitializeRequest = shared.InitializeRequest

// CallbackInfo contains information about a registered callback.
type CallbackInfo = shared.CallbackInfo

// CanUseToolRequest requests permission to use a tool.
type CanUseToolRequest = shared.CanUseToolRequest

// HookCallbackRequest executes a registered hook callback.
type HookCallbackRequest = shared.HookCallbackRequest

// SetPermissionModeRequest changes the permission mode at runtime.
type SetPermissionModeRequest = shared.SetPermissionModeRequest

// SetModelRequest switches the model mid-conversation.
type SetModelRequest = shared.SetModelRequest

// InterruptRequest cancels the current operation.
type InterruptRequest = shared.InterruptRequest

// MCPMessageRequest routes messages to MCP servers.
type MCPMessageRequest = shared.MCPMessageRequest

// SDKControlRequest represents a control protocol request from SDK to CLI.
type SDKControlRequest = shared.SDKControlRequest

// ControlResponseData represents the response data for a control request.
type ControlResponseData = shared.ControlResponseData

// SDKControlResponse represents a control protocol response from CLI to SDK.
type SDKControlResponse = shared.SDKControlResponse

// PendingControlResponses manages pending control protocol requests and responses.
type PendingControlResponses = shared.PendingControlResponses

// Hook System Types

// HookRegistry manages hook registration and callback invocation.
type HookRegistry = shared.HookRegistry

// NewHookRegistry creates a new hook registry.
var NewHookRegistry = shared.NewHookRegistry

// Permission Types

// PermissionBehavior represents the behavior for a permission rule.
type PermissionBehavior = shared.PermissionBehavior

// PermissionUpdateDestination represents where permission updates should be saved.
type PermissionUpdateDestination = shared.PermissionUpdateDestination

// PermissionUpdateType represents the type of permission update.
type PermissionUpdateType = shared.PermissionUpdateType

// PermissionRuleValue represents a permission rule.
type PermissionRuleValue = shared.PermissionRuleValue

// PermissionUpdate represents a permission update configuration.
type PermissionUpdate = shared.PermissionUpdate

// ToolPermissionContext provides context information for tool permission callbacks.
type ToolPermissionContext = shared.ToolPermissionContext

// PermissionResult is the interface for permission callback results.
type PermissionResult = shared.PermissionResult

// PermissionResultAllow represents an allow permission result.
type PermissionResultAllow = shared.PermissionResultAllow

// PermissionResultDeny represents a deny permission result.
type PermissionResultDeny = shared.PermissionResultDeny

// CanUseToolFunc is the callback function type for tool permission decisions.
type CanUseToolFunc = shared.CanUseToolFunc

// Re-export message type constants
const (
	MessageTypeUser               = shared.MessageTypeUser
	MessageTypeAssistant          = shared.MessageTypeAssistant
	MessageTypeSystem             = shared.MessageTypeSystem
	MessageTypeResult             = shared.MessageTypeResult
	MessageTypeSDKControlRequest  = shared.MessageTypeSDKControlRequest
	MessageTypeSDKControlResponse = shared.MessageTypeSDKControlResponse
)

// Re-export control request type constants
const (
	ControlRequestTypeInitialize        = shared.ControlRequestTypeInitialize
	ControlRequestTypeCanUseTool        = shared.ControlRequestTypeCanUseTool
	ControlRequestTypeHookCallback      = shared.ControlRequestTypeHookCallback
	ControlRequestTypeSetPermissionMode = shared.ControlRequestTypeSetPermissionMode
	ControlRequestTypeSetModel          = shared.ControlRequestTypeSetModel
	ControlRequestTypeInterrupt         = shared.ControlRequestTypeInterrupt
	ControlRequestTypeMCPMessage        = shared.ControlRequestTypeMCPMessage
)

// DefaultControlRequestTimeout is the default timeout for control requests (60 seconds).
const DefaultControlRequestTimeout = shared.DefaultControlRequestTimeout

// Re-export permission behavior constants
const (
	PermissionBehaviorAllow = shared.PermissionBehaviorAllow
	PermissionBehaviorDeny  = shared.PermissionBehaviorDeny
	PermissionBehaviorAsk   = shared.PermissionBehaviorAsk
)

// Re-export permission update destination constants
const (
	PermissionUpdateDestinationUserSettings    = shared.PermissionUpdateDestinationUserSettings
	PermissionUpdateDestinationProjectSettings = shared.PermissionUpdateDestinationProjectSettings
	PermissionUpdateDestinationLocalSettings   = shared.PermissionUpdateDestinationLocalSettings
	PermissionUpdateDestinationSession         = shared.PermissionUpdateDestinationSession
)

// Re-export permission update type constants
const (
	PermissionUpdateTypeAddRules          = shared.PermissionUpdateTypeAddRules
	PermissionUpdateTypeReplaceRules      = shared.PermissionUpdateTypeReplaceRules
	PermissionUpdateTypeRemoveRules       = shared.PermissionUpdateTypeRemoveRules
	PermissionUpdateTypeSetMode           = shared.PermissionUpdateTypeSetMode
	PermissionUpdateTypeAddDirectories    = shared.PermissionUpdateTypeAddDirectories
	PermissionUpdateTypeRemoveDirectories = shared.PermissionUpdateTypeRemoveDirectories
)

// Re-export content block type constants
const (
	ContentBlockTypeText       = shared.ContentBlockTypeText
	ContentBlockTypeThinking   = shared.ContentBlockTypeThinking
	ContentBlockTypeToolUse    = shared.ContentBlockTypeToolUse
	ContentBlockTypeToolResult = shared.ContentBlockTypeToolResult
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

// GenerateRequestID generates a unique request ID in the format: req_{counter}_{8-char-hex-random}
var GenerateRequestID = shared.GenerateRequestID

// NewPendingControlResponses creates a new PendingControlResponses manager.
var NewPendingControlResponses = shared.NewPendingControlResponses
