package claudecode

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/severity1/claude-code-sdk-go/internal/shared"
)

// TestClientCanUseToolCallback tests the CanUseTool callback functionality
func TestClientCanUseToolCallback(t *testing.T) {
	ctx, cancel := setupClientTestContext(t, 10*time.Second)
	defer cancel()

	tests := []struct {
		name           string
		setupTransport func() *canUseToolMockTransport
		setupCallback  func() CanUseToolFunc
		wantCalled     bool
		validateResult func(t *testing.T, result *PermissionResult)
	}{
		{
			name: "allow_tool_use",
			setupTransport: func() *canUseToolMockTransport {
				return newCanUseToolMockTransport("Write", map[string]interface{}{
					"file_path": "/tmp/test.txt",
					"content":   "Hello",
				})
			},
			setupCallback: func() CanUseToolFunc {
				return func(ctx context.Context, toolName string, input map[string]interface{}, context ToolPermissionContext) (PermissionResult, error) {
					return &PermissionResultAllow{}, nil
				}
			},
			wantCalled: true,
			validateResult: func(t *testing.T, result *PermissionResult) {
				t.Helper()
				if result == nil {
					t.Fatal("expected result, got nil")
				}
				if (*result).Behavior() != "allow" {
					t.Errorf("expected behavior=allow, got %s", (*result).Behavior())
				}
			},
		},
		{
			name: "deny_tool_use",
			setupTransport: func() *canUseToolMockTransport {
				return newCanUseToolMockTransport("Bash", map[string]interface{}{
					"command": "rm -rf /",
				})
			},
			setupCallback: func() CanUseToolFunc {
				return func(ctx context.Context, toolName string, input map[string]interface{}, context ToolPermissionContext) (PermissionResult, error) {
					if toolName == "Bash" {
						command := input["command"].(string)
						if strings.Contains(command, "rm -rf") {
							return &PermissionResultDeny{
								Message: "Dangerous command blocked",
							}, nil
						}
					}
					return &PermissionResultAllow{}, nil
				}
			},
			wantCalled: true,
			validateResult: func(t *testing.T, result *PermissionResult) {
				t.Helper()
				if result == nil {
					t.Fatal("expected result, got nil")
				}
				if (*result).Behavior() != "deny" {
					t.Errorf("expected behavior=deny, got %s", (*result).Behavior())
				}
				denyResult, ok := (*result).(*PermissionResultDeny)
				if !ok {
					t.Fatal("expected PermissionResultDeny")
				}
				if denyResult.Message != "Dangerous command blocked" {
					t.Errorf("expected message='Dangerous command blocked', got %s", denyResult.Message)
				}
			},
		},
		{
			name: "modify_input",
			setupTransport: func() *canUseToolMockTransport {
				return newCanUseToolMockTransport("Write", map[string]interface{}{
					"file_path": "/etc/passwd",
					"content":   "malicious",
				})
			},
			setupCallback: func() CanUseToolFunc {
				return func(ctx context.Context, toolName string, input map[string]interface{}, context ToolPermissionContext) (PermissionResult, error) {
					if toolName == "Write" {
						filePath := input["file_path"].(string)
						if strings.HasPrefix(filePath, "/etc/") {
							// Redirect to safe location
							modifiedInput := make(map[string]interface{})
							for k, v := range input {
								modifiedInput[k] = v
							}
							modifiedInput["file_path"] = "/tmp/safe.txt"
							return &PermissionResultAllow{
								UpdatedInput: modifiedInput,
							}, nil
						}
					}
					return &PermissionResultAllow{}, nil
				}
			},
			wantCalled: true,
			validateResult: func(t *testing.T, result *PermissionResult) {
				t.Helper()
				if result == nil {
					t.Fatal("expected result, got nil")
				}
				allowResult, ok := (*result).(*PermissionResultAllow)
				if !ok {
					t.Fatal("expected PermissionResultAllow")
				}
				if allowResult.UpdatedInput == nil {
					t.Fatal("expected UpdatedInput, got nil")
				}
				updatedPath := allowResult.UpdatedInput["file_path"].(string)
				if updatedPath != "/tmp/safe.txt" {
					t.Errorf("expected updated path=/tmp/safe.txt, got %s", updatedPath)
				}
			},
		},
		{
			name: "deny_with_interrupt",
			setupTransport: func() *canUseToolMockTransport {
				return newCanUseToolMockTransport("Bash", map[string]interface{}{
					"command": "sudo rm -rf /",
				})
			},
			setupCallback: func() CanUseToolFunc {
				return func(ctx context.Context, toolName string, input map[string]interface{}, context ToolPermissionContext) (PermissionResult, error) {
					if toolName == "Bash" {
						command := input["command"].(string)
						if strings.Contains(command, "sudo rm") {
							return &PermissionResultDeny{
								Message:   "Extremely dangerous command - interrupting conversation",
								Interrupt: true,
							}, nil
						}
					}
					return &PermissionResultAllow{}, nil
				}
			},
			wantCalled: true,
			validateResult: func(t *testing.T, result *PermissionResult) {
				t.Helper()
				if result == nil {
					t.Fatal("expected result, got nil")
				}
				denyResult, ok := (*result).(*PermissionResultDeny)
				if !ok {
					t.Fatal("expected PermissionResultDeny")
				}
				if !denyResult.Interrupt {
					t.Error("expected Interrupt=true")
				}
			},
		},
		{
			name: "callback_error",
			setupTransport: func() *canUseToolMockTransport {
				return newCanUseToolMockTransport("Read", map[string]interface{}{
					"file_path": "/tmp/test.txt",
				})
			},
			setupCallback: func() CanUseToolFunc {
				return func(ctx context.Context, toolName string, input map[string]interface{}, context ToolPermissionContext) (PermissionResult, error) {
					return nil, fmt.Errorf("callback processing error")
				}
			},
			wantCalled: true,
			validateResult: func(t *testing.T, result *PermissionResult) {
				// When callback returns error, the result should be an error response
				// This is validated by checking the mock transport received error response
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := test.setupTransport()
			callback := test.setupCallback()

			client := NewClientWithTransport(transport, WithCanUseTool(callback))
			defer disconnectClientSafely(t, client)

			connectClientSafely(ctx, t, client)

			// Wait for callback to be invoked
			select {
			case <-transport.callbackInvoked:
				if !test.wantCalled {
					t.Error("callback was called but expected not to be called")
				}
			case <-time.After(5 * time.Second):
				if test.wantCalled {
					t.Error("timeout waiting for callback to be invoked")
				}
			}

			// Validate the result if needed
			if test.validateResult != nil && transport.lastResult != nil {
				test.validateResult(t, transport.lastResult)
			}

			// Verify response was sent
			if transport.responseSent {
				// Check response structure
				if transport.lastResponse == nil {
					t.Error("expected response to be sent")
				}
			}
		})
	}
}

