package claudecode

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestPermissionManager tests permission manager implementation.
func TestPermissionManager(t *testing.T) {
	t.Run("NewPermissionManager creates valid manager", func(t *testing.T) {
		pm := NewPermissionManager()
		if pm == nil {
			t.Fatal("NewPermissionManager returned nil")
		}
		if pm.HasCallback() {
			t.Error("Expected HasCallback to return false for new manager")
		}
	})

	t.Run("SetPermissionCallback and HasCallback", func(t *testing.T) {
		pm := NewPermissionManager()
		callback := func(ctx context.Context, toolName string, input map[string]any, permContext ToolPermissionContext) (PermissionResult, error) {
			return NewPermissionResultAllow(), nil
		}

		pm.SetPermissionCallback(callback)
		if !pm.HasCallback() {
			t.Error("Expected HasCallback to return true after setting callback")
		}
	})

	t.Run("CheckPermission allows when no callback set", func(t *testing.T) {
		pm := NewPermissionManager()
		ctx := context.Background()

		result, err := pm.CheckPermission(ctx, "test_tool", map[string]any{"arg": "value"}, ToolPermissionContext{})
		if err != nil {
			t.Fatalf("CheckPermission failed: %v", err)
		}
		if result.Behavior() != PermissionBehaviorAllow {
			t.Errorf("Expected Allow behavior, got: %s", result.Behavior())
		}
	})

	t.Run("CheckPermission calls callback and returns result", func(t *testing.T) {
		pm := NewPermissionManager()
		callbackCalled := false

		pm.SetPermissionCallback(func(ctx context.Context, toolName string, input map[string]any, permContext ToolPermissionContext) (PermissionResult, error) {
			callbackCalled = true
			if toolName != "test_tool" {
				t.Errorf("Expected toolName 'test_tool', got: %s", toolName)
			}
			if input["arg"] != "value" {
				t.Errorf("Expected input arg 'value', got: %v", input["arg"])
			}
			return NewPermissionResultDeny("Tool not allowed"), nil
		})

		ctx := context.Background()
		result, err := pm.CheckPermission(ctx, "test_tool", map[string]any{"arg": "value"}, ToolPermissionContext{})

		if err != nil {
			t.Fatalf("CheckPermission failed: %v", err)
		}
		if !callbackCalled {
			t.Error("Callback was not called")
		}
		if result.Behavior() != PermissionBehaviorDeny {
			t.Errorf("Expected Deny behavior, got: %s", result.Behavior())
		}
		if result.Message() != "Tool not allowed" {
			t.Errorf("Expected message 'Tool not allowed', got: %s", result.Message())
		}
	})

	t.Run("CheckPermission handles callback panic", func(t *testing.T) {
		pm := NewPermissionManager()

		pm.SetPermissionCallback(func(ctx context.Context, toolName string, input map[string]any, permContext ToolPermissionContext) (PermissionResult, error) {
			panic("test panic")
		})

		ctx := context.Background()
		result, err := pm.CheckPermission(ctx, "panic_tool", map[string]any{}, ToolPermissionContext{})

		if err == nil {
			t.Error("Expected error from panic, got nil")
		}
		if result.Behavior() != PermissionBehaviorDeny {
			t.Errorf("Expected Deny behavior on panic, got: %s", result.Behavior())
		}
	})

	t.Run("Concurrent CheckPermission calls", func(t *testing.T) {
		pm := NewPermissionManager()
		callCount := 0
		var mu sync.Mutex

		pm.SetPermissionCallback(func(ctx context.Context, toolName string, input map[string]any, permContext ToolPermissionContext) (PermissionResult, error) {
			mu.Lock()
			callCount++
			mu.Unlock()
			time.Sleep(10 * time.Millisecond) // Small delay to test concurrency
			return NewPermissionResultAllow(), nil
		})

		const numGoroutines = 10
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				ctx := context.Background()
				_, err := pm.CheckPermission(ctx, "tool", map[string]any{"id": id}, ToolPermissionContext{})
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent CheckPermission failed: %v", err)
		}

		mu.Lock()
		finalCallCount := callCount
		mu.Unlock()

		if finalCallCount != numGoroutines {
			t.Errorf("Expected %d callback calls, got: %d", numGoroutines, finalCallCount)
		}
	})
}

