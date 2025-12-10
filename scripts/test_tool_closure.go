// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/severity1/claude-code-sdk-go"
)

func main() {
	fmt.Println("🔧 Testing Tool Usage Channel Closure Fix")
	fmt.Println("=====================================")

	// Test case with tool usage (from bug report scenario)
	client := claudecode.NewClient(
		claudecode.WithCwd("/workspace/goagent/claude-code-sdk-go"),
		claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions),
		claudecode.WithAllowedTools("Read", "Write", "Bash"),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	fmt.Println("✓ Client connected successfully")

	// Send a query that uses tools
	err := client.Query(ctx, "List all Go files in the current directory using the Read tool")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Println("✓ Tool-using query sent successfully")

	// Test ReceiveMessages channel closure
	msgChan := client.ReceiveMessages(ctx)
	timeout := time.After(10 * time.Second) // Longer timeout for tool operations

	messageCount := 0
	channelClosed := false
	hasToolUse := false
	hasToolResult := false
	hasResultMessage := false

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				channelClosed = true
				fmt.Printf("✅ SUCCESS: Channel closed after %d messages\n", messageCount)
				fmt.Printf("   - Tool uses detected: %v\n", hasToolUse)
				fmt.Printf("   - Tool results detected: %v\n", hasToolResult)
				fmt.Printf("   - ResultMessage detected: %v\n", hasResultMessage)
				goto testComplete
			}
			messageCount++

			// Track message types (similar to bug report)
			switch msg.(type) {
			case *claudecode.ToolUseBlock:
				hasToolUse = true
				fmt.Printf("🔧 Received tool use message %d\n", messageCount)
			case *claudecode.ToolResultBlock:
				hasToolResult = true
				fmt.Printf("✅ Received tool result message %d\n", messageCount)
			case *claudecode.ResultMessage:
				hasResultMessage = true
				fmt.Printf("🏁 Received result message %d\n", messageCount)
			default:
				fmt.Printf("📨 Received message %d: %T\n", messageCount, msg)
			}

		case <-timeout:
			fmt.Printf("❌ BUG: Channel stayed open after %d messages\n", messageCount)
			fmt.Println("❌ This suggests tool usage scenarios still have the closure bug")
			return

		case <-ctx.Done():
			fmt.Println("⚠️ Context cancelled")
			return
		}
	}

testComplete:
	if channelClosed {
		fmt.Println("🎉 Tool usage fix verified: Channel closes even with complex tool interactions!")
	} else {
		fmt.Println("❌ Tool usage fix failed: Channel did not close")
	}
}
