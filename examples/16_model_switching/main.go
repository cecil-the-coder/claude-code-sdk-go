// Package main demonstrates dynamic model switching mid-conversation.
// This example shows how to switch between different Claude models (e.g., Haiku for analysis,
// Opus for implementation) while preserving conversation context.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	claudecode "github.com/severity1/claude-code-sdk-go"
)

func main() {
	// Create a context with timeout for the entire operation
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start with Haiku for quick analysis
	haikuModel := "claude-3-5-haiku-20241022"

	// Use WithClient for automatic resource management
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("=== Dynamic Model Switching Example ===")
		fmt.Println()

		// Step 1: Use Haiku for initial analysis (fast, cost-effective)
		fmt.Printf("Using model: %s (Haiku - Fast analysis)\n", haikuModel)
		fmt.Println("Asking: Analyze the pros and cons of TDD")

		err := client.Query(ctx, "Briefly analyze the pros and cons of Test-Driven Development in 2-3 sentences.")
		if err != nil {
			return fmt.Errorf("haiku query failed: %w", err)
		}

		// Receive and display Haiku's response
		fmt.Println("\nHaiku's Analysis:")
		if err := receiveAndPrint(ctx, client); err != nil {
			return err
		}

		// Step 2: Switch to Opus for detailed implementation
		opusModel := "claude-opus-4-1-20250805"
		fmt.Printf("\n--- Switching to model: %s (Opus - Deep implementation) ---\n", opusModel)

		if err := client.SetModel(ctx, &opusModel); err != nil {
			return fmt.Errorf("model switch failed: %w", err)
		}

		fmt.Println("Model switched successfully!")
		fmt.Println("\nAsking: Give me a detailed implementation plan for TDD")

		err = client.Query(ctx, "Based on the TDD analysis, give me a detailed step-by-step implementation plan with code examples in Go.")
		if err != nil {
			return fmt.Errorf("opus query failed: %w", err)
		}

		// Receive and display Opus's detailed response
		fmt.Println("\nOpus's Detailed Implementation Plan:")
		if err := receiveAndPrint(ctx, client); err != nil {
			return err
		}

		// Step 3: Switch back to Haiku for quick summary
		fmt.Printf("\n--- Switching back to model: %s (Haiku - Quick summary) ---\n", haikuModel)

		if err := client.SetModel(ctx, &haikuModel); err != nil {
			return fmt.Errorf("model switch back failed: %w", err)
		}

		fmt.Println("Model switched successfully!")
		fmt.Println("\nAsking: Summarize the key takeaways")

		err = client.Query(ctx, "Summarize the key takeaways from our TDD discussion in 1-2 sentences.")
		if err != nil {
			return fmt.Errorf("final haiku query failed: %w", err)
		}

		// Receive and display final summary
		fmt.Println("\nHaiku's Summary:")
		if err := receiveAndPrint(ctx, client); err != nil {
			return err
		}

		fmt.Println("\n=== Example Complete ===")
		fmt.Println("Successfully demonstrated:")
		fmt.Println("  - Haiku for quick analysis")
		fmt.Println("  - Opus for detailed implementation")
		fmt.Println("  - Haiku again for summary")
		fmt.Println("  - Conversation context preserved across all model switches")

		return nil
	})

	if err != nil {
		log.Fatalf("Example failed: %v", err)
	}
}

// receiveAndPrint receives and prints assistant responses
func receiveAndPrint(ctx context.Context, client claudecode.Client) error {
	msgChan := client.ReceiveMessages(ctx)

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				// Channel closed, we're done
				return nil
			}

			// Handle assistant message
			if assistantMsg, ok := msg.(*claudecode.AssistantMessage); ok {
				for _, content := range assistantMsg.Content {
					if textBlock, ok := content.(*claudecode.TextBlock); ok {
						fmt.Println(textBlock.Text)
					}
				}
			}

			// Handle result message (end of response)
			if resultMsg, ok := msg.(*claudecode.ResultMessage); ok {
				if resultMsg.TotalCostUSD != nil {
					fmt.Printf("\n[Cost: $%.6f]\n", *resultMsg.TotalCostUSD)
				}
				return nil
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
