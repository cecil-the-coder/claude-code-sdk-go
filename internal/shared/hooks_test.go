package shared

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"
)

// TestHookEventTypes tests all 6 hook event type constants
func TestHookEventTypes(t *testing.T) {
	tests := []struct {
		name     string
		hookType HookEventName
		expected string
	}{
		{"pre_tool_use", HookEventPreToolUse, "pre_tool_use"},
		{"post_tool_use", HookEventPostToolUse, "post_tool_use"},
		{"user_prompt_submit", HookEventUserPromptSubmit, "user_prompt_submit"},
		{"stop", HookEventStop, "stop"},
		{"subagent_stop", HookEventSubagentStop, "subagent_stop"},
		{"pre_compact", HookEventPreCompact, "pre_compact"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.hookType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.hookType))
			}
		})
	}
}

// TestHookCallback tests hook callback function signature
func TestHookCallback(t *testing.T) {
	called := false
	callback := func(ctx context.Context, input HookInput, toolUseID *string, context HookContext) (map[string]any, error) {
		called = true
		// Verify context is usable
		if ctx == nil {
			t.Error("Expected non-nil context")
		}
		// Return valid hook output
		return map[string]any{
			"continue": true,
		}, nil
	}

	ctx := context.Background()
	input := HookInput{
		HookEventName: HookEventPreToolUse,
		SessionID:     "test-session",
		ToolName:      hookStrPtr("Bash"),
		ToolInput:     map[string]any{"command": "ls"},
	}
	toolUseID := "tool-123"
	hookCtx := HookContext{}

	result, err := callback(ctx, input, &toolUseID, hookCtx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !called {
		t.Error("Expected callback to be called")
	}
	if cont, ok := result["continue"].(bool); !ok || !cont {
		t.Error("Expected continue: true in result")
	}
}

// TestHookMatcher tests hook matcher structure and defaults
func TestHookMatcher(t *testing.T) {
	t.Run("basic_matcher", func(t *testing.T) {
		matcher := &HookMatcher{
			Matcher: hookStrPtr("Bash"),
			Hooks:   []HookCallback{},
		}

		if matcher.Matcher == nil || *matcher.Matcher != "Bash" {
			t.Error("Expected matcher pattern to be 'Bash'")
		}
		if matcher.Hooks == nil {
			t.Error("Expected hooks slice to be initialized")
		}
		if matcher.Timeout != nil {
			t.Error("Expected timeout to be nil (use default)")
		}
	})

	t.Run("matcher_with_timeout", func(t *testing.T) {
		timeout := 30.0
		matcher := &HookMatcher{
			Matcher: hookStrPtr("Edit|Write"),
			Hooks:   []HookCallback{},
			Timeout: &timeout,
		}

		if matcher.Timeout == nil || *matcher.Timeout != 30.0 {
			t.Error("Expected timeout to be 30.0")
		}
	})

	t.Run("nil_matcher", func(t *testing.T) {
		// nil matcher means match all
		matcher := &HookMatcher{
			Matcher: nil,
			Hooks:   []HookCallback{},
		}

		if matcher.Matcher != nil {
			t.Error("Expected matcher to be nil (match all)")
		}
	})
}

// TestHookInputPreToolUse tests PreToolUse hook input structure
func TestHookInputPreToolUse(t *testing.T) {
	input := HookInput{
		HookEventName:  HookEventPreToolUse,
		SessionID:      "session-123",
		TranscriptPath: hookStrPtr("/path/to/transcript"),
		Cwd:            hookStrPtr("/working/dir"),
		PermissionMode: hookStrPtr("default"),
		ToolName:       hookStrPtr("Bash"),
		ToolInput:      map[string]any{"command": "ls -la"},
	}

	// Validate required fields
	if input.HookEventName != HookEventPreToolUse {
		t.Error("Expected HookEventName to be pre_tool_use")
	}
	if input.SessionID != "session-123" {
		t.Error("Expected SessionID to be session-123")
	}
	if input.ToolName == nil || *input.ToolName != "Bash" {
		t.Error("Expected ToolName to be Bash")
	}
	if input.ToolInput == nil {
		t.Error("Expected ToolInput to be non-nil")
	}
	if cmd, ok := input.ToolInput["command"].(string); !ok || cmd != "ls -la" {
		t.Error("Expected command to be 'ls -la'")
	}
}

// TestHookInputPostToolUse tests PostToolUse hook input structure
func TestHookInputPostToolUse(t *testing.T) {
	input := HookInput{
		HookEventName: HookEventPostToolUse,
		SessionID:     "session-123",
		ToolName:      hookStrPtr("Bash"),
		ToolInput:     map[string]any{"command": "ls"},
		ToolResponse:  hookStrPtr("file1.txt\nfile2.txt"),
	}

	if input.HookEventName != HookEventPostToolUse {
		t.Error("Expected HookEventName to be post_tool_use")
	}
	if input.ToolResponse == nil || *input.ToolResponse != "file1.txt\nfile2.txt" {
		t.Error("Expected ToolResponse to contain file listing")
	}
}

// TestHookInputUserPromptSubmit tests UserPromptSubmit hook input structure
func TestHookInputUserPromptSubmit(t *testing.T) {
	input := HookInput{
		HookEventName: HookEventUserPromptSubmit,
		SessionID:     "session-123",
		Prompt:        hookStrPtr("Help me fix this bug"),
	}

	if input.HookEventName != HookEventUserPromptSubmit {
		t.Error("Expected HookEventName to be user_prompt_submit")
	}
	if input.Prompt == nil || *input.Prompt != "Help me fix this bug" {
		t.Error("Expected Prompt to be 'Help me fix this bug'")
	}
}

// TestHookInputStop tests Stop hook input structure
func TestHookInputStop(t *testing.T) {
	input := HookInput{
		HookEventName:  HookEventStop,
		SessionID:      "session-123",
		StopHookActive: hookBoolPtr(true),
	}

	if input.HookEventName != HookEventStop {
		t.Error("Expected HookEventName to be stop")
	}
	if input.StopHookActive == nil || !*input.StopHookActive {
		t.Error("Expected StopHookActive to be true")
	}
}

// TestHookInputSubagentStop tests SubagentStop hook input structure
func TestHookInputSubagentStop(t *testing.T) {
	input := HookInput{
		HookEventName:  HookEventSubagentStop,
		SessionID:      "session-123",
		StopHookActive: hookBoolPtr(false),
	}

	if input.HookEventName != HookEventSubagentStop {
		t.Error("Expected HookEventName to be subagent_stop")
	}
	if input.StopHookActive == nil || *input.StopHookActive {
		t.Error("Expected StopHookActive to be false")
	}
}

// TestHookInputPreCompact tests PreCompact hook input structure
func TestHookInputPreCompact(t *testing.T) {
	input := HookInput{
		HookEventName:      HookEventPreCompact,
		SessionID:          "session-123",
		Trigger:            hookStrPtr("max_turns"),
		CustomInstructions: hookStrPtr("Keep important context"),
	}

	if input.HookEventName != HookEventPreCompact {
		t.Error("Expected HookEventName to be pre_compact")
	}
	if input.Trigger == nil || *input.Trigger != "max_turns" {
		t.Error("Expected Trigger to be 'max_turns'")
	}
	if input.CustomInstructions == nil {
		t.Error("Expected CustomInstructions to be non-nil")
	}
}

// TestHookContext tests hook context structure
func TestHookContext(t *testing.T) {
	ctx := HookContext{
		AdditionalData: map[string]any{
			"request_id": "req-123",
			"timestamp":  time.Now().Unix(),
		},
	}

	if ctx.AdditionalData == nil {
		t.Error("Expected AdditionalData to be non-nil")
	}
	if reqID, ok := ctx.AdditionalData["request_id"].(string); !ok || reqID != "req-123" {
		t.Error("Expected request_id to be 'req-123'")
	}
}

// TestHookOutputControl tests hook output control fields
func TestHookOutputControl(t *testing.T) {
	tests := []struct {
		name   string
		output map[string]any
		check  func(*testing.T, map[string]any)
	}{
		{
			name: "continue",
			output: map[string]any{
				"continue": true,
			},
			check: func(t *testing.T, output map[string]any) {
				if cont, ok := output["continue"].(bool); !ok || !cont {
					t.Error("Expected continue: true")
				}
			},
		},
		{
			name: "suppress_output",
			output: map[string]any{
				"continue":       true,
				"suppressOutput": true,
			},
			check: func(t *testing.T, output map[string]any) {
				if suppress, ok := output["suppressOutput"].(bool); !ok || !suppress {
					t.Error("Expected suppressOutput: true")
				}
			},
		},
		{
			name: "stop_reason",
			output: map[string]any{
				"continue":   false,
				"stopReason": "user_cancelled",
			},
			check: func(t *testing.T, output map[string]any) {
				if reason, ok := output["stopReason"].(string); !ok || reason != "user_cancelled" {
					t.Error("Expected stopReason: user_cancelled")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.output)
		})
	}
}

// TestHookOutputDecision tests hook output decision fields
func TestHookOutputDecision(t *testing.T) {
	output := map[string]any{
		"decision":      "deny",
		"systemMessage": "Tool usage denied for security reasons",
		"reason":        "Contains dangerous command",
	}

	if decision, ok := output["decision"].(string); !ok || decision != "deny" {
		t.Error("Expected decision: deny")
	}
	if msg, ok := output["systemMessage"].(string); !ok || msg == "" {
		t.Error("Expected non-empty systemMessage")
	}
	if reason, ok := output["reason"].(string); !ok || reason == "" {
		t.Error("Expected non-empty reason")
	}
}

// TestHookOutputAsync tests async hook execution
func TestHookOutputAsync(t *testing.T) {
	output := map[string]any{
		"continue":     true,
		"async":        true,
		"asyncTimeout": 120.0,
	}

	if async, ok := output["async"].(bool); !ok || !async {
		t.Error("Expected async: true")
	}
	if timeout, ok := output["asyncTimeout"].(float64); !ok || timeout != 120.0 {
		t.Error("Expected asyncTimeout: 120.0")
	}
}

// TestHookPatternMatching tests pattern matching for tool names
func TestHookPatternMatching(t *testing.T) {
	tests := []struct {
		name        string
		pattern     *string
		toolName    string
		shouldMatch bool
	}{
		{"exact_match", hookStrPtr("Bash"), "Bash", true},
		{"no_match", hookStrPtr("Bash"), "Edit", false},
		{"pipe_pattern", hookStrPtr("Bash|Edit"), "Bash", true},
		{"pipe_pattern_second", hookStrPtr("Bash|Edit"), "Edit", true},
		{"pipe_no_match", hookStrPtr("Bash|Edit"), "Write", false},
		{"nil_pattern_matches_all", nil, "AnyTool", true},
		{"regex_pattern", hookStrPtr("^(Bash|Edit|Write)$"), "Bash", true},
		{"regex_no_match", hookStrPtr("^(Bash|Edit)$"), "Write", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := matchesPattern(tt.pattern, tt.toolName)
			if matched != tt.shouldMatch {
				t.Errorf("Pattern %v matching %s: expected %v, got %v",
					tt.pattern, tt.toolName, tt.shouldMatch, matched)
			}
		})
	}
}