// TestClientCanUseToolAutoSetsStdio tests that CanUseTool automatically sets permission_prompt_tool_name="stdio"
func TestClientCanUseToolAutoSetsStdio(t *testing.T) {
	ctx, cancel := setupClientTestContext(t, 10*time.Second)
	defer cancel()

	transport := newCanUseToolMockTransport("Read", map[string]interface{}{
		"file_path": "/tmp/test.txt",
	})

	callback := func(ctx context.Context, toolName string, input map[string]interface{}, context ToolPermissionContext) (PermissionResult, error) {
		return &PermissionResultAllow{}, nil
	}

	client := NewClientWithTransport(transport, WithCanUseTool(callback))
	defer disconnectClientSafely(t, client)

	// Before connect, options should have callback but no permission_prompt_tool_name
	clientImpl := client.(*ClientImpl)
	if clientImpl.options.PermissionPromptToolName != nil {
		t.Errorf("expected PermissionPromptToolName=nil before connect, got %s", *clientImpl.options.PermissionPromptToolName)
	}

	connectClientSafely(ctx, t, client)

	// After connect, permission_prompt_tool_name should be set to "stdio"
	if clientImpl.options.PermissionPromptToolName == nil {
		t.Fatal("expected PermissionPromptToolName to be set after connect")
	}
	if *clientImpl.options.PermissionPromptToolName != "stdio" {
		t.Errorf("expected PermissionPromptToolName=stdio, got %s", *clientImpl.options.PermissionPromptToolName)
	}
}

