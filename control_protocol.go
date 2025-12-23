package claudecode

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ControlRequestType represents the type of control request
type ControlRequestType string

const (
	ControlRequestTypeInitialize         ControlRequestType = "initialize"
	ControlRequestTypeCanUseTool        ControlRequestType = "can_use_tool"
	ControlRequestTypeSetPermissionMode ControlRequestType = "set_permission_mode"
	ControlRequestTypeSetModel           ControlRequestType = "set_model"
	ControlRequestTypeInterrupt           ControlRequestType = "interrupt"
	ControlRequestTypeRewindFiles         ControlRequestType = "rewind_files"
)

// ControlRequest represents a control protocol request
type ControlRequest struct {
	ID     string                    `json:"id"`
	Subtype ControlRequestType        `json:"subtype"`
	Data    map[string]any            `json:"data,omitempty"`
}

// ControlResponseType represents the type of control response
type ControlResponseType string

const (
	ControlResponseTypeSuccess ControlResponseType = "success"
	ControlResponseTypeError   ControlResponseType = "error"
)

// ControlResponse represents a control protocol response
type ControlResponse struct {
	ID     string                    `json:"id"`
	Subtype ControlResponseType       `json:"subtype"`
	Data    map[string]any            `json:"data,omitempty"`
	Error   *ControlResponseError     `json:"error,omitempty"`
}

// ControlResponseError represents a control protocol error
type ControlResponseError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ControlProtocol manages bidirectional control communication
type ControlProtocol interface {
	// SendRequest sends a control request and returns response
	SendRequest(ctx context.Context, req *ControlRequest) (*ControlResponse, error)

	// RegisterHandler registers a handler for specific control request types
	RegisterHandler(subtype ControlRequestType, handler ControlRequestHandler)

	// HasPermissionSupport returns true if permission callbacks are supported
	HasPermissionSupport() bool

	// HasControlSupport returns true if control protocol is enabled
	HasControlSupport() bool
}

// ControlRequestHandler handles incoming control requests
type ControlRequestHandler func(ctx context.Context, data map[string]any) (map[string]any, error)

// PendingControlResponse tracks pending control responses
type PendingControlResponse struct {
	ResponseChan chan *ControlResponse
	TimeoutChan  chan struct{}
	Done         bool
	mu           sync.RWMutex
}

// ControlClient extends the base Client interface with control protocol support
type ControlClient interface {
	Client

	// SetPermissionMode changes permission mode during conversation
	SetPermissionMode(ctx context.Context, mode PermissionMode) error

	// SetModel changes the AI model during conversation
	SetModel(ctx context.Context, model string) error

	// Interrupt stops the current operation
	Interrupt(ctx context.Context) error

	// RewindFiles restores files to a previous checkpoint
	RewindFiles(ctx context.Context, userMessageID string) error

	// HasPermissionSupport returns true if permission callbacks are supported
	HasPermissionSupport() bool

	// HasControlSupport returns true if control protocol is enabled
	HasControlSupport() bool
}

// controlProtocol implements ControlProtocol
type controlProtocol struct {
	transport          Transport
	pendingResponses   map[string]*PendingControlResponse
	pendingResponsesMu sync.RWMutex
	handlers          map[ControlRequestType]ControlRequestHandler
	handlersMu         sync.RWMutex
	requestID          int64
	requestIDMu        sync.Mutex
}

// NewControlProtocol creates a new control protocol instance
func NewControlProtocol(transport Transport) ControlProtocol {
	return &controlProtocol{
		transport:        transport,
		pendingResponses: make(map[string]*PendingControlResponse),
		handlers:         make(map[ControlRequestType]ControlRequestHandler),
	}
}

