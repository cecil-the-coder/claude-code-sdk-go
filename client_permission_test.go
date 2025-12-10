package claudecode

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestClientSetPermissionMode tests the SetPermissionMode method
func TestClientSetPermissionMode(t *testing.T) {
	ctx, cancel := setupClientTestContext(t, 10*time.Second)
	defer cancel()

	tests := []struct {
		name           string
		mode           string
		setupTransport func() *permissionMockTransport
		wantErr        bool
		errorContains  string
	}{
		{
			name: "set_default_mode",
			mode: "default",
			setupTransport: func() *permissionMockTransport {
				return newPermissionMockTransport(true, nil)
			},
			wantErr: false,
		},
		{
			name: "set_accept_edits_mode",
			mode: "acceptEdits",
			setupTransport: func() *permissionMockTransport {
				return newPermissionMockTransport(true, nil)
			},
			wantErr: false,
		},
		{
			name: "set_plan_mode",
			mode: "plan",
			setupTransport: func() *permissionMockTransport {
				return newPermissionMockTransport(true, nil)
			},
			wantErr: false,
		},
		{
			name: "set_bypass_permissions_mode",
			mode: "bypassPermissions",
			setupTransport: func() *permissionMockTransport {
				return newPermissionMockTransport(true, nil)
			},
			wantErr: false,
		},
		{
			name: "cli_validation_error",
			mode: "invalid_mode",
			setupTransport: func() *permissionMockTransport {
				return newPermissionMockTransport(false, fmt.Errorf("invalid permission mode: invalid_mode"))
			},
			wantErr:       true,
			errorContains: "invalid permission mode",
		},
		{
			name: "not_connected",
			mode: "default",
			setupTransport: func() *permissionMockTransport {
				return newPermissionMockTransport(true, nil)
			},
			wantErr:       true,
			errorContains: "not connected",
		},
		{
			name: "timeout_error",
			mode: "default",
			setupTransport: func() *permissionMockTransport {
				return newPermissionMockTransportWithTimeout()
			},
			wantErr:       true,
			errorContains: "timeout",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := test.setupTransport()
			client := setupClientForTest(t, transport)
			defer disconnectClientSafely(t, client)

			// Connect only if not testing "not_connected" case
			if test.name != "not_connected" {
				connectClientSafely(ctx, t, client)
			}

			// Call SetPermissionMode
			err := client.SetPermissionMode(ctx, test.mode)

			// Verify error expectation
			if test.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if test.errorContains != "" && !strings.Contains(err.Error(), test.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", test.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify the control request was sent correctly
				if len(transport.sentMessages) == 0 {
					t.Fatal("Expected control request to be sent")
				}

				// Verify request structure
				sentMsgInterface := transport.sentMessages[0]
				sentMsg, ok := sentMsgInterface.(StreamMessage)
				if !ok {
					t.Fatalf("Expected StreamMessage, got %T", sentMsgInterface)
				}

				// The Message field should be an SDKControlRequest
				req, ok := sentMsg.Message.(*SDKControlRequest)
				if !ok {
					t.Fatalf("Expected Message to be SDKControlRequest, got %T", sentMsg.Message)
				}

				// Verify it's a SetPermissionMode request
				permReq, ok := req.Request.(*SetPermissionModeRequest)
				if !ok {
					t.Fatalf("Expected SetPermissionModeRequest, got %T", req.Request)
				}

				if permReq.Mode != test.mode {
					t.Errorf("Expected mode %q, got %q", test.mode, permReq.Mode)
				}
			}
		})
	}
}

// TestClientSetPermissionModeTransitions tests mode transitions
func TestClientSetPermissionModeTransitions(t *testing.T) {
	ctx, cancel := setupClientTestContext(t, 15*time.Second)
	defer cancel()

	transport := newPermissionMockTransport(true, nil)
	client := setupClientForTest(t, transport)
	defer disconnectClientSafely(t, client)

	connectClientSafely(ctx, t, client)

	// Test sequence of mode changes
	modes := []string{"default", "acceptEdits", "plan", "bypassPermissions", "default"}

	for i, mode := range modes {
		err := client.SetPermissionMode(ctx, mode)
		if err != nil {
			t.Fatalf("Failed to set mode %q at step %d: %v", mode, i, err)
		}

		// Verify each request was sent
		if len(transport.sentMessages) != i+1 {
			t.Errorf("Expected %d messages sent, got %d", i+1, len(transport.sentMessages))
		}
	}
}

