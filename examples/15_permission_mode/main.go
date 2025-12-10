// Package main demonstrates dynamic permission mode changes for runtime permission control.
//
// This example shows how to use SetPermissionMode() to switch between different
// permission policies during a conversation, enabling workflows like:
//   - Review suggestions in default mode (manual approval)
//   - Apply fixes automatically with acceptEdits mode
//   - Plan changes without execution using plan mode
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	claudecode "github.com/severity1/claude-code-sdk-go"
)

func main() {
	// Create context with timeout for the entire workflow
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Use WithClient for automatic resource management
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("=== Permission Mode Workflow Example ===")
		fmt.Println()

		// Phase 1: Review mode (default)
		// In default mode, Claude will ask for permission before making changes
		fmt.Println("Phase 1: Code Review (default mode)")
		fmt.Println("Asking Claude to review code and suggest improvements...")

		err := client.Query(ctx, `Please review the following code and suggest improvements:

func processData(data []string) []string {
    result := []string{}
    for i := 0; i < len(data); i++ {
        if data[i] != "" {
            result = append(result, data[i])
        }
    }
    return result
}

Identify any issues and suggest how to fix them.`)
		if err != nil {
			return fmt.Errorf("review query failed: %w", err)
		}

		// Receive and display Claude's suggestions
		fmt.Println("\nReceiving Claude's review and suggestions...")
		for msg := range client.ReceiveMessages(ctx) {
			if assistantMsg, ok := msg.(*claudecode.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(*claudecode.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			} else if _, ok := msg.(*claudecode.ResultMessage); ok {
				// Review complete
				break
			}
		}

		// Phase 2: Automated fixes with acceptEdits mode
		// Switch to acceptEdits mode to automatically apply fixes without manual approval
		fmt.Println("\n\nPhase 2: Applying Fixes (acceptEdits mode)")
		fmt.Println("Switching to acceptEdits mode for automated fixes...")

		if err := client.SetPermissionMode(ctx, "acceptEdits"); err != nil {
			return fmt.Errorf("failed to set acceptEdits mode: %w", err)
		}

		fmt.Println("Mode changed to acceptEdits - fixes will be applied automatically")
		fmt.Println("Asking Claude to apply the suggested improvements...")

		err = client.Query(ctx, "Please apply the improvements you suggested to the code file.")
		if err != nil {
			return fmt.Errorf("apply fixes query failed: %w", err)
		}

		// Receive confirmation of applied fixes
		fmt.Println("\nReceiving confirmation...")
		for msg := range client.ReceiveMessages(ctx) {
			if assistantMsg, ok := msg.(*claudecode.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(*claudecode.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			} else if _, ok := msg.(*claudecode.ResultMessage); ok {
				// Fixes applied
				break
			}
		}

		// Phase 3: Planning mode
		// Switch to plan mode to analyze further changes without executing them
		fmt.Println("\n\nPhase 3: Planning Additional Changes (plan mode)")
		fmt.Println("Switching to plan mode for analysis...")

		if err := client.SetPermissionMode(ctx, "plan"); err != nil {
			return fmt.Errorf("failed to set plan mode: %w", err)
		}

		fmt.Println("Mode changed to plan - Claude will analyze but not execute changes")
		fmt.Println("Asking Claude to plan performance optimizations...")

		err = client.Query(ctx, "What performance optimizations could be applied to this code? Create a detailed plan.")
		if err != nil {
			return fmt.Errorf("planning query failed: %w", err)
		}

		// Receive optimization plan
		fmt.Println("\nReceiving optimization plan...")
		for msg := range client.ReceiveMessages(ctx) {
			if assistantMsg, ok := msg.(*claudecode.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(*claudecode.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			} else if _, ok := msg.(*claudecode.ResultMessage); ok {
				// Planning complete
				break
			}
		}

		// Phase 4: Return to default mode
		// Always good practice to return to safe default mode
		fmt.Println("\n\nPhase 4: Returning to Default Mode")
		if err := client.SetPermissionMode(ctx, "default"); err != nil {
			return fmt.Errorf("failed to reset to default mode: %w", err)
		}

		fmt.Println("Mode reset to default - conversation ready for next task")

		fmt.Println("\n=== Workflow Complete ===")
		fmt.Println("\nThis example demonstrated:")
		fmt.Println("1. Reviewing code with manual permissions (default)")
		fmt.Println("2. Applying fixes automatically (acceptEdits)")
		fmt.Println("3. Planning changes without execution (plan)")
		fmt.Println("4. Returning to safe default mode")

		return nil
	})

	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}
}
