package claudecode

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestHookSystem tests hook system implementation.
func TestHookSystem(t *testing.T) {
	t.Run("NewHookSystem creates valid system", func(t *testing.T) {
		hs := NewHookSystem()
		if hs == nil {
			t.Fatal("NewHookSystem returned nil")
		}
		if hs.HasHooks() {
			t.Error("Expected HasHooks to return false for new system")
		}
	})

	t.Run("AddHook and HasHooks", func(t *testing.T) {
		hs := NewHookSystem()

		callback := func(ctx context.Context, input interface{}, hookCtx HookContext) (HookOutput, error) {
			return HookOutput{Behavior: HookBehaviorContinue}, nil
		}

		err := hs.AddHook("test_tool", callback)
		if err != nil {
			t.Fatalf("AddHook failed: %v", err)
		}
		if !hs.HasHooks() {
			t.Error("Expected HasHooks to return true after adding hook")
		}
	})

	t.Run("AddHook with multiple callbacks", func(t *testing.T) {
		hs := NewHookSystem()

		callback1 := func(ctx context.Context, input interface{}, hookCtx HookContext) (HookOutput, error) {
			return HookOutput{Behavior: HookBehaviorContinue}, nil
		}
		callback2 := func(ctx context.Context, input interface{}, hookCtx HookContext) (HookOutput, error) {
			return HookOutput{Behavior: HookBehaviorContinue}, nil
		}

		err := hs.AddHook("test_tool", callback1, callback2)
		if err != nil {
			t.Fatalf("AddHook with multiple callbacks failed: %v", err)
		}
	})

	t.Run("RemoveHook", func(t *testing.T) {
		hs := NewHookSystem()

		callback := func(ctx context.Context, input interface{}, hookCtx HookContext) (HookOutput, error) {
			return HookOutput{Behavior: HookBehaviorContinue}, nil
		}

		hs.AddHook("test_tool", callback)
		if !hs.HasHooks() {
			t.Error("Expected HasHooks to return true after adding hook")
		}

		err := hs.RemoveHook("test_tool")
		if err != nil {
			t.Fatalf("RemoveHook failed: %v", err)
		}
		if hs.HasHooks() {
			t.Error("Expected HasHooks to return false after removing hook")
		}
	})

	t.Run("ExecuteHooks with no matching hooks", func(t *testing.T) {
		hs := NewHookSystem()

		callback := func(ctx context.Context, input interface{}, hookCtx HookContext) (HookOutput, error) {
			return HookOutput{Behavior: HookBehaviorContinue}, nil
		}
		hs.AddHook("other_tool", callback)

		result, err := hs.ExecuteHooks(context.Background(), HookEventTypePreToolUse, PreToolUseHookInput{
			BaseHookInput: BaseHookInput{
				SessionID: "test-session",
			},
			ToolName: "test_tool",
		})

		if err != nil {
			t.Fatalf("ExecuteHooks failed: %v", err)
		}
		if result.Behavior != HookBehaviorContinue {
			t.Errorf("Expected Continue behavior, got: %s", result.Behavior)
		}
	})

	t.Run("ExecuteHooks with matching hooks", func(t *testing.T) {
		hs := NewHookSystem()

		called := false
		callback := func(ctx context.Context, input interface{}, hookCtx HookContext) (HookOutput, error) {
			called = true
			inputTyped := input.(PreToolUseHookInput)
			if inputTyped.ToolName != "test_tool" {
				t.Errorf("Expected ToolName 'test_tool', got: %s", inputTyped.ToolName)
			}
			return HookOutput{Behavior: HookBehaviorContinue}, nil
		}
		hs.AddHook("test_tool", callback)

		result, err := hs.ExecuteHooks(context.Background(), HookEventTypePreToolUse, PreToolUseHookInput{
			BaseHookInput: BaseHookInput{
				SessionID: "test-session",
			},
			ToolName: "test_tool",
		})

		if err != nil {
			t.Fatalf("ExecuteHooks failed: %v", err)
		}
		if !called {
			t.Error("Hook was not called")
		}
		if result.Behavior != HookBehaviorContinue {
			t.Errorf("Expected Continue behavior, got: %s", result.Behavior)
		}
	})

	t.Run("ExecuteHooks with stop behavior", func(t *testing.T) {
		hs := NewHookSystem()

		callback := func(ctx context.Context, input interface{}, hookCtx HookContext) (HookOutput, error) {
			return HookOutput{
				Behavior: HookBehaviorStop,
				Message:  "Stopped by hook",
			}, nil
		}
		hs.AddHook("test_tool", callback)

		result, err := hs.ExecuteHooks(context.Background(), HookEventTypePreToolUse, PreToolUseHookInput{
			BaseHookInput: BaseHookInput{
				SessionID: "test-session",
			},
			ToolName: "test_tool",
		})

		if err != nil {
			t.Fatalf("ExecuteHooks failed: %v", err)
		}
		if result.Behavior != HookBehaviorStop {
			t.Errorf("Expected Stop behavior, got: %s", result.Behavior)
		}
		if result.Message != "Stopped by hook" {
			t.Errorf("Expected message 'Stopped by hook', got: %s", result.Message)
		}
	})

	t.Run("ExecuteHooks with timeout", func(t *testing.T) {
		hs := NewHookSystem()

		callback := func(ctx context.Context, input interface{}, hookCtx HookContext) (HookOutput, error) {
			<-ctx.Done() // Wait for context cancellation
			return HookOutput{Behavior: HookBehaviorContinue}, nil
		}
		hs.AddHook("test_tool", callback)

		// Use very short timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		start := time.Now()
		result, err := hs.ExecuteHooks(ctx, HookEventTypePreToolUse, PreToolUseHookInput{
			BaseHookInput: BaseHookInput{
				SessionID: "test-session",
			},
			ToolName: "test_tool",
		})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("ExecuteHooks failed: %v", err)
		}
		if duration > 5*time.Second {
			t.Errorf("ExecuteHooks took too long: %v", duration)
		}
		// Should return default continue on timeout
		if result.Behavior != HookBehaviorContinue {
			t.Errorf("Expected Continue behavior on timeout, got: %s", result.Behavior)
		}
	})

	t.Run("ExecuteHooks with callback panic", func(t *testing.T) {
		hs := NewHookSystem()

		callback := func(ctx context.Context, input interface{}, hookCtx HookContext) (HookOutput, error) {
			panic("test panic")
		}
		hs.AddHook("test_tool", callback)

		result, err := hs.ExecuteHooks(context.Background(), HookEventTypePreToolUse, PreToolUseHookInput{
			BaseHookInput: BaseHookInput{
				SessionID: "test-session",
			},
			ToolName: "test_tool",
		})

		if err != nil {
			t.Fatalf("ExecuteHooks failed: %v", err)
		}
		// Should return default continue on panic
		if result.Behavior != HookBehaviorContinue {
			t.Errorf("Expected Continue behavior on panic, got: %s", result.Behavior)
		}
	})

	t.Run("Concurrent ExecuteHooks calls", func(t *testing.T) {
		hs := NewHookSystem()
		callCount := 0
		var mu sync.Mutex

		callback := func(ctx context.Context, input interface{}, hookCtx HookContext) (HookOutput, error) {
			mu.Lock()
			callCount++
			mu.Unlock()
			time.Sleep(10 * time.Millisecond) // Small delay to test concurrency
			return HookOutput{Behavior: HookBehaviorContinue}, nil
		}
		hs.AddHook("test_tool", callback)

		const numGoroutines = 10
		var wg sync.WaitGroup
		results := make(chan *HookOutput, numGoroutines)
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				ctx := context.Background()
				result, err := hs.ExecuteHooks(ctx, HookEventTypePreToolUse, PreToolUseHookInput{
					BaseHookInput: BaseHookInput{
						SessionID: "test-session",
					},
					ToolName: "test_tool",
				})
				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}(i)
		}

		wg.Wait()
		close(results)
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent ExecuteHooks failed: %v", err)
		}

		mu.Lock()
		finalCallCount := callCount
		mu.Unlock()

		if finalCallCount != numGoroutines {
			t.Errorf("Expected %d callback calls, got: %d", numGoroutines, finalCallCount)
		}

		// Check all results have continue behavior
		for result := range results {
			if result.Behavior != HookBehaviorContinue {
				t.Errorf("Expected Continue behavior, got: %s", result.Behavior)
			}
		}
	})
}

