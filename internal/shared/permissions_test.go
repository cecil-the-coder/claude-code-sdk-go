package shared

import (
	"encoding/json"
	"testing"
)

// Test functions first (primary purpose)

func TestPermissionResultAllow_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		result   *PermissionResultAllow
		expected string
	}{
		{
			name:     "empty allow result",
			result:   &PermissionResultAllow{},
			expected: `{"behavior":"allow"}`,
		},
		{
			name: "allow with updated input",
			result: &PermissionResultAllow{
				UpdatedInput: map[string]interface{}{
					"file_path": "/tmp/safe.txt",
				},
			},
			expected: `{"behavior":"allow","updated_input":{"file_path":"/tmp/safe.txt"}}`,
		},
		{
			name: "allow with updated permissions",
			result: &PermissionResultAllow{
				UpdatedPermissions: []PermissionUpdate{
					{
						Type: PermissionUpdateTypeAddRules,
						Rules: []PermissionRuleValue{
							{ToolName: "Write"},
						},
					},
				},
			},
		},
		{
			name: "allow with both updated input and permissions",
			result: &PermissionResultAllow{
				UpdatedInput: map[string]interface{}{
					"command": "ls -la",
				},
				UpdatedPermissions: []PermissionUpdate{
					{
						Type: PermissionUpdateTypeSetMode,
						Mode: ptr(PermissionModeAcceptEdits),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			// Verify it's valid JSON
			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			// Check behavior field
			if parsed["behavior"] != "allow" {
				t.Errorf("expected behavior=allow, got %v", parsed["behavior"])
			}

			// For simple cases, check exact match
			if tt.expected != "" {
				if string(data) != tt.expected {
					t.Errorf("expected JSON %s, got %s", tt.expected, string(data))
				}
			}
		})
	}
}

func TestPermissionResultDeny_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		result   *PermissionResultDeny
		expected string
	}{
		{
			name:     "empty deny result",
			result:   &PermissionResultDeny{},
			expected: `{"behavior":"deny"}`,
		},
		{
			name: "deny with message",
			result: &PermissionResultDeny{
				Message: "Dangerous command",
			},
			expected: `{"behavior":"deny","message":"Dangerous command"}`,
		},
		{
			name: "deny with interrupt",
			result: &PermissionResultDeny{
				Interrupt: true,
			},
			expected: `{"behavior":"deny","interrupt":true}`,
		},
		{
			name: "deny with message and interrupt",
			result: &PermissionResultDeny{
				Message:   "Cannot access system files",
				Interrupt: true,
			},
			expected: `{"behavior":"deny","interrupt":true,"message":"Cannot access system files"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			// Verify it's valid JSON
			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			// Check behavior field
			if parsed["behavior"] != "deny" {
				t.Errorf("expected behavior=deny, got %v", parsed["behavior"])
			}

			// Check message if present
			if tt.result.Message != "" {
				if parsed["message"] != tt.result.Message {
					t.Errorf("expected message=%s, got %v", tt.result.Message, parsed["message"])
				}
			}

			// Check interrupt if true
			if tt.result.Interrupt {
				if interrupt, ok := parsed["interrupt"].(bool); !ok || !interrupt {
					t.Errorf("expected interrupt=true, got %v", parsed["interrupt"])
				}
			}
		})
	}
}

func TestPermissionResultBehavior(t *testing.T) {
	t.Run("allow behavior", func(t *testing.T) {
		result := &PermissionResultAllow{}
		if result.Behavior() != "allow" {
			t.Errorf("expected behavior=allow, got %s", result.Behavior())
		}
	})

	t.Run("deny behavior", func(t *testing.T) {
		result := &PermissionResultDeny{}
		if result.Behavior() != "deny" {
			t.Errorf("expected behavior=deny, got %s", result.Behavior())
		}
	})
}

func TestPermissionUpdate_MarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		update PermissionUpdate
	}{
		{
			name: "add rules",
			update: PermissionUpdate{
				Type: PermissionUpdateTypeAddRules,
				Rules: []PermissionRuleValue{
					{ToolName: "Write", RuleContent: ptr("*.txt")},
					{ToolName: "Read"},
				},
				Destination: ptr(PermissionUpdateDestinationSession),
			},
		},
		{
			name: "set mode",
			update: PermissionUpdate{
				Type:        PermissionUpdateTypeSetMode,
				Mode:        ptr(PermissionModeAcceptEdits),
				Destination: ptr(PermissionUpdateDestinationUserSettings),
			},
		},
		{
			name: "add directories",
			update: PermissionUpdate{
				Type:        PermissionUpdateTypeAddDirectories,
				Directories: []string{"/tmp", "/home/user"},
				Destination: ptr(PermissionUpdateDestinationProjectSettings),
			},
		},
		{
			name: "replace rules with behavior",
			update: PermissionUpdate{
				Type: PermissionUpdateTypeReplaceRules,
				Rules: []PermissionRuleValue{
					{ToolName: "Bash"},
				},
				Behavior:    ptr(PermissionBehaviorDeny),
				Destination: ptr(PermissionUpdateDestinationLocalSettings),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.update)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			// Unmarshal to verify structure
			var parsed PermissionUpdate
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			// Verify type
			if parsed.Type != tt.update.Type {
				t.Errorf("expected type=%s, got %s", tt.update.Type, parsed.Type)
			}

			// Verify destination if present
			if tt.update.Destination != nil {
				if parsed.Destination == nil || *parsed.Destination != *tt.update.Destination {
					t.Errorf("destination mismatch")
				}
			}
		})
	}
}

func TestToolPermissionContext_MarshalJSON(t *testing.T) {
	ctx := ToolPermissionContext{
		Suggestions: []PermissionUpdate{
			{
				Type: PermissionUpdateTypeAddRules,
				Rules: []PermissionRuleValue{
					{ToolName: "Write"},
				},
			},
		},
	}

	data, err := json.Marshal(ctx)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ToolPermissionContext
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(parsed.Suggestions) != 1 {
		t.Errorf("expected 1 suggestion, got %d", len(parsed.Suggestions))
	}
}

// Helper functions

func ptr[T any](v T) *T {
	return &v
}
