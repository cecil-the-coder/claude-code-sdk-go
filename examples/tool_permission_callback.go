package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/severity1/claude-code-sdk-go"
)

// Tool usage log for demonstration
var toolUsageLog []map[string]interface{}

// myPermissionCallback controls tool permissions based on tool type and input.
func myPermissionCallback(ctx context.Context, toolName string, input map[string]interface{}, toolContext claudecode.ToolPermissionContext) (claudecode.PermissionResult, error) {
	// Log the tool request
	toolUsageLog = append(toolUsageLog, map[string]interface{}{
		"tool":  toolName,
		"input": input,
	})

	fmt.Printf("\n[PERMISSION CHECK] Tool: %s\n", toolName)
	fmt.Printf("   Input: %v\n", input)

	// Always allow read operations
	if toolName == "Read" || toolName == "Glob" || toolName == "Grep" {
		fmt.Printf("   Decision: ALLOW (read-only operation)\n")
		return &claudecode.PermissionResultAllow{}, nil
	}

	// Deny write operations to system directories
	if toolName == "Write" || toolName == "Edit" {
		filePath, ok := input["file_path"].(string)
		if !ok {
			return &claudecode.PermissionResultAllow{}, nil
		}

		if strings.HasPrefix(filePath, "/etc/") || strings.HasPrefix(filePath, "/usr/") || strings.HasPrefix(filePath, "/sys/") {
			fmt.Printf("   Decision: DENY (system directory write blocked)\n")
			return &claudecode.PermissionResultDeny{
				Message: fmt.Sprintf("Cannot write to system directory: %s", filePath),
			}, nil
		}

		// Redirect writes outside /tmp to /tmp for safety
		if !strings.HasPrefix(filePath, "/tmp/") && !strings.HasPrefix(filePath, "./") {
			safePath := filepath.Join("/tmp", filepath.Base(filePath))
			fmt.Printf("   Decision: ALLOW with modified path (%s -> %s)\n", filePath, safePath)

			modifiedInput := make(map[string]interface{})
			for k, v := range input {
				modifiedInput[k] = v
			}
			modifiedInput["file_path"] = safePath

			return &claudecode.PermissionResultAllow{
				UpdatedInput: modifiedInput,
			}, nil
		}

		fmt.Printf("   Decision: ALLOW (safe path)\n")
		return &claudecode.PermissionResultAllow{}, nil
	}

	// Check dangerous bash commands
	if toolName == "Bash" {
		command, ok := input["command"].(string)
		if !ok {
			return &claudecode.PermissionResultAllow{}, nil
		}

		dangerousCommands := []string{"rm -rf", "sudo", "chmod 777", "dd if=", "mkfs", "> /dev/"}

		for _, dangerous := range dangerousCommands {
			if strings.Contains(command, dangerous) {
				fmt.Printf("   Decision: DENY (dangerous command pattern: %s)\n", dangerous)
				return &claudecode.PermissionResultDeny{
					Message:   fmt.Sprintf("Dangerous command pattern detected: %s", dangerous),
					Interrupt: strings.Contains(command, "rm -rf /"), // Interrupt on especially dangerous commands
				}, nil
			}
		}

		fmt.Printf("   Decision: ALLOW (safe command)\n")
		return &claudecode.PermissionResultAllow{}, nil
	}

	// For all other tools, allow by default
	fmt.Printf("   Decision: ALLOW (default policy)\n")
	return &claudecode.PermissionResultAllow{}, nil
}

func main() {
	fmt.Println("=" + strings.Repeat("=", 58))
	fmt.Println("Tool Permission Callback Example")
	fmt.Println("=" + strings.Repeat("=", 58))
	fmt.Println("\nThis example demonstrates how to:")
	fmt.Println("1. Allow/deny tools based on type")
	fmt.Println("2. Modify tool inputs for safety")
	fmt.Println("3. Log tool usage")
	fmt.Println("4. Block dangerous commands")
	fmt.Println("=" + strings.Repeat("=", 58))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Configure client with permission callback
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("\nSending query to Claude...")
		err := client.Query(ctx, `
			Please demonstrate the following operations:
			1. List files in the current directory
			2. Create a simple text file at /tmp/safe.txt
			3. Try to create a file at /etc/dangerous.txt
			4. Run a safe bash command like 'echo hello'
			5. Try to run a dangerous command like 'rm -rf /tmp'
		`)
		if err != nil {
			return fmt.Errorf("query failed: %w", err)
		}

		fmt.Println("\nProcessing response...")
		messageCount := 0

		for msg := range client.ReceiveMessages(ctx) {
			messageCount++

			if assistantMsg, ok := msg.(*claudecode.AssistantMessage); ok {
				// Print Claude's text responses
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(*claudecode.TextBlock); ok {
						fmt.Printf("\nClaude: %s\n", textBlock.Text)
					}
				}
			} else if resultMsg, ok := msg.(*claudecode.ResultMessage); ok {
				fmt.Println("\nTask completed!")
				if resultMsg.DurationMs > 0 {
					fmt.Printf("   Duration: %dms\n", resultMsg.DurationMs)
				}
				if resultMsg.TotalCostUSD != nil {
					fmt.Printf("   Cost: $%.4f\n", *resultMsg.TotalCostUSD)
				}
				fmt.Printf("   Messages processed: %d\n", messageCount)
			}
		}

		return nil
	}, claudecode.WithCanUseTool(myPermissionCallback),
		claudecode.WithCwd(".")) // Set working directory

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print tool usage summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Tool Usage Summary")
	fmt.Println(strings.Repeat("=", 60))
	for i, usage := range toolUsageLog {
		fmt.Printf("\n%d. Tool: %s\n", i+1, usage["tool"])
		if input, ok := usage["input"].(map[string]interface{}); ok {
			for k, v := range input {
				fmt.Printf("   %s: %v\n", k, v)
			}
		}
	}

	fmt.Println("\nExample completed successfully!")
}