// TestPermissionResult tests permission result creation and behavior.
func TestPermissionResult(t *testing.T) {
	t.Run("NewPermissionResultAllow creates allow result", func(t *testing.T) {
		result := NewPermissionResultAllow()
		if result.Behavior() != PermissionBehaviorAllow {
			t.Errorf("Expected Allow behavior, got: %s", result.Behavior())
		}
		if result.Message() != "" {
			t.Errorf("Expected empty message, got: %s", result.Message())
		}
		if result.UpdatedInput() != nil {
			t.Errorf("Expected nil UpdatedInput, got: %v", result.UpdatedInput())
		}
		if len(result.UpdatedPermissions()) != 0 {
			t.Errorf("Expected empty UpdatedPermissions, got: %v", result.UpdatedPermissions())
		}
		if result.ShouldInterrupt() {
			t.Error("Expected ShouldInterrupt to be false for Allow result")
		}
	})

	t.Run("NewPermissionResultDeny creates deny result with message", func(t *testing.T) {
		message := "Access denied"
		result := NewPermissionResultDeny(message)
		if result.Behavior() != PermissionBehaviorDeny {
			t.Errorf("Expected Deny behavior, got: %s", result.Behavior())
		}
		if result.Message() != message {
			t.Errorf("Expected message '%s', got: %s", message, result.Message())
		}
		if result.UpdatedInput() != nil {
			t.Errorf("Expected nil UpdatedInput, got: %v", result.UpdatedInput())
		}
		if len(result.UpdatedPermissions()) != 0 {
			t.Errorf("Expected empty UpdatedPermissions, got: %v", result.UpdatedPermissions())
		}
		if result.ShouldInterrupt() {
			t.Error("Expected ShouldInterrupt to be false for default deny result")
		}
	})

	t.Run("PermissionResultDeny WithInterrupt", func(t *testing.T) {
		result := NewPermissionResultDeny("Interrupt").WithInterrupt()
		if !result.ShouldInterrupt() {
			t.Error("Expected ShouldInterrupt to be true after WithInterrupt")
		}
	})

	t.Run("PermissionResultAllow WithInput and WithPermissions", func(t *testing.T) {
		input := map[string]any{"modified": true}
		behavior := PermissionBehaviorAllow
		permissions := []PermissionUpdate{
			{Type: PermissionUpdateDestinationUserSettings, Behavior: &behavior},
		}

		result := NewPermissionResultAllow().WithInput(input).WithPermissions(permissions)

		if result.UpdatedInput()["modified"] != true {
			t.Error("WithInput did not set updated input")
		}
		if len(result.UpdatedPermissions()) != 1 {
			t.Error("WithPermissions did not set updated permissions")
		}
	})
}

// TestToolPermissionContext tests tool permission context.
func TestToolPermissionContext(t *testing.T) {
	t.Run("Empty context works", func(t *testing.T) {
		context := ToolPermissionContext{}

		if context.Signal != nil {
			t.Errorf("Expected nil Signal, got: %v", context.Signal)
		}
		if len(context.Suggestions) != 0 {
			t.Errorf("Expected empty Suggestions, got: %v", context.Suggestions)
		}
	})

	t.Run("Context with signal and suggestions", func(t *testing.T) {
		signal := "test signal"
		suggestions := []PermissionUpdate{
			{Type: PermissionUpdateDestinationUserSettings},
		}

		context := ToolPermissionContext{
			Signal:      signal,
			Suggestions: suggestions,
		}

		if context.Signal != signal {
			t.Errorf("Expected Signal '%v', got: %v", signal, context.Signal)
		}
		if len(context.Suggestions) != 1 {
			t.Errorf("Expected 1 suggestion, got: %d", len(context.Suggestions))
		}
		if context.Suggestions[0].Type != PermissionUpdateDestinationUserSettings {
			t.Errorf("Expected UserSettings suggestion, got: %v", context.Suggestions[0].Type)
		}
	})
}

// TestPermissionUpdate tests permission update structure.
func TestPermissionUpdate(t *testing.T) {
	t.Run("PermissionUpdate with behavior", func(t *testing.T) {
		behavior := PermissionBehaviorAllow
		update := PermissionUpdate{
			Type:     PermissionUpdateDestinationProjectSettings,
			Behavior: &behavior,
		}

		if update.Type != PermissionUpdateDestinationProjectSettings {
			t.Errorf("Expected ProjectSettings type, got: %v", update.Type)
		}
		if update.Behavior == nil {
			t.Error("Expected non-nil Behavior")
		}
		if *update.Behavior != PermissionBehaviorAllow {
			t.Errorf("Expected Allow behavior, got: %v", *update.Behavior)
		}
	})

	t.Run("PermissionUpdate with rules", func(t *testing.T) {
		rules := []PermissionRuleValue{
			{ToolName: "test_tool", RuleContent: "allow"},
		}
		update := PermissionUpdate{
			Type:  PermissionUpdateDestinationLocalSettings,
			Rules: rules,
		}

		if len(update.Rules) != 1 {
			t.Errorf("Expected 1 rule, got: %d", len(update.Rules))
		}
		if update.Rules[0].ToolName != "test_tool" {
			t.Errorf("Expected tool_name 'test_tool', got: %s", update.Rules[0].ToolName)
		}
		if update.Rules[0].RuleContent != "allow" {
			t.Errorf("Expected rule_content 'allow', got: %s", update.Rules[0].RuleContent)
		}
	})
}