// Example of using custom agents with Claude Code SDK.
//
// This example demonstrates how to define and use custom agents with specific
// tools, prompts, and models.
//
// Usage:
//
//	go run examples/agents/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	claudecode "github.com/severity1/claude-code-sdk-go"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Run examples
	if err := codeReviewerExample(ctx); err != nil {
		log.Printf("Code reviewer example error: %v", err)
	}

	if err := documentationWriterExample(ctx); err != nil {
		log.Printf("Documentation writer example error: %v", err)
	}

	if err := multipleAgentsExample(ctx); err != nil {
		log.Printf("Multiple agents example error: %v", err)
	}
}

func codeReviewerExample(ctx context.Context) error {
	fmt.Println("=== Code Reviewer Agent Example ===")

	sonnet := "sonnet"
	agents := map[string]claudecode.AgentDefinition{
		"code-reviewer": {
			Description: "Reviews code for best practices and potential issues",
			Prompt: "You are a code reviewer. Analyze code for bugs, performance issues, " +
				"security vulnerabilities, and adherence to best practices. " +
				"Provide constructive feedback.",
			Tools: []string{"Read", "Grep"},
			Model: &sonnet,
		},
	}

	iterator, err := claudecode.Query(
		ctx,
		"Use the code-reviewer agent to review the code in internal/shared/options.go",
		claudecode.WithAgents(agents),
	)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer iterator.Close()

	for {
		msg, err := iterator.Next(ctx)
		if err != nil {
			if err == claudecode.ErrNoMoreMessages {
				break
			}
			return fmt.Errorf("iterator error: %w", err)
		}

		if msg == nil {
			break
		}

		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		case *claudecode.ResultMessage:
			if m.TotalCostUSD != nil && *m.TotalCostUSD > 0 {
				fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}

	fmt.Println()
	return nil
}

func documentationWriterExample(ctx context.Context) error {
	fmt.Println("=== Documentation Writer Agent Example ===")

	agents := map[string]claudecode.AgentDefinition{
		"doc-writer": {
			Description: "Writes comprehensive documentation",
			Prompt: "You are a technical documentation expert. Write clear, comprehensive " +
				"documentation with examples. Focus on clarity and completeness.",
			Tools: []string{"Read", "Write", "Edit"},
			Model: nil, // Use default model
		},
	}

	iterator, err := claudecode.Query(
		ctx,
		"Use the doc-writer agent to explain what AgentDefinition is used for",
		claudecode.WithAgents(agents),
	)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer iterator.Close()

	for {
		msg, err := iterator.Next(ctx)
		if err != nil {
			if err == claudecode.ErrNoMoreMessages {
				break
			}
			return fmt.Errorf("iterator error: %w", err)
		}

		if msg == nil {
			break
		}

		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		case *claudecode.ResultMessage:
			if m.TotalCostUSD != nil && *m.TotalCostUSD > 0 {
				fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}

	fmt.Println()
	return nil
}

func multipleAgentsExample(ctx context.Context) error {
	fmt.Println("=== Multiple Agents Example ===")

	sonnet := "sonnet"
	agents := map[string]claudecode.AgentDefinition{
		"analyzer": {
			Description: "Analyzes code structure and patterns",
			Prompt:      "You are a code analyzer. Examine code structure, patterns, and architecture.",
			Tools:       []string{"Read", "Grep", "Glob"},
			Model:       nil,
		},
		"tester": {
			Description: "Creates and runs tests",
			Prompt:      "You are a testing expert. Write comprehensive tests and ensure code quality.",
			Tools:       []string{"Read", "Write", "Bash"},
			Model:       &sonnet,
		},
	}

	iterator, err := claudecode.Query(
		ctx,
		"Use the analyzer agent to find all Go files in the examples/ directory",
		claudecode.WithAgents(agents),
	)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer iterator.Close()

	for {
		msg, err := iterator.Next(ctx)
		if err != nil {
			if err == claudecode.ErrNoMoreMessages {
				break
			}
			return fmt.Errorf("iterator error: %w", err)
		}

		if msg == nil {
			break
		}

		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		case *claudecode.ResultMessage:
			if m.TotalCostUSD != nil && *m.TotalCostUSD > 0 {
				fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}

	fmt.Println()
	return nil
}