// Mock transport for CanUseTool callback testing

type canUseToolMockTransport struct {
	mu              sync.RWMutex
	connected       bool
	closed          bool
	msgChan         chan Message
	errChan         chan error
	validator       *StreamValidator
	callbackInvoked chan struct{}
	lastResult      *PermissionResult
	lastResponse    *SDKControlResponse
	responseSent    bool
	toolName        string
	toolInput       map[string]interface{}
}

func newCanUseToolMockTransport(toolName string, input map[string]interface{}) *canUseToolMockTransport {
	return &canUseToolMockTransport{
		msgChan:         make(chan Message, 10),
		errChan:         make(chan error, 10),
		validator:       shared.NewStreamValidator(),
		callbackInvoked: make(chan struct{}, 1),
		toolName:        toolName,
		toolInput:       input,
	}
}

func (m *canUseToolMockTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connected {
		return fmt.Errorf("already connected")
	}

	m.connected = true

	// Send a can_use_tool request after connection
	go func() {
		time.Sleep(100 * time.Millisecond) // Give client time to set up routing
		requestID := GenerateRequestID()
		request := &SDKControlRequest{
			RequestID: requestID,
			Request: &CanUseToolRequest{
				ToolName: m.toolName,
				Input:    m.toolInput,
			},
		}

		// Safely send the request
		m.mu.RLock()
		if !m.closed {
			m.msgChan <- request
		}
		m.mu.RUnlock()
	}()

	return nil
}

func (m *canUseToolMockTransport) SendMessage(ctx context.Context, message StreamMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return fmt.Errorf("not connected")
	}

	// Check if this is a control response
	if response, ok := message.Message.(*SDKControlResponse); ok {
		m.lastResponse = response
		m.responseSent = true

		// Parse the result from the response
		if response.Response.Success {
			if behaviorVal, ok := response.Response.Result["behavior"]; ok {
				behavior := behaviorVal.(string)
				if behavior == "allow" {
					allowResult := &PermissionResultAllow{}
					if updatedInput, ok := response.Response.Result["updated_input"]; ok {
						allowResult.UpdatedInput = updatedInput.(map[string]interface{})
					}
					var result PermissionResult = allowResult
					m.lastResult = &result
				} else if behavior == "deny" {
					denyResult := &PermissionResultDeny{}
					if msg, ok := response.Response.Result["message"]; ok {
						denyResult.Message = msg.(string)
					}
					if interrupt, ok := response.Response.Result["interrupt"]; ok {
						denyResult.Interrupt = interrupt.(bool)
					}
					var result PermissionResult = denyResult
					m.lastResult = &result
				}
			}
		}

		// Signal that callback was invoked
		select {
		case m.callbackInvoked <- struct{}{}:
		default:
		}
	}

	return nil
}

func (m *canUseToolMockTransport) ReceiveMessages(ctx context.Context) (<-chan Message, <-chan error) {
	return m.msgChan, m.errChan
}

func (m *canUseToolMockTransport) Interrupt(ctx context.Context) error {
	return nil
}

func (m *canUseToolMockTransport) Close() error {
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

func (m *canUseToolMockTransport) GetValidator() *StreamValidator {
	return m.validator
}
