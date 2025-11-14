// Package main demonstrates streaming with Client API using automatic resource management.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/severity1/claude-code-sdk-go"
)

func main() {
	fmt.Println("Claude Code SDK - Client Streaming Example")
	fmt.Println("Asking: Explain Go goroutines with a simple example")

	ctx := context.Background()
	question := "Explain what Go goroutines are and show a simple example"

	// WithClient handles connection lifecycle automatically
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("\nConnected! Streaming response:")

		if err := client.Query(ctx, question); err != nil {
			return fmt.Errorf("query failed: %w", err)
		}

		// Stream messages in real-time
		// ReceiveMessages() returns a channel that closes automatically when
		// Claude's response completes (detected by ResultMessage)
		msgChan := client.ReceiveMessages(ctx)
		for {
			select {
			case message := <-msgChan:
				if message == nil {
					// Channel closed - this now works correctly after the fix!
					fmt.Println("\n\n✓ Channel closed automatically")
					return nil // Stream ended

				switch msg := message.(type) {
				case *claudecode.AssistantMessage:
					// Print streaming text as it arrives
					for _, block := range msg.Content {
						if textBlock, ok := block.(*claudecode.TextBlock); ok {
							fmt.Print(textBlock.Text)
						}
					}
				case *claudecode.ResultMessage:
					if msg.IsError {
						return fmt.Errorf("error: %s", msg.Result)
					}
					return nil // Success, stream complete
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
	if err != nil {
		log.Fatalf("Streaming failed: %v", err)
	}

	fmt.Println("\n\nStreaming completed!")
}
