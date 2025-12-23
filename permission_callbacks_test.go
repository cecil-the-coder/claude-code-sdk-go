package claudecode

import (
	"context"
	"testing"
	"time"
)

func TestPermissionManager_AllowByDefault(t *testing.T) {
	t.Helper()

	pm := NewPermissionManager()

	// No callback set - should allow by default
	ctx := context.Background()
	result, err := pm.CheckPermission(ctx, "bash", map[string]any{"command": "echo hello"}, ToolPermissionContext{})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Behavior() != PermissionBehaviorAllow {
		t.Errorf("Expected allow behavior, got: %s", result.Behavior())
	}
}

func TestPermissionManager_WithCallback(t *testing.T) {
	t.Helper()

	pm := NewPermissionManager()

	// Set a callback that denies dangerous commands
	pm.SetPermissionCallback(func(ctx context.Context, toolName string, input map[string]any, permContext ToolPermissionContext) (PermissionResult, error) {
		if toolName == "bash" {
			if cmd, ok := input["command"].(string); ok && cmd == "rm -rf /" {
				return NewPermissionResultDeny("Dangerous command not allowed").WithInterrupt(), nil
			}
		}
		return NewPermissionResultAllow(), nil
	})

	ctx := context.Background()

	// Test safe command
	result, err := pm.CheckPermission(ctx, "bash", map[string]any{"command": "echo hello"}, ToolPermissionContext{})
	if err != nil {
		t.Fatalf("Expected no error for safe command, got: %v", err)
	}
	if result.Behavior() != PermissionBehaviorAllow {
		t.Errorf("Expected allow for safe command, got: %s", result.Behavior())
	}

	// Test dangerous command
	result, err = pm.CheckPermission(ctx, "bash", map[string]any{"command": "rm -rf /"}, ToolPermissionContext{})
	if err != nil {
		t.Fatalf("Expected no error for dangerous command, got: %v", err)
	}
	if result.Behavior() != PermissionBehaviorDeny {
		t.Errorf("Expected deny for dangerous command, got: %s", result.Behavior())
	}
	if !result.ShouldInterrupt() {
		t.Error("Expected interrupt flag for dangerous command")
	}
}

func TestPermissionManager_CallbackTimeout(t *testing.T) {
	t.Helper()

	pm := NewPermissionManager()

	// Set a callback that blocks forever
	pm.SetPermissionCallback(func(ctx context.Context, toolName string, input map[string]any, permContext ToolPermissionContext) (PermissionResult, error) {
		<-ctx.Done() // Block until context is cancelled
		return NewPermissionResultAllow(), nil
	})

	// Use a context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := pm.CheckPermission(ctx, "bash", map[string]any{"command": "echo hello"}, ToolPermissionContext{})

	if err != nil {
		t.Fatalf("Expected no error on timeout, got: %v", err)
	}

	if result.Behavior() != PermissionBehaviorDeny {
		t.Errorf("Expected deny on timeout, got: %s", result.Behavior())
	}
}

func TestPermissionManager_CallbackPanic(t *testing.T) {
	t.Helper()

	pm := NewPermissionManager()

	// Set a callback that panics
	pm.SetPermissionCallback(func(ctx context.Context, toolName string, input map[string]any, permContext ToolPermissionContext) (PermissionResult, error) {
		panic("test panic")
	})

	ctx := context.Background()
	result, err := pm.CheckPermission(ctx, "bash", map[string]any{"command": "echo hello"}, ToolPermissionContext{})

	if err == nil {
		t.Error("Expected error for panicked callback")
	}

	if result.Behavior() != PermissionBehaviorDeny {
		t.Errorf("Expected deny for panicked callback, got: %s", result.Behavior())
	}
}

func TestPermissionResultAllow_WithMethods(t *testing.T) {
	t.Helper()

	result := NewPermissionResultAllow()

	// Test WithInput
	input := map[string]any{"modified": true}
	result = result.WithInput(input)

	if result.UpdatedInput()["modified"] != true {
		t.Error("WithInput failed to set updated input")
	}

	// Test WithPermissions
	update := PermissionUpdate{
		Type:     PermissionUpdateDestinationSessionSettings,
		Behavior: &[]PermissionBehavior{PermissionBehaviorAllow}[0],
	}
	result = result.WithPermissions([]PermissionUpdate{update})

	if len(result.UpdatedPermissions()) != 1 {
		t.Error("WithPermissions failed to set updates")
	}
}

func TestPermissionResultDeny_WithInterrupt(t *testing.T) {
	t.Helper()

	result := NewPermissionResultDeny("Test denial")

	if result.ShouldInterrupt() {
		t.Error("Deny result should not interrupt by default")
	}

	result = result.WithInterrupt()

	if !result.ShouldInterrupt() {
		t.Error("WithInterrupt failed to set interrupt flag")
	}

	if result.Message() != "Test denial" {
		t.Errorf("Expected message 'Test denial', got: %s", result.Message())
	}
}

func TestPermissionManager_HasCallback(t *testing.T) {
	t.Helper()

	pm := NewPermissionManager()

	if pm.HasCallback() {
		t.Error("Should not have callback initially")
	}

	pm.SetPermissionCallback(func(ctx context.Context, toolName string, input map[string]any, permContext ToolPermissionContext) (PermissionResult, error) {
		return NewPermissionResultAllow(), nil
	})

	if !pm.HasCallback() {
		t.Error("Should have callback after setting one")
	}
}