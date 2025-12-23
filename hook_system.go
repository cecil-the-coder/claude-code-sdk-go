package claudecode

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HookEventType represents supported hook event types
type HookEventType string

const (
	HookEventTypePreToolUse       HookEventType = "PreToolUse"
	HookEventTypePostToolUse      HookEventType = "PostToolUse"
	HookEventTypeUserPromptSubmit HookEventType = "UserPromptSubmit"
	HookEventTypeStop              HookEventType = "Stop"
	HookEventTypeSubagentStop      HookEventType = "SubagentStop"
	HookEventTypePreCompact        HookEventType = "PreCompact"
)

// HookBehavior represents hook execution behavior
type HookBehavior string

const (
	HookBehaviorContinue HookBehavior = "continue"
	HookBehaviorStop     HookBehavior = "stop"
)

// BaseHookInput contains fields common to all hook events
type BaseHookInput struct {
	SessionID     string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd           string `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
}

// PreToolUseHookInput represents input data for PreToolUse events
type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName HookEventType `json:"hook_event_name"`
	ToolName     string                 `json:"tool_name"`
	ToolInput    map[string]any           `json:"tool_input"`
}

// PostToolUseHookInput represents input data for PostToolUse events
type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName HookEventType `json:"hook_event_name"`
	ToolName     string                 `json:"tool_name"`
	ToolInput    map[string]any           `json:"tool_input"`
	ToolResponse any                     `json:"tool_response"`
}

// UserPromptSubmitHookInput represents input data for UserPromptSubmit events
type UserPromptSubmitHookInput struct {
	BaseHookInput
	HookEventName HookEventType `json:"hook_event_name"`
	Prompt       string `json:"prompt"`
}

// StopHookInput represents input data for Stop events
type StopHookInput struct {
	BaseHookInput
	HookEventName HookEventType `json:"hook_event_name"`
	StopHookActive bool `json:"stop_hook_active"`
}

// SubagentStopHookInput represents input data for SubagentStop events
type SubagentStopHookInput struct {
	BaseHookInput
	HookEventName HookEventType `json:"hook_event_name"`
	StopHookActive bool `json:"stop_hook_active"`
}

// PreCompactHookInput represents input data for PreCompact events
type PreCompactHookInput struct {
	BaseHookInput
	HookEventName HookEventType `json:"hook_event_name"`
	Trigger      string `json:"trigger"` // "manual" or "auto"
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// HookOutput represents the result of a hook execution
type HookOutput struct {
	Behavior    HookBehavior   `json:"behavior"`
	Message     string         `json:"message,omitempty"`
	Permissions []PermissionUpdate `json:"permissions,omitempty"`
	Context     map[string]any `json:"context,omitempty"`
}

// HookContext provides execution context for hooks
type HookContext struct {
	SessionID     string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd           string `json:"cwd"`
}

// HookCallback defines the function signature for hook callbacks
type HookCallback func(
	ctx context.Context,
	input interface{},
	context HookContext,
) (HookOutput, error)

// HookMatcher defines pattern matching for hook registration
type HookMatcher struct {
	Pattern   string         `json:"pattern"`
	Hooks     []HookCallback `json:"-"`
	Timeout   time.Duration    `json:"timeout,omitempty"`
}

// HookSystem manages hook registration and execution
type HookSystem interface {
	// AddHook registers hooks for a specific pattern
	AddHook(pattern string, hooks ...HookCallback) error

	// RemoveHook removes hooks matching a pattern
	RemoveHook(pattern string) error

	// ExecuteHooks executes hooks for a specific event type
	ExecuteHooks(ctx context.Context, eventType HookEventType, input interface{}) (*HookOutput, error)

	// HasHooks returns true if any hooks are registered
	HasHooks() bool
}

// hookSystem implements HookSystem
type hookSystem struct {
	matchers map[string][]HookCallback
	mu        sync.RWMutex
}

// NewHookSystem creates a new hook system
func NewHookSystem() HookSystem {
	return &hookSystem{
		matchers: make(map[string][]HookCallback),
	}
}

// AddHook registers hooks for a specific pattern
func (hs *hookSystem) AddHook(pattern string, hooks ...HookCallback) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hs.matchers[pattern] = append(hs.matchers[pattern], hooks...)
	return nil
}

// RemoveHook removes hooks matching a pattern
func (hs *hookSystem) RemoveHook(pattern string) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	delete(hs.matchers, pattern)
	return nil
}

// ExecuteHooks executes hooks for a specific event type
func (hs *hookSystem) ExecuteHooks(ctx context.Context, eventType HookEventType, input interface{}) (*HookOutput, error) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	// Find matching hooks for this event type
	var matchingHooks []HookCallback
	for pattern, hooks := range hs.matchers {
		if hs.patternMatches(eventType, pattern) {
			matchingHooks = append(matchingHooks, hooks...)
		}
	}

	if len(matchingHooks) == 0 {
		return &HookOutput{Behavior: HookBehaviorContinue}, nil
	}

	// Execute hooks sequentially with timeout protection
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for _, hook := range matchingHooks {
		output, err := hook(timeoutCtx, input, hs.createHookContext(eventType))
		if err != nil {
			return nil, fmt.Errorf("hook execution failed: %w", err)
		}

		// If hook requests stop, return immediately
		if output.Behavior == HookBehaviorStop {
			return &output, nil
		}

		// Apply permission updates if provided
		if len(output.Permissions) > 0 {
			// TODO: Apply permission updates to permissionManager
		}

		// Apply context modifications if provided
		if len(output.Context) > 0 {
			// TODO: Apply context modifications
		}
	}

	// Default to continue if no hook requested stop
	return &HookOutput{Behavior: HookBehaviorContinue}, nil
}

// HasHooks returns true if any hooks are registered
func (hs *hookSystem) HasHooks() bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	return len(hs.matchers) > 0
}

// createHookContext creates hook execution context
func (hs *hookSystem) createHookContext(eventType HookEventType) HookContext {
	// TODO: Extract actual context from session/client
	return HookContext{
		SessionID:     "unknown", // Extract from client
		TranscriptPath: "unknown", // Extract from client
		Cwd:           "unknown", // Extract from client
	}
}

// patternMatches checks if an event type matches a pattern
func (hs *hookSystem) patternMatches(eventType HookEventType, pattern string) bool {
	switch pattern {
	case "*":
		return true // Wildcard matches all events
	case string(eventType):
		return true // Exact match
	default:
		// TODO: Support more sophisticated pattern matching
		return false
	}
}