// TestHookRegistration tests hook registration and callback ID assignment
func TestHookRegistration(t *testing.T) {
	registry := NewHookRegistry()

	callback1 := func(ctx context.Context, input HookInput, toolUseID *string, context HookContext) (map[string]any, error) {
		return map[string]any{"continue": true}, nil
	}
	callback2 := func(ctx context.Context, input HookInput, toolUseID *string, context HookContext) (map[string]any, error) {
		return map[string]any{"continue": true}, nil
	}

	matcher := &HookMatcher{
		Matcher: hookStrPtr("Bash"),
		Hooks:   []HookCallback{callback1, callback2},
	}

	ids := registry.RegisterHooks(HookEventPreToolUse, matcher)

	if len(ids) != 2 {
		t.Errorf("Expected 2 callback IDs, got %d", len(ids))
	}

	// Verify IDs are unique
	if ids[0] == ids[1] {
		t.Error("Expected unique callback IDs")
	}

	// Verify callbacks can be retrieved
	cb1 := registry.GetCallback(ids[0])
	if cb1 == nil {
		t.Error("Expected to retrieve callback by ID")
	}

	cb2 := registry.GetCallback(ids[1])
	if cb2 == nil {
		t.Error("Expected to retrieve callback by ID")
	}
}

// TestHookRegistryInvoke tests invoking registered hooks
func TestHookRegistryInvoke(t *testing.T) {
	registry := NewHookRegistry()

	callCount := 0
	callback := func(ctx context.Context, input HookInput, toolUseID *string, context HookContext) (map[string]any, error) {
		callCount++
		return map[string]any{
			"continue":  true,
			"callCount": callCount,
		}, nil
	}

	matcher := &HookMatcher{
		Matcher: hookStrPtr("Bash"),
		Hooks:   []HookCallback{callback},
	}

	ids := registry.RegisterHooks(HookEventPreToolUse, matcher)
	if len(ids) != 1 {
		t.Fatalf("Expected 1 callback ID, got %d", len(ids))
	}

	// Invoke the callback
	ctx := context.Background()
	input := HookInput{
		HookEventName: HookEventPreToolUse,
		SessionID:     "test-session",
		ToolName:      hookStrPtr("Bash"),
		ToolInput:     map[string]any{"command": "ls"},
	}
	toolUseID := "tool-123"
	hookCtx := HookContext{}

	result, err := registry.InvokeCallback(ctx, ids[0], input, &toolUseID, hookCtx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected callback to be called once, got %d", callCount)
	}
	if count, ok := result["callCount"].(int); !ok || count != 1 {
		t.Error("Expected callCount: 1 in result")
	}
}

