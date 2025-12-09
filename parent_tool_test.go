package claudecode

import (
	"context"
	"testing"

	"github.com/severity1/claude-code-sdk-go/internal/shared"
)

// MockTransport captures messages for testing
type MockTransport struct {
	sentMessages []shared.StreamMessage
	connectErr   error
	sendErr      error
}

func (m *MockTransport) Connect(ctx context.Context) error {
	return m.connectErr
}

func (m *MockTransport) SendMessage(ctx context.Context, message shared.StreamMessage) error {
	m.sentMessages = append(m.sentMessages, message)
	return m.sendErr
}

func (m *MockTransport) ReceiveMessages(ctx context.Context) (<-chan shared.Message, <-chan error) {
	msgCh := make(chan shared.Message)
	errCh := make(chan error)
	go func() {
		defer close(msgCh)
		defer close(errCh)
	}()
	return msgCh, errCh
}

func (m *MockTransport) Interrupt(ctx context.Context) error {
	return nil
}

func (m *MockTransport) Close() error {
	return nil
}

func (m *MockTransport) GetValidator() *shared.StreamValidator {
	return &shared.StreamValidator{}
}

func TestParentToolUseID(t *testing.T) {
	tests := []struct {
		name             string
		queryMethod      func(client Client, ctx context.Context) error
		expectedParentID *string
	}{
		{
			name: "Query without ParentToolUseID",
			queryMethod: func(client Client, ctx context.Context) error {
				return client.Query(ctx, "Hello world")
			},
			expectedParentID: nil,
		},
		{
			name: "QueryWithParentTool with nil ParentToolUseID",
			queryMethod: func(client Client, ctx context.Context) error {
				return client.QueryWithParentTool(ctx, "Hello world", nil)
			},
			expectedParentID: nil,
		},
		{
			name: "QueryWithParentTool with valid ParentToolUseID",
			queryMethod: func(client Client, ctx context.Context) error {
				parentID := "parent_tool_123"
				return client.QueryWithParentTool(ctx, "Hello world", &parentID)
			},
			expectedParentID: func() *string { s := "parent_tool_123"; return &s }(),
		},
		{
			name: "QueryWithSessionAndParentTool with both values",
			queryMethod: func(client Client, ctx context.Context) error {
				parentID := "parent_tool_456"
				return client.QueryWithSessionAndParentTool(ctx, "Hello world", "session123", &parentID)
			},
			expectedParentID: func() *string { s := "parent_tool_456"; return &s }(),
		},
		{
			name: "QueryWithSessionAndParentTool with nil ParentToolUseID",
			queryMethod: func(client Client, ctx context.Context) error {
				return client.QueryWithSessionAndParentTool(ctx, "Hello world", "session123", nil)
			},
			expectedParentID: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock transport
			transport := &MockTransport{}

			// Create client with mock transport
			client := NewClientWithTransport(transport)

			// Connect client first
			ctx := context.Background()
			err := client.Connect(ctx)
			if err != nil {
				t.Fatalf("Failed to connect client: %v", err)
			}

			// Execute query
			err = tt.queryMethod(client, ctx)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			// Verify exactly one message was sent
			if len(transport.sentMessages) != 1 {
				t.Fatalf("Expected exactly 1 message, got %d", len(transport.sentMessages))
			}

			msg := transport.sentMessages[0]

			// Verify message structure
			if msg.Type != "user" {
				t.Errorf("Expected message type 'user', got '%s'", msg.Type)
			}
			if msg.Message == nil {
				t.Error("Message content should not be nil")
			}

			// Verify ParentToolUseID is correctly set
			if (tt.expectedParentID == nil && msg.ParentToolUseID != nil) ||
				(tt.expectedParentID != nil && msg.ParentToolUseID == nil) ||
				(tt.expectedParentID != nil && msg.ParentToolUseID != nil && *tt.expectedParentID != *msg.ParentToolUseID) {
				t.Errorf("Expected ParentToolUseID %v, got %v", tt.expectedParentID, msg.ParentToolUseID)
			}

			// Verify session ID is set correctly when applicable
			var expectedSessionID string
			if tt.name == "QueryWithSessionAndParentTool with both values" ||
				tt.name == "QueryWithSessionAndParentTool with nil ParentToolUseID" {
				expectedSessionID = "session123"
			} else {
				expectedSessionID = "default"
			}
			if msg.SessionID != expectedSessionID {
				t.Errorf("Expected SessionID '%s', got '%s'", expectedSessionID, msg.SessionID)
			}
		})
	}
}

func TestStreamMessageStructure(t *testing.T) {
	// Test that StreamMessage struct includes ParentToolUseID field
	transport := &MockTransport{}
	client := NewClientWithTransport(transport)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}

	parentID := "test_parent_tool_123"

	err = client.QueryWithParentTool(ctx, "Test message", &parentID)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(transport.sentMessages) != 1 {
		t.Fatalf("Expected exactly 1 message, got %d", len(transport.sentMessages))
	}
	msg := transport.sentMessages[0]

	// Verify the structure contains all expected fields
	if msg.Type != "user" {
		t.Errorf("Expected message type 'user', got '%s'", msg.Type)
	}
	if msg.Message == nil {
		t.Error("Message content should not be nil")
	}
	if msg.ParentToolUseID == nil || *msg.ParentToolUseID != parentID {
		t.Errorf("Expected ParentToolUseID '%s', got %v", parentID, msg.ParentToolUseID)
	}
	if msg.SessionID != "default" {
		t.Errorf("Expected SessionID 'default', got '%s'", msg.SessionID)
	}
}

func TestParentToolUseIDEmptyString(t *testing.T) {
	// Test behavior with empty string ParentToolUseID
	transport := &MockTransport{}
	client := NewClientWithTransport(transport)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}

	emptyParentID := ""

	err = client.QueryWithParentTool(ctx, "Test message", &emptyParentID)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(transport.sentMessages) != 1 {
		t.Fatalf("Expected exactly 1 message, got %d", len(transport.sentMessages))
	}
	msg := transport.sentMessages[0]

	// Empty string should be preserved (not treated as nil)
	if msg.ParentToolUseID == nil {
		t.Error("Expected ParentToolUseID to be non-nil")
	} else if *msg.ParentToolUseID != "" {
		t.Errorf("Expected empty string, got '%s'", *msg.ParentToolUseID)
	}
}