// TestHookInputTypes tests hook input data structures.
func TestHookInputTypes(t *testing.T) {
	t.Run("PreToolUseHookInput", func(t *testing.T) {
		input := PreToolUseHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:     "session123",
				TranscriptPath: "/path/to/transcript",
				Cwd:           "/working/dir",
				PermissionMode: func() *string { s := "accept"; return &s }(),
			},
			HookEventName: HookEventTypePreToolUse,
			ToolName:     "test_tool",
			ToolInput:    map[string]any{"arg": "value"},
		}

		if input.SessionID != "session123" {
			t.Errorf("Expected SessionID 'session123', got: %s", input.SessionID)
		}
		if input.TranscriptPath != "/path/to/transcript" {
			t.Errorf("Expected TranscriptPath '/path/to/transcript', got: %s", input.TranscriptPath)
		}
		if input.Cwd != "/working/dir" {
			t.Errorf("Expected Cwd '/working/dir', got: %s", input.Cwd)
		}
		if input.PermissionMode == nil || *input.PermissionMode != "accept" {
			t.Errorf("Expected PermissionMode 'accept', got: %v", input.PermissionMode)
		}
		if input.HookEventName != HookEventTypePreToolUse {
			t.Errorf("Expected HookEventName PreToolUse, got: %s", input.HookEventName)
		}
		if input.ToolName != "test_tool" {
			t.Errorf("Expected ToolName 'test_tool', got: %s", input.ToolName)
		}
		if input.ToolInput["arg"] != "value" {
			t.Errorf("Expected ToolInput arg 'value', got: %v", input.ToolInput["arg"])
		}
	})

	t.Run("PostToolUseHookInput", func(t *testing.T) {
		response := map[string]any{"result": "success"}
		input := PostToolUseHookInput{
			BaseHookInput: BaseHookInput{
				SessionID: "session123",
			},
			HookEventName: HookEventTypePostToolUse,
			ToolName:     "test_tool",
			ToolInput:    map[string]any{"arg": "value"},
			ToolResponse: response,
		}

		// Convert ToolResponse to map for comparison
		responseMap, ok := input.ToolResponse.(map[string]any)
		if !ok {
			t.Errorf("Expected ToolResponse to be map, got: %T", input.ToolResponse)
		}
		if len(responseMap) != len(response) {
			t.Errorf("Expected ToolResponse length %d, got: %d", len(response), len(responseMap))
		}
		// Check key-value pairs match
		for k, v := range response {
			if responseMap[k] != v {
				t.Errorf("Expected ToolResponse[%s] = %v, got: %v", k, v, responseMap[k])
			}
		}
	})

	t.Run("UserPromptSubmitHookInput", func(t *testing.T) {
		input := UserPromptSubmitHookInput{
			BaseHookInput: BaseHookInput{
				SessionID: "session123",
			},
			HookEventName: HookEventTypeUserPromptSubmit,
			Prompt:       "Test prompt message",
		}

		if input.Prompt != "Test prompt message" {
			t.Errorf("Expected Prompt 'Test prompt message', got: %s", input.Prompt)
		}
	})

	t.Run("StopHookInput", func(t *testing.T) {
		input := StopHookInput{
			BaseHookInput: BaseHookInput{
				SessionID: "session123",
			},
			HookEventName:  HookEventTypeStop,
			StopHookActive: true,
		}

		if !input.StopHookActive {
			t.Error("Expected StopHookActive to be true")
		}
	})

	t.Run("PreCompactHookInput", func(t *testing.T) {
		instructions := "Custom compact instructions"
		input := PreCompactHookInput{
			BaseHookInput: BaseHookInput{
				SessionID: "session123",
			},
			HookEventName:      HookEventTypePreCompact,
			Trigger:           "manual",
			CustomInstructions: &instructions,
		}

		if input.Trigger != "manual" {
			t.Errorf("Expected Trigger 'manual', got: %s", input.Trigger)
		}
		if input.CustomInstructions == nil || *input.CustomInstructions != instructions {
			t.Errorf("Expected CustomInstructions '%s', got: %v", instructions, input.CustomInstructions)
		}
	})
}

