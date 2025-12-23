package claudecode

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/severity1/claude-code-sdk-go/internal/shared"
)

// TestControlProtocol tests the control protocol implementation.
func TestControlProtocol(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(*MockControlTransport) ControlProtocol
		testFunc   func(*testing.T, ControlProtocol, *MockControlTransport)
		expectErr  bool
	}{
		{
			name: "HasControlSupport returns true when transport supports it",
			setupFunc: func(transport *MockControlTransport) ControlProtocol {
				transport.supportsControl = true
				return NewControlProtocol(transport)
			},
			testFunc: func(t *testing.T, cp ControlProtocol, transport *MockControlTransport) {
				if !cp.HasControlSupport() {
					t.Error("Expected HasControlSupport to return true")
				}
			},
		},
		{
			name: "HasControlSupport returns false when transport doesn't support it",
			setupFunc: func(transport *MockControlTransport) ControlProtocol {
				transport.supportsControl = false
				return NewControlProtocol(transport)
			},
			testFunc: func(t *testing.T, cp ControlProtocol, transport *MockControlTransport) {
				if cp.HasControlSupport() {
					t.Error("Expected HasControlSupport to return false")
				}
			},
		},
		{
			name: "SendRequest fails when transport doesn't support control",
			setupFunc: func(transport *MockControlTransport) ControlProtocol {
				transport.supportsControl = false
				return NewControlProtocol(transport)
			},
			testFunc: func(t *testing.T, cp ControlProtocol, transport *MockControlTransport) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				req := &ControlRequest{
					Subtype: ControlRequestTypeSetPermissionMode,
					Data: map[string]any{
						"mode": PermissionModeAcceptEdits,
					},
				}
				_, err := cp.SendRequest(ctx, req)
				if err == nil {
					t.Error("Expected error when transport doesn't support control")
				}
			},
			expectErr: true,
		},
		{
			name: "RegisterHandler and HasPermissionSupport",
			setupFunc: func(transport *MockControlTransport) ControlProtocol {
				transport.supportsControl = true
				cp := NewControlProtocol(transport)

				// Register handlers for permission support
				cp.RegisterHandler(ControlRequestTypeInitialize, func(ctx context.Context, data map[string]any) (map[string]any, error) {
					return map[string]any{"status": "initialized"}, nil
				})
				cp.RegisterHandler(ControlRequestTypeCanUseTool, func(ctx context.Context, data map[string]any) (map[string]any, error) {
					return map[string]any{"allowed": true}, nil
				})

				return cp
			},
			testFunc: func(t *testing.T, cp ControlProtocol, transport *MockControlTransport) {
				// Test HasPermissionSupport
				if !cp.HasPermissionSupport() {
					t.Error("Expected HasPermissionSupport to return true when both handlers are registered")
				}

				// Test HasControlSupport
				if !cp.HasControlSupport() {
					t.Error("Expected HasControlSupport to return true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewMockControlTransport()
			cp := tt.setupFunc(transport)
			tt.testFunc(t, cp, transport)
		})
	}
}

// MockControlTransport implements Transport and ControlRequestTransport for testing.
type MockControlTransport struct {
	mu                sync.Mutex
	connected         bool
	supportsControl   bool
	sentRequests      []*ControlRequest
	lastRequestID     string
	responseChan       chan *ControlResponse
}

// NewMockControlTransport creates a new mock control transport.
func NewMockControlTransport() *MockControlTransport {
	return &MockControlTransport{
		responseChan: make(chan *ControlResponse, 10),
	}
}

// Connect implements Transport.
func (m *MockControlTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.connected = true
	return nil
}

// SendMessage implements Transport.
func (m *MockControlTransport) SendMessage(ctx context.Context, message shared.StreamMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return fmt.Errorf("transport not connected")
	}

	return nil
}

// ReceiveMessages implements Transport.
func (m *MockControlTransport) ReceiveMessages(ctx context.Context) (<-chan Message, <-chan error) {
	msgChan := make(chan Message, 100)
	errChan := make(chan error, 100)
	return msgChan, errChan
}

// Interrupt implements Transport.
func (m *MockControlTransport) Interrupt(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return fmt.Errorf("transport not connected")
	}

	return nil
}

// Close implements Transport.
func (m *MockControlTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connected {
		m.connected = false
		close(m.responseChan)
	}

	return nil
}

// GetValidator implements Transport.
func (m *MockControlTransport) GetValidator() *shared.StreamValidator {
	return &shared.StreamValidator{}
}

// SendControlRequest implements ControlRequestTransport.
func (m *MockControlTransport) SendControlRequest(ctx context.Context, req *ControlRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected || !m.supportsControl {
		return fmt.Errorf("transport does not support control requests")
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.sentRequests = append(m.sentRequests, req)
	m.lastRequestID = req.ID

	// Send to response channel for testing
	select {
	case m.responseChan <- &ControlResponse{
		ID:      req.ID,
		Subtype: ControlResponseTypeSuccess,
		Data:    map[string]any{"received": true},
	}:
	default:
	}

	return nil
}

// SupportsControlRequests implements ControlRequestTransport.
func (m *MockControlTransport) SupportsControlRequests() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.supportsControl
}

// GetSentRequests returns all sent control requests.
func (m *MockControlTransport) GetSentRequests() []*ControlRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	requests := make([]*ControlRequest, len(m.sentRequests))
	copy(requests, m.sentRequests)
	return requests
}

// SendResponse sends a control response for testing.
func (m *MockControlTransport) SendResponse(resp *ControlResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()

	select {
	case m.responseChan <- resp:
	default:
	}
}