// SendRequest sends a control request and waits for response
func (cp *controlProtocol) SendRequest(ctx context.Context, req *ControlRequest) (*ControlResponse, error) {
	if !cp.HasControlSupport() {
		return nil, fmt.Errorf("control protocol not supported by transport")
	}

	// Generate unique request ID
	cp.requestIDMu.Lock()
	cp.requestID++
	reqID := fmt.Sprintf("sdk-ctrl-%d", cp.requestID)
	cp.requestIDMu.Unlock()

	req.ID = reqID

	// Register pending response
	pending := &PendingControlResponse{
		ResponseChan: make(chan *ControlResponse, 1),
		TimeoutChan:  make(chan struct{}, 1),
	}

	cp.pendingResponsesMu.Lock()
	cp.pendingResponses[reqID] = pending
	cp.pendingResponsesMu.Unlock()

	// Send request via transport
	ctrlTransport, ok := cp.transport.(ControlRequestTransport)
	if !ok {
		cp.cleanupPendingResponse(reqID)
		return nil, fmt.Errorf("transport does not support control requests")
	}

	if err := ctrlTransport.SendControlRequest(ctx, req); err != nil {
		cp.cleanupPendingResponse(reqID)
		return nil, fmt.Errorf("failed to send control request: %w", err)
	}

	// Wait for response with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	select {
	case response := <-pending.ResponseChan:
		if response.Error != nil {
			return nil, fmt.Errorf("control request failed: %s", response.Error.Message)
		}
		return response, nil

	case <-pending.TimeoutChan:
		return nil, fmt.Errorf("control request timeout after 30 seconds")

	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for control response")
	}
}

// RegisterHandler registers a handler for control request types
func (cp *controlProtocol) RegisterHandler(subtype ControlRequestType, handler ControlRequestHandler) {
	cp.handlersMu.Lock()
	defer cp.handlersMu.Unlock()
	cp.handlers[subtype] = handler
}

// HasPermissionSupport returns true if permission callbacks are supported
func (cp *controlProtocol) HasPermissionSupport() bool {
	cp.handlersMu.RLock()
	defer cp.handlersMu.RUnlock()

	_, hasInit := cp.handlers[ControlRequestTypeInitialize]
	_, hasCanUse := cp.handlers[ControlRequestTypeCanUseTool]

	return hasInit && hasCanUse
}

// HasControlSupport returns true if control protocol is enabled
func (cp *controlProtocol) HasControlSupport() bool {
	// Check if transport supports control requests
	ctrlTransport, ok := cp.transport.(ControlRequestTransport)
	if !ok {
		return false
	}

	return ctrlTransport.SupportsControlRequests()
}

// HandleControlResponse processes incoming control responses
func (cp *controlProtocol) HandleControlResponse(response *ControlResponse) error {
	cp.pendingResponsesMu.RLock()
	pending, exists := cp.pendingResponses[response.ID]
	cp.pendingResponsesMu.RUnlock()

	if !exists {
		return fmt.Errorf("received response for unknown request ID: %s", response.ID)
	}

	cp.pendingResponsesMu.Lock()
	defer cp.pendingResponsesMu.Unlock()

	if pending.Done {
		return fmt.Errorf("response already processed for request ID: %s", response.ID)
	}

	pending.Done = true
	close(pending.ResponseChan)
	close(pending.TimeoutChan)

	cp.cleanupPendingResponse(response.ID)
	return nil
}

// HandleControlRequest processes incoming control requests
func (cp *controlProtocol) HandleControlRequest(ctx context.Context, req *ControlRequest) (*ControlResponse, error) {
	cp.handlersMu.RLock()
	handler, exists := cp.handlers[req.Subtype]
	cp.handlersMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no handler registered for control request type: %s", req.Subtype)
	}

	// Execute handler
	data, err := handler(ctx, req.Data)
	if err != nil {
		return &ControlResponse{
			ID:   req.ID,
			Subtype: ControlResponseTypeError,
			Error: &ControlResponseError{
				Message: err.Error(),
			},
		}, nil
	}

	return &ControlResponse{
		ID:   req.ID,
		Subtype: ControlResponseTypeSuccess,
		Data:    data,
	}, nil
}

// cleanupPendingResponse removes and cleans up a pending response
func (cp *controlProtocol) cleanupPendingResponse(requestID string) {
	cp.pendingResponsesMu.Lock()
	defer cp.pendingResponsesMu.Unlock()

	if pending, exists := cp.pendingResponses[requestID]; exists {
		if !pending.Done {
			close(pending.ResponseChan)
			close(pending.TimeoutChan)
		}
		delete(cp.pendingResponses, requestID)
	}
}