// TestHookOutput tests hook output structure.
func TestHookOutput(t *testing.T) {
	t.Run("HookOutput with continue behavior", func(t *testing.T) {
		permissions := []PermissionUpdate{
			{Type: PermissionUpdateDestinationUserSettings},
		}
		context := map[string]any{"key": "value"}
		output := HookOutput{
			Behavior:    HookBehaviorContinue,
			Message:     "Continue message",
			Permissions: permissions,
			Context:     context,
		}

		if output.Behavior != HookBehaviorContinue {
			t.Errorf("Expected Continue behavior, got: %s", output.Behavior)
		}
		if output.Message != "Continue message" {
			t.Errorf("Expected message 'Continue message', got: %s", output.Message)
		}
		if len(output.Permissions) != 1 {
			t.Errorf("Expected 1 permission, got: %d", len(output.Permissions))
		}
		if output.Permissions[0].Type != PermissionUpdateDestinationUserSettings {
			t.Errorf("Expected UserSettings permission, got: %v", output.Permissions[0].Type)
		}
		if output.Context["key"] != "value" {
			t.Errorf("Expected Context key 'value', got: %v", output.Context["key"])
		}
	})

	t.Run("HookOutput with stop behavior", func(t *testing.T) {
		output := HookOutput{
			Behavior: HookBehaviorStop,
			Message:  "Stopped",
		}

		if output.Behavior != HookBehaviorStop {
			t.Errorf("Expected Stop behavior, got: %s", output.Behavior)
		}
		if output.Message != "Stopped" {
			t.Errorf("Expected message 'Stopped', got: %s", output.Message)
		}
		if len(output.Permissions) != 0 {
			t.Errorf("Expected no permissions, got: %v", output.Permissions)
		}
		if len(output.Context) != 0 {
			t.Errorf("Expected no context, got: %v", output.Context)
		}
	})
}

// TestHookContext tests hook context structure.
func TestHookContext(t *testing.T) {
	t.Run("HookContext fields", func(t *testing.T) {
		context := HookContext{
			SessionID:     "session123",
			TranscriptPath: "/path/to/transcript",
			Cwd:           "/working/dir",
		}

		if context.SessionID != "session123" {
			t.Errorf("Expected SessionID 'session123', got: %s", context.SessionID)
		}
		if context.TranscriptPath != "/path/to/transcript" {
			t.Errorf("Expected TranscriptPath '/path/to/transcript', got: %s", context.TranscriptPath)
		}
		if context.Cwd != "/working/dir" {
			t.Errorf("Expected Cwd '/working/dir', got: %s", context.Cwd)
		}
	})
}