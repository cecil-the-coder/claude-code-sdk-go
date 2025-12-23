package shared

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"
)

// HookEventName represents the type of hook event.
type HookEventName string

const (
	// HookEventPreToolUse fires before tool execution (can modify input, control permissions)
	HookEventPreToolUse HookEventName = "pre_tool_use"

	// HookEventPostToolUse fires after tool execution (add context to response)
	HookEventPostToolUse HookEventName = "post_tool_use"

	// HookEventUserPromptSubmit fires when user submits a prompt
	HookEventUserPromptSubmit HookEventName = "user_prompt_submit"

	// HookEventStop fires when execution stops
	HookEventStop HookEventName = "stop"

	// HookEventSubagentStop fires when a subagent completes
	HookEventSubagentStop HookEventName = "subagent_stop"

	// HookEventPreCompact fires before compacting conversation history
	HookEventPreCompact HookEventName = "pre_compact"
)

// DefaultHookTimeout is the default timeout for hook execution in seconds.
const DefaultHookTimeout = 60.0

// HookCallback is the function signature for hook callbacks.
// It receives:
//   - ctx: Context for cancellation and timeouts
//   - input: Hook-specific input data (discriminated union)
//   - toolUseID: Optional tool use ID (for tool-related hooks)
//   - context: Additional context data
//
// Returns:
//   - map[string]any: Hook output (control, decision, hook-specific fields)
//   - error: Error if hook execution fails
type HookCallback func(ctx context.Context, input HookInput, toolUseID *string, context HookContext) (map[string]any, error)

// HookMatcher defines pattern matching and callbacks for a hook event.
type HookMatcher struct {
	// Matcher is a regex pattern to match tool names (nil matches all)
	// Examples: "Bash", "Bash|Edit|Write", "^(Bash|Edit)$"
	Matcher *string `json:"matcher,omitempty"`

	// Hooks is the list of callbacks to execute for this matcher
	Hooks []HookCallback `json:"-"`

	// Timeout is the execution timeout in seconds (nil uses default 60s)
	Timeout *float64 `json:"timeout,omitempty"`
}

// HookInput represents the input data for hook callbacks.
// This is a discriminated union based on HookEventName.
type HookInput struct {
	// Common fields (all hook types)
	HookEventName  HookEventName `json:"hook_event_name"`
	SessionID      string        `json:"session_id"`
	TranscriptPath *string       `json:"transcript_path,omitempty"`
	Cwd            *string       `json:"cwd,omitempty"`
	PermissionMode *string       `json:"permission_mode,omitempty"`

	// PreToolUse and PostToolUse fields
	ToolName     *string        `json:"tool_name,omitempty"`
	ToolInput    map[string]any `json:"tool_input,omitempty"`
	ToolResponse *string        `json:"tool_response,omitempty"` // PostToolUse only

	// UserPromptSubmit fields
	Prompt *string `json:"prompt,omitempty"`

	// Stop and SubagentStop fields
	StopHookActive *bool `json:"stop_hook_active,omitempty"`

	// PreCompact fields
	Trigger            *string `json:"trigger,omitempty"`
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// HookContext provides additional context to hook callbacks.
type HookContext struct {
	// AdditionalData contains extra context-specific data
	AdditionalData map[string]any `json:"additional_data,omitempty"`
}

// HookSystem defines the interface for hook management.
type HookSystem interface {
	Register(eventName HookEventName, matcher *HookMatcher) (string, error)
	Unregister(callbackID string) error
	Execute(ctx context.Context, eventName HookEventName, input HookInput, toolUseID *string, hookCtx HookContext) (map[string]any, error)
	HasHooks(eventName HookEventName) bool
	Clear() error
}

// HookRegistry manages hook registration and callback invocation.
type HookRegistry struct {
	mu            sync.RWMutex
	callbacks     map[string]HookCallback
	callbackIDSeq atomic.Uint64
}

// NewHookRegistry creates a new hook registry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		callbacks: make(map[string]HookCallback),
	}
}

// RegisterHooks registers hook callbacks and returns their assigned IDs.
func (r *HookRegistry) RegisterHooks(eventType HookEventName, matcher *HookMatcher) []string {
	if matcher == nil || len(matcher.Hooks) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ids := make([]string, 0, len(matcher.Hooks))
	for _, callback := range matcher.Hooks {
		// Generate unique callback ID
		seq := r.callbackIDSeq.Add(1)
		id := fmt.Sprintf("hook_%s_%d", eventType, seq)

		// Store callback
		r.callbacks[id] = callback
		ids = append(ids, id)
	}

	return ids
}

// GetCallback retrieves a callback by ID.
func (r *HookRegistry) GetCallback(id string) HookCallback {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.callbacks[id]
}

// InvokeCallback invokes a callback by ID.
func (r *HookRegistry) InvokeCallback(ctx context.Context, id string, input HookInput, toolUseID *string, hookCtx HookContext) (map[string]any, error) {
	r.mu.RLock()
	callback := r.callbacks[id]
	r.mu.RUnlock()

	if callback == nil {
		return nil, fmt.Errorf("hook callback not found: %s", id)
	}

	return callback(ctx, input, toolUseID, hookCtx)
}

// MatchesPattern checks if a tool name matches a hook pattern.
func MatchesPattern(pattern *string, toolName string) bool {
	if pattern == nil {
		return true // nil pattern matches all
	}

	// Try regex match
	re, err := regexp.Compile(*pattern)
	if err != nil {
		// If not valid regex, try exact match
		return *pattern == toolName
	}

	return re.MatchString(toolName)
}

// Register implements HookSystem interface
func (r *HookRegistry) Register(eventName HookEventName, matcher *HookMatcher) (string, error) {
	if matcher == nil || len(matcher.Hooks) == 0 {
		return "", nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// For simplicity, register the first hook and return its ID
	// In a full implementation, you might want to handle multiple hooks
	callbackID := fmt.Sprintf("hook_%d", r.callbackIDSeq.Add(1))
	r.callbacks[callbackID] = matcher.Hooks[0]

	return callbackID, nil
}

// Unregister implements HookSystem interface
func (r *HookRegistry) Unregister(callbackID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.callbacks, callbackID)
	return nil
}

// Execute implements HookSystem interface
func (r *HookRegistry) Execute(ctx context.Context, eventName HookEventName, input HookInput, toolUseID *string, hookCtx HookContext) (map[string]any, error) {
	// This is a simplified implementation
	// In a full implementation, you would filter by eventName and matching patterns
	r.mu.RLock()
	defer r.mu.RUnlock()

	for id, callback := range r.callbacks {
		// Execute the callback
		result, err := callback(ctx, input, toolUseID, hookCtx)
		if err != nil {
			return nil, fmt.Errorf("hook %s failed: %w", id, err)
		}
		if result != nil {
			return result, nil
		}
	}

	return map[string]any{"continue": true}, nil
}

// HasHooks implements HookSystem interface
func (r *HookRegistry) HasHooks(eventName HookEventName) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.callbacks) > 0
}

// Clear implements HookSystem interface
func (r *HookRegistry) Clear() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.callbacks = make(map[string]HookCallback)
	return nil
}

// NewHookSystem creates a new HookRegistry instance.
func NewHookSystem() HookSystem {
	return &HookRegistry{
		callbacks: make(map[string]HookCallback),
	}
}
