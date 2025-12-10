// Package shared provides control protocol types for bidirectional SDK↔CLI communication.
package shared

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Message type constants for control protocol
const (
	MessageTypeSDKControlRequest  = "sdk_control_request"
	MessageTypeSDKControlResponse = "sdk_control_response"
)

// Control request type constants
const (
	ControlRequestTypeInitialize        = "initialize"
	ControlRequestTypeCanUseTool        = "can_use_tool"
	ControlRequestTypeHookCallback      = "hook_callback"
	ControlRequestTypeSetPermissionMode = "set_permission_mode"
	ControlRequestTypeSetModel          = "set_model"
	ControlRequestTypeInterrupt         = "interrupt"
	ControlRequestTypeMCPMessage        = "mcp_message"
)

// Default timeout for control requests
const DefaultControlRequestTimeout = 60 * time.Second

// ControlRequest is the interface for all control request types.
// This enables discriminated union pattern for different request subtypes.
type ControlRequest interface {
	RequestType() string
}

// InitializeRequest registers hooks and callbacks with the CLI.
type InitializeRequest struct {
	Hooks     []string       `json:"hooks"`
	Callbacks []CallbackInfo `json:"callbacks"`
}

// RequestType returns the request type for InitializeRequest.
func (r *InitializeRequest) RequestType() string {
	return ControlRequestTypeInitialize
}

// CallbackInfo contains information about a registered callback.
type CallbackInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CanUseToolRequest requests permission to use a tool.
type CanUseToolRequest struct {
	ToolName    string                 `json:"tool_name"`
	Input       map[string]interface{} `json:"input"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// RequestType returns the request type for CanUseToolRequest.
func (r *CanUseToolRequest) RequestType() string {
	return ControlRequestTypeCanUseTool
}

// HookCallbackRequest executes a registered hook callback.
type HookCallbackRequest struct {
	CallbackID string                 `json:"callback_id"`
	Input      map[string]interface{} `json:"input"`
	ToolUseID  string                 `json:"tool_use_id"`
}

// RequestType returns the request type for HookCallbackRequest.
func (r *HookCallbackRequest) RequestType() string {
	return ControlRequestTypeHookCallback
}

// SetPermissionModeRequest changes the permission mode at runtime.
type SetPermissionModeRequest struct {
	Mode string `json:"mode"`
}

// RequestType returns the request type for SetPermissionModeRequest.
func (r *SetPermissionModeRequest) RequestType() string {
	return ControlRequestTypeSetPermissionMode
}

// SetModelRequest switches the model mid-conversation.
type SetModelRequest struct {
	Model string `json:"model"`
}

// RequestType returns the request type for SetModelRequest.
func (r *SetModelRequest) RequestType() string {
	return ControlRequestTypeSetModel
}

// InterruptRequest cancels the current operation.
type InterruptRequest struct{}

// RequestType returns the request type for InterruptRequest.
func (r *InterruptRequest) RequestType() string {
	return ControlRequestTypeInterrupt
}

// MCPMessageRequest routes messages to MCP servers.
type MCPMessageRequest struct {
	ServerName string                 `json:"server_name"`
	Message    map[string]interface{} `json:"message"`
}

// RequestType returns the request type for MCPMessageRequest.
func (r *MCPMessageRequest) RequestType() string {
	return ControlRequestTypeMCPMessage
}

// SDKControlRequest represents a control protocol request from SDK to CLI.
type SDKControlRequest struct {
	RequestID string         `json:"request_id"`
	Request   ControlRequest `json:"request"`
}

// Type returns the message type for SDKControlRequest.
func (r *SDKControlRequest) Type() string {
	return MessageTypeSDKControlRequest
}

// MarshalJSON implements custom JSON marshaling for SDKControlRequest.
func (r *SDKControlRequest) MarshalJSON() ([]byte, error) {
	// Create a map with type field
	data := map[string]interface{}{
		"type":       MessageTypeSDKControlRequest,
		"request_id": r.RequestID,
	}

	// Marshal the request to get its JSON representation
	requestData, err := json.Marshal(r.Request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Unmarshal into map to merge with top-level data
	var requestMap map[string]interface{}
	if err := json.Unmarshal(requestData, &requestMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request map: %w", err)
	}

	// Add request fields to data
	data["request"] = requestMap

	return json.Marshal(data)
}

// UnmarshalJSON implements custom JSON unmarshaling for SDKControlRequest.
func (r *SDKControlRequest) UnmarshalJSON(data []byte) error {
	// Parse raw structure
	var raw struct {
		Type      string          `json:"type"`
		RequestID string          `json:"request_id"`
		Request   json.RawMessage `json:"request"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal SDKControlRequest: %w", err)
	}

	r.RequestID = raw.RequestID

	// Discriminate on request type
	var typeCheck struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw.Request, &typeCheck); err != nil {
		return fmt.Errorf("failed to check request type: %w", err)
	}

	// Instantiate appropriate request type
	var request ControlRequest
	switch typeCheck.Type {
	case ControlRequestTypeInitialize:
		var req InitializeRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return fmt.Errorf("failed to unmarshal InitializeRequest: %w", err)
		}
		request = &req
	case ControlRequestTypeCanUseTool:
		var req CanUseToolRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return fmt.Errorf("failed to unmarshal CanUseToolRequest: %w", err)
		}
		request = &req
	case ControlRequestTypeHookCallback:
		var req HookCallbackRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return fmt.Errorf("failed to unmarshal HookCallbackRequest: %w", err)
		}
		request = &req
	case ControlRequestTypeSetPermissionMode:
		var req SetPermissionModeRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return fmt.Errorf("failed to unmarshal SetPermissionModeRequest: %w", err)
		}
		request = &req
	case ControlRequestTypeSetModel:
		var req SetModelRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return fmt.Errorf("failed to unmarshal SetModelRequest: %w", err)
		}
		request = &req
	case ControlRequestTypeInterrupt:
		request = &InterruptRequest{}
	case ControlRequestTypeMCPMessage:
		var req MCPMessageRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return fmt.Errorf("failed to unmarshal MCPMessageRequest: %w", err)
		}
		request = &req
	default:
		return fmt.Errorf("unknown control request type: %s", typeCheck.Type)
	}

	r.Request = request
	return nil
}

