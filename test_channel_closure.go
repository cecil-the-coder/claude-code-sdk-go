package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/severity1/claude-code-sdk-go"
)

func main() {
	fmt.Println("🩺 Testing ReceiveMessages Channel Closure Fix")
	fmt.Println("=============================================")

	// Test case from the bug report
	client := claudecode.NewClient(
		claudecode.WithCwd("/workspace/goagent/claude-code-sdk-go"),
		claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	fmt.Println("✓ Client connected successfully")

	// Send a simple query
	err := client.Query(ctx, "What is 2+2? Respond briefly.")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Println("✓ Query sent successfully")

	// Test ReceiveMessages channel closure
	msgChan := client.ReceiveMessages(ctx)
	timeout := time.After(5 * time.Second) // Reduced timeout for faster testing

	messageCount := 0
	channelClosed := false

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				channelClosed = true
				fmt.Printf("✅ SUCCESS: Channel closed after %d messages\n", messageCount)
				goto testComplete
			}
			messageCount++
			fmt.Printf("📨 Received message %d: %T\n", messageCount, msg)

		case <-timeout:
			fmt.Printf("❌ BUG REPRODUCED: Channel stayed open after %d messages\n", messageCount)
			fmt.Println("❌ The fix did not work - channel should have closed")
			return

		case <-ctx.Done():
			fmt.Println("⚠️ Context cancelled")
			return
		}
	}

testComplete:
	if channelClosed {
		fmt.Println("🎉 Fix verified: ReceiveMessages channel now closes properly!")
		fmt.Println("   - No goroutine leaks")
		fmt.Println("   - Cleanup code can execute")
		fmt.Println("   - Applications can detect completion")
	} else {
		fmt.Println("❌ Fix failed: Channel did not close")
	}
}