package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	claudecode "github.com/severity1/claude-code-sdk-go"
)

// This example demonstrates the comprehensive Hook System for lifecycle events.
// Hooks allow you to intercept and respond to events like tool execution,
// user prompts, conversation stops, and history compaction.

func main() {
	ctx := context.Background()

	// Example 1: PreToolUse hook - Log and validate tool usage before execution
	fmt.Println("=== Example 1: PreToolUse Hook - Logging and Validation ===")
	preToolUseExample(ctx)

	fmt.Println("\n=== Example 2: PostToolUse Hook - Inject Additional Context ===")
	postToolUseExample(ctx)

	fmt.Println("\n=== Example 3: UserPromptSubmit Hook - Track User Interactions ===")
	userPromptSubmitExample(ctx)

	fmt.Println("\n=== Example 4: Stop Hook - Cleanup and Summary ===")
	stopHookExample(ctx)

	fmt.Println("\n=== Example 5: Multiple Hooks - Combined Usage ===")
	multipleHooksExample(ctx)

	fmt.Println("\n=== Example 6: Pattern Matching - Tool-Specific Hooks ===")
	patternMatchingExample(ctx)
}

// Example 1: PreToolUse hook logs all tool usage and blocks dangerous commands
func preToolUseExample(ctx context.Context) {
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("Query: What files are in the current directory?")
		return client.Query(ctx, "What files are in the current directory?")
	}, claudecode.WithHooks(map[claudecode.HookEventName]*claudecode.HookMatcher{
		claudecode.HookEventPreToolUse: {
			Matcher: nil, // nil matcher matches all tools
			Hooks: []claudecode.HookCallback{
				func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
					// Log all tool usage
					fmt.Printf("[PreToolUse] Tool: %s, Input: %v\n", *input.ToolName, input.ToolInput)

					// Block dangerous bash commands
					if *input.ToolName == "Bash" {
						if command, ok := input.ToolInput["command"].(string); ok {
							if strings.Contains(command, "rm -rf /") {
								fmt.Println("[PreToolUse] ⛔ Blocked dangerous command!")
								return map[string]any{
									"decision":      "deny",
									"systemMessage": "Dangerous command blocked for safety",
									"reason":        "Command contains 'rm -rf /' which is prohibited",
								}, nil
							}
						}
					}

					// Allow the tool to execute
					return map[string]any{
						"continue": true,
					}, nil
				},
			},
		},
	}))

	if err != nil {
		log.Printf("Error: %v\n", err)
	}
}

// Example 2: PostToolUse hook adds context after tool execution
func postToolUseExample(ctx context.Context) {
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("Query: Edit the README file")
		return client.Query(ctx, "Edit the README file")
	}, claudecode.WithHooks(map[claudecode.HookEventName]*claudecode.HookMatcher{
		claudecode.HookEventPostToolUse: {
			Matcher: strPtr("Edit|Write"), // Only trigger for Edit or Write tools
			Hooks: []claudecode.HookCallback{
				func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
					fmt.Printf("[PostToolUse] Tool: %s completed\n", *input.ToolName)
					fmt.Printf("[PostToolUse] Response: %s\n", *input.ToolResponse)

					// Add additional context to the response
					return map[string]any{
						"continue":          true,
						"additionalContext": "File modification logged for audit trail",
					}, nil
				},
			},
		},
	}))

	if err != nil {
		log.Printf("Error: %v\n", err)
	}
}

// Example 3: UserPromptSubmit hook tracks user interactions
func userPromptSubmitExample(ctx context.Context) {
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("Query: Help me debug this code")
		return client.Query(ctx, "Help me debug this code")
	}, claudecode.WithHooks(map[claudecode.HookEventName]*claudecode.HookMatcher{
		claudecode.HookEventUserPromptSubmit: {
			Hooks: []claudecode.HookCallback{
				func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
					fmt.Printf("[UserPromptSubmit] User asked: %s\n", *input.Prompt)
					fmt.Printf("[UserPromptSubmit] Session: %s\n", input.SessionID)

					// Track user interactions (could log to analytics, etc.)
					return map[string]any{
						"continue": true,
					}, nil
				},
			},
		},
	}))

	if err != nil {
		log.Printf("Error: %v\n", err)
	}
}

// Example 4: Stop hook performs cleanup and generates summary
func stopHookExample(ctx context.Context) {
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("Query: Calculate 2+2")
		return client.Query(ctx, "Calculate 2+2")
	}, claudecode.WithHooks(map[claudecode.HookEventName]*claudecode.HookMatcher{
		claudecode.HookEventStop: {
			Hooks: []claudecode.HookCallback{
				func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
					fmt.Println("[Stop] Conversation ended, performing cleanup...")
					fmt.Printf("[Stop] Session ID: %s\n", input.SessionID)

					// Perform cleanup, generate summary, save state, etc.
					return map[string]any{
						"continue": true,
					}, nil
				},
			},
		},
	}))

	if err != nil {
		log.Printf("Error: %v\n", err)
	}
}