// TestClientSetPermissionModeConcurrency tests concurrent mode changes
func TestClientSetPermissionModeConcurrency(t *testing.T) {
	ctx, cancel := setupClientTestContext(t, 15*time.Second)
	defer cancel()

	transport := newPermissionMockTransport(true, nil)
	client := setupClientForTest(t, transport)
	defer disconnectClientSafely(t, client)

	connectClientSafely(ctx, t, client)

	const numGoroutines = 5
	modes := []string{"default", "acceptEdits", "plan", "bypassPermissions"}

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mode := modes[id%len(modes)]
			err := client.SetPermissionMode(ctx, mode)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %w", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent SetPermissionMode error: %v", err)
	}

	// Verify all requests were sent
	if len(transport.sentMessages) != numGoroutines {
		t.Errorf("Expected %d messages sent, got %d", numGoroutines, len(transport.sentMessages))
	}
}

// TestClientSetPermissionModeContextCancellation tests context cancellation
func TestClientSetPermissionModeContextCancellation(t *testing.T) {
	transport := newPermissionMockTransport(true, nil)
	client := setupClientForTest(t, transport)
	defer disconnectClientSafely(t, client)

	// Connect with a valid context
	connectCtx, connectCancel := setupClientTestContext(t, 5*time.Second)
	defer connectCancel()
	connectClientSafely(connectCtx, t, client)

	// Create a cancelled context for SetPermissionMode
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.SetPermissionMode(ctx, "default")
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

// TestClientSetPermissionModeTimeout tests timeout handling
func TestClientSetPermissionModeTimeout(t *testing.T) {
	ctx, cancel := setupClientTestContext(t, 10*time.Second)
	defer cancel()

	// Create transport that never responds
	transport := newPermissionMockTransportWithTimeout()
	client := setupClientForTest(t, transport)
	defer disconnectClientSafely(t, client)

	connectClientSafely(ctx, t, client)

	// Use a short timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer timeoutCancel()

	err := client.SetPermissionMode(timeoutCtx, "default")
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// permissionMockTransport is a mock transport for testing permission mode changes
type permissionMockTransport struct {
	mu              sync.Mutex
	connected       bool
	closed          bool
	sentMessages    []interface{}
	msgChan         chan Message
	errChan         chan error
	responseSuccess bool
	responseError   error
	noResponse      bool // For timeout testing
	pendingMgr      *PendingControlResponses
}

func newPermissionMockTransport(success bool, respErr error) *permissionMockTransport {
	return &permissionMockTransport{
		msgChan:         make(chan Message, 10),
		errChan:         make(chan error, 10),
		responseSuccess: success,
		responseError:   respErr,
		pendingMgr:      NewPendingControlResponses(),
	}
}

func newPermissionMockTransportWithTimeout() *permissionMockTransport {
	return &permissionMockTransport{
		msgChan:    make(chan Message, 10),
		errChan:    make(chan error, 10),
		noResponse: true,
		pendingMgr: NewPendingControlResponses(),
	}
}

func (m *permissionMockTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.connected = true
	return nil
}

func (m *permissionMockTransport) SendMessage(ctx context.Context, message StreamMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if !m.connected {
		return fmt.Errorf("not connected")
	}

	m.sentMessages = append(m.sentMessages, message)

	// If this is a control request, send a response (unless we're testing timeout)
	if !m.noResponse {
		// Check if the Message field is an SDKControlRequest
		if req, ok := message.Message.(*SDKControlRequest); ok {
			// Send response asynchronously to simulate CLI behavior
			go func() {
				time.Sleep(10 * time.Millisecond)

				response := &SDKControlResponse{
					RequestID: req.RequestID,
					Response: ControlResponseData{
						Success: m.responseSuccess,
					},
				}

				if m.responseError != nil {
					response.Response.Error = m.responseError.Error()
					response.Response.Success = false
				}

				m.msgChan <- response
			}()
		}
	}

	return nil
}

func (m *permissionMockTransport) ReceiveMessages(_ context.Context) (<-chan Message, <-chan error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		closedMsgChan := make(chan Message)
		closedErrChan := make(chan error)
		close(closedMsgChan)
		close(closedErrChan)
		return closedMsgChan, closedErrChan
	}

	return m.msgChan, m.errChan
}

func (m *permissionMockTransport) Interrupt(_ context.Context) error {
	return nil
}

func (m *permissionMockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.connected = false
	m.closed = true

	close(m.msgChan)
	close(m.errChan)

	return nil
}

func (m *permissionMockTransport) GetValidator() *StreamValidator {
	return &StreamValidator{}
}