// TestHookInputJSONMarshaling tests JSON marshaling/unmarshaling of HookInput
func TestHookInputJSONMarshaling(t *testing.T) {
	tests := []struct {
		name  string
		input HookInput
	}{
		{
			name: "pre_tool_use",
			input: HookInput{
				HookEventName: HookEventPreToolUse,
				SessionID:     "session-123",
				ToolName:      hookStrPtr("Bash"),
				ToolInput:     map[string]any{"command": "ls"},
			},
		},
		{
			name: "post_tool_use",
			input: HookInput{
				HookEventName: HookEventPostToolUse,
				SessionID:     "session-123",
				ToolName:      hookStrPtr("Edit"),
				ToolInput:     map[string]any{"file": "test.go"},
				ToolResponse:  hookStrPtr("File edited successfully"),
			},
		},
		{
			name: "user_prompt_submit",
			input: HookInput{
				HookEventName: HookEventUserPromptSubmit,
				SessionID:     "session-123",
				Prompt:        hookStrPtr("Help me debug this"),
			},
		},
		{
			name: "stop",
			input: HookInput{
				HookEventName:  HookEventStop,
				SessionID:      "session-123",
				StopHookActive: hookBoolPtr(true),
			},
		},
		{
			name: "pre_compact",
			input: HookInput{
				HookEventName: HookEventPreCompact,
				SessionID:     "session-123",
				Trigger:       hookStrPtr("max_turns"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal back
			var unmarshaled HookInput
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify event name matches
			if unmarshaled.HookEventName != tt.input.HookEventName {
				t.Errorf("Expected HookEventName %s, got %s",
					tt.input.HookEventName, unmarshaled.HookEventName)
			}

			// Verify SessionID matches
			if unmarshaled.SessionID != tt.input.SessionID {
				t.Errorf("Expected SessionID %s, got %s",
					tt.input.SessionID, unmarshaled.SessionID)
			}
		})
	}
}

// Helper functions for test file
func hookStrPtr(s string) *string {
	return &s
}

func hookBoolPtr(b bool) *bool {
	return &b
}

// matchesPattern tests if a tool name matches a hook pattern
func matchesPattern(pattern *string, toolName string) bool {
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