// Example 5: Multiple hooks working together
func multipleHooksExample(ctx context.Context) {
	toolUsageCount := 0

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("Query: List files and create a summary")
		return client.Query(ctx, "List files and create a summary")
	}, claudecode.WithHooks(map[claudecode.HookEventName]*claudecode.HookMatcher{
		claudecode.HookEventPreToolUse: {
			Hooks: []claudecode.HookCallback{
				func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
					toolUsageCount++
					fmt.Printf("[PreToolUse] Tool #%d: %s\n", toolUsageCount, *input.ToolName)
					return map[string]any{"continue": true}, nil
				},
			},
		},
		claudecode.HookEventPostToolUse: {
			Hooks: []claudecode.HookCallback{
				func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
					fmt.Printf("[PostToolUse] Tool %s completed successfully\n", *input.ToolName)
					return map[string]any{"continue": true}, nil
				},
			},
		},
		claudecode.HookEventStop: {
			Hooks: []claudecode.HookCallback{
				func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
					fmt.Printf("[Stop] Total tools used: %d\n", toolUsageCount)
					return map[string]any{"continue": true}, nil
				},
			},
		},
	}))

	if err != nil {
		log.Printf("Error: %v\n", err)
	}
}

// Example 6: Pattern matching for specific tools
func patternMatchingExample(ctx context.Context) {
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("Query: Read and write some files")
		return client.Query(ctx, "Read and write some files")
	}, claudecode.WithHooks(map[claudecode.HookEventName]*claudecode.HookMatcher{
		claudecode.HookEventPreToolUse: {
			Matcher: strPtr("^(Bash|Edit|Write)$"), // Only match Bash, Edit, or Write
			Hooks: []claudecode.HookCallback{
				func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
					fmt.Printf("[PreToolUse] Matched tool: %s (pattern: Bash|Edit|Write)\n", *input.ToolName)
					return map[string]any{"continue": true}, nil
				},
			},
		},
	}))

	if err != nil {
		log.Printf("Error: %v\n", err)
	}
}

// Advanced Example: Async hook execution with timeout
func asyncHookExample(ctx context.Context) {
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("Query: Perform a long-running task")
		return client.Query(ctx, "Perform a long-running task")
	}, claudecode.WithHooks(map[claudecode.HookEventName]*claudecode.HookMatcher{
		claudecode.HookEventPreToolUse: {
			Timeout: floatPtr(120.0), // 2 minute timeout
			Hooks: []claudecode.HookCallback{
				func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
					// Simulate async processing
					time.Sleep(100 * time.Millisecond)

					return map[string]any{
						"continue":     true,
						"async":        true,
						"asyncTimeout": 120.0,
					}, nil
				},
			},
		},
	}))

	if err != nil {
		log.Printf("Error: %v\n", err)
	}
}

// PreCompact hook - triggered before conversation history compaction
func preCompactExample(ctx context.Context) {
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("Query: Long conversation that may trigger compaction")
		return client.Query(ctx, "Long conversation that may trigger compaction")
	}, claudecode.WithHooks(map[claudecode.HookEventName]*claudecode.HookMatcher{
		claudecode.HookEventPreCompact: {
			Hooks: []claudecode.HookCallback{
				func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
					fmt.Printf("[PreCompact] Compaction triggered: %s\n", *input.Trigger)
					fmt.Printf("[PreCompact] Custom instructions: %s\n", *input.CustomInstructions)

					// Optionally modify compaction behavior
					return map[string]any{
						"continue": true,
					}, nil
				},
			},
		},
	}))

	if err != nil {
		log.Printf("Error: %v\n", err)
	}
}

// SubagentStop hook - triggered when a subagent completes
func subagentStopExample(ctx context.Context) {
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("Query: Use a subagent to analyze code")
		return client.Query(ctx, "Use a subagent to analyze code")
	}, claudecode.WithHooks(map[claudecode.HookEventName]*claudecode.HookMatcher{
		claudecode.HookEventSubagentStop: {
			Hooks: []claudecode.HookCallback{
				func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
					fmt.Printf("[SubagentStop] Subagent completed in session: %s\n", input.SessionID)

					return map[string]any{
						"continue": true,
					}, nil
				},
			},
		},
	}))

	if err != nil {
		log.Printf("Error: %v\n", err)
	}
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func floatPtr(f float64) *float64 {
	return &f
}