// ControlResponseData represents the response data for a control request.
type ControlResponseData struct {
	Success bool                   `json:"success"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// SDKControlResponse represents a control protocol response from CLI to SDK.
type SDKControlResponse struct {
	RequestID string              `json:"request_id"`
	Response  ControlResponseData `json:"response"`
}

// Type returns the message type for SDKControlResponse.
func (r *SDKControlResponse) Type() string {
	return MessageTypeSDKControlResponse
}

// MarshalJSON implements custom JSON marshaling for SDKControlResponse.
func (r *SDKControlResponse) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"type":       MessageTypeSDKControlResponse,
		"request_id": r.RequestID,
		"response":   r.Response,
	}
	return json.Marshal(data)
}

// UnmarshalJSON implements custom JSON unmarshaling for SDKControlResponse.
func (r *SDKControlResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type      string              `json:"type"`
		RequestID string              `json:"request_id"`
		Response  ControlResponseData `json:"response"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal SDKControlResponse: %w", err)
	}

	r.RequestID = raw.RequestID
	r.Response = raw.Response
	return nil
}

// Request ID generation with atomic counter and random suffix
var requestCounter atomic.Uint64

// GenerateRequestID generates a unique request ID in the format: req_{counter}_{8-char-hex-random}
func GenerateRequestID() string {
	// Increment counter atomically
	counter := requestCounter.Add(1)

	// Generate 4 random bytes for hex suffix (8 hex characters)
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp-based suffix if random fails
		return fmt.Sprintf("req_%d_%d", counter, time.Now().UnixNano()%0xFFFFFFFF)
	}

	hexSuffix := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("req_%d_%s", counter, hexSuffix)
}

// PendingControlResponses manages pending control protocol requests and their responses.
// It provides thread-safe tracking of in-flight requests with timeout support.
type PendingControlResponses struct {
	mu       sync.RWMutex
	pending  map[string]chan interface{}
	timeouts map[string]*time.Timer
}

// NewPendingControlResponses creates a new PendingControlResponses manager.
func NewPendingControlResponses() *PendingControlResponses {
	return &PendingControlResponses{
		pending:  make(map[string]chan interface{}),
		timeouts: make(map[string]*time.Timer),
	}
}

// Register registers a new pending request with a timeout.
// Returns a channel that will receive the response or be closed on timeout.
func (p *PendingControlResponses) Register(requestID string, timeout time.Duration) chan interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	respChan := make(chan interface{}, 1)
	p.pending[requestID] = respChan

	// Set up timeout
	timer := time.AfterFunc(timeout, func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		// Check if still pending (might have been resolved)
		if ch, exists := p.pending[requestID]; exists {
			// Close channel to signal timeout
			close(ch)
			delete(p.pending, requestID)
			delete(p.timeouts, requestID)
		}
	})

	p.timeouts[requestID] = timer

	return respChan
}

// Resolve resolves a pending request with a response.
func (p *PendingControlResponses) Resolve(requestID string, response interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if ch, exists := p.pending[requestID]; exists {
		// Cancel timeout
		if timer, ok := p.timeouts[requestID]; ok {
			timer.Stop()
			delete(p.timeouts, requestID)
		}

		// Send response and close channel
		select {
		case ch <- response:
		default:
			// Channel full or closed, ignore
		}
		close(ch)

		delete(p.pending, requestID)
	}
}

// Wait waits for a response on the given channel with proper error handling.
func (p *PendingControlResponses) Wait(requestID string, respChan chan interface{}) (interface{}, error) {
	response, ok := <-respChan
	if !ok {
		// Channel closed without response - timeout
		return nil, fmt.Errorf("control request timeout: %s", requestID)
	}
	return response, nil
}

// Unregister removes a pending request without resolving it.
// This is useful for cleanup when a request is cancelled.
func (p *PendingControlResponses) Unregister(requestID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if ch, exists := p.pending[requestID]; exists {
		// Cancel timeout
		if timer, ok := p.timeouts[requestID]; ok {
			timer.Stop()
			delete(p.timeouts, requestID)
		}

		close(ch)
		delete(p.pending, requestID)
	}
}

// Len returns the number of pending requests.
func (p *PendingControlResponses) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.pending)
}
