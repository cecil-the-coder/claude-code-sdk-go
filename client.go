package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/severity1/claude-code-sdk-go/internal/cli"
	"github.com/severity1/claude-code-sdk-go/internal/subprocess"
)

const defaultSessionID = "default"

// Client provides bidirectional streaming communication with Claude Code CLI.
type Client interface {
	Connect(ctx context.Context, prompt ...StreamMessage) error
	Disconnect() error
	Query(ctx context.Context, prompt string) error
	QueryWithParentTool(ctx context.Context, prompt string, parentToolUseID *string) error
	QueryWithSession(ctx context.Context, prompt string, sessionID string) error
	QueryWithSessionAndParentTool(ctx context.Context, prompt string, sessionID string, parentToolUseID *string) error
	QueryAsync(ctx context.Context, prompt string) (QueryHandle, error)
	QueryWithSessionAsync(ctx context.Context, prompt string, sessionID string) (QueryHandle, error)
	QueryStream(ctx context.Context, messages <-chan StreamMessage) error
	ReceiveMessages(ctx context.Context) <-chan Message
	ReceiveMessagesWithErrors(ctx context.Context) (<-chan Message, <-chan error)
	ReceiveResponse(ctx context.Context) MessageIterator
	Interrupt(ctx context.Context) error
	SetPermissionMode(ctx context.Context, mode string) error
	SetModel(ctx context.Context, model *string) error
	GetStreamIssues() []StreamIssue
	GetStreamStats() StreamStats
}

// ClientImpl implements the Client interface.
type ClientImpl struct {
	mu              sync.RWMutex
	transport       Transport
	customTransport Transport // For testing with WithTransport
	options         *Options
	connected       bool
	msgChan         <-chan Message
	errChan         <-chan error

	// Active query tracking for async operations
	activeQueries map[string]*queryHandle
	queriesMu     sync.RWMutex

	// Control protocol management
	pendingControlResponses *PendingControlResponses
	controlMu               sync.Mutex

	// Hook system
	hookRegistry *HookRegistry
}

// NewClient creates a new Client with the given options.
func NewClient(opts ...Option) Client {
	options := NewOptions(opts...)
	client := &ClientImpl{
		options:                 options,
		activeQueries:           make(map[string]*queryHandle),
		pendingControlResponses: NewPendingControlResponses(),
		hookRegistry:            NewHookRegistry(),
	}
	return client
}

// NewClientWithTransport creates a new Client with a custom transport (for testing).
func NewClientWithTransport(transport Transport, opts ...Option) Client {
	options := NewOptions(opts...)
	return &ClientImpl{
		customTransport:         transport,
		options:                 options,
		activeQueries:           make(map[string]*queryHandle),
		pendingControlResponses: NewPendingControlResponses(),
		hookRegistry:            NewHookRegistry(),
	}
}

// WithClient provides Go-idiomatic resource management equivalent to Python SDK's async context manager.
// It automatically connects to Claude Code CLI, executes the provided function, and ensures proper cleanup.
// This eliminates the need for manual Connect/Disconnect calls and prevents resource leaks.
//
// The function follows Go's established resource management patterns using defer for guaranteed cleanup,
// similar to how database connections, files, and other resources are typically managed in Go.
//
// Example - Basic usage:
//
//	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
//	    return client.Query(ctx, "What is 2+2?")
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example - With configuration options:
//
//	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
//	    if err := client.Query(ctx, "Calculate the area of a circle with radius 5"); err != nil {
//	        return err
//	    }
//
//	    // Process responses
//	    for msg := range client.ReceiveMessages(ctx) {
//	        if assistantMsg, ok := msg.(*claudecode.AssistantMessage); ok {
//	            fmt.Println("Claude:", assistantMsg.Content[0].(*claudecode.TextBlock).Text)
//	        }
//	    }
//	    return nil
//	}, claudecode.WithSystemPrompt("You are a helpful math tutor"),
//	   claudecode.WithAllowedTools("Read", "Write"))
//
// The client will be automatically connected before fn is called and disconnected after fn returns,
// even if fn returns an error or panics. This provides 100% functional parity with Python SDK's
// 'async with ClaudeSDKClient()' pattern while using idiomatic Go resource management.
//
// Parameters:
//   - ctx: Context for connection management and cancellation
//   - fn: Function to execute with the connected client
//   - opts: Optional client configuration options
//
// Returns an error if connection fails or if fn returns an error.
// Disconnect errors are handled gracefully without overriding the original error from fn.
func WithClient(ctx context.Context, fn func(Client) error, opts ...Option) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	client := NewClient(opts...)

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect client: %w", err)
	}

	defer func() {
		// Following Go idiom: cleanup errors don't override the original error
		// This matches patterns in database/sql, os.File, and other stdlib packages
		if disconnectErr := client.Disconnect(); disconnectErr != nil {
			// Log cleanup errors but don't return them to preserve the original error
			// This follows the standard Go pattern for resource cleanup
			_ = disconnectErr // Explicitly acknowledge we're ignoring this error
		}
	}()

	return fn(client)
}

// WithClientTransport provides Go-idiomatic resource management with a custom transport for testing.
// This is the testing-friendly version of WithClient that accepts an explicit transport parameter.
//
// Usage in tests:
//
//	transport := newClientMockTransport()
//	err := WithClientTransport(ctx, transport, func(client claudecode.Client) error {
//	    return client.Query(ctx, "What is 2+2?")
//	}, opts...)
//
// Parameters:
//   - ctx: Context for connection management and cancellation
//   - transport: Custom transport to use (typically a mock for testing)
//   - fn: Function to execute with the connected client
//   - opts: Optional client configuration options
//
// Returns an error if connection fails or if fn returns an error.
// Disconnect errors are handled gracefully without overriding the original error from fn.
func WithClientTransport(ctx context.Context, transport Transport, fn func(Client) error, opts ...Option) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	client := NewClientWithTransport(transport, opts...)

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect client: %w", err)
	}

	defer func() {
		// Following Go idiom: cleanup errors don't override the original error
		if disconnectErr := client.Disconnect(); disconnectErr != nil {
			// Log cleanup errors but don't return them to preserve the original error
			_ = disconnectErr // Explicitly acknowledge we're ignoring this error
		}
	}()

	return fn(client)
}

// validateOptions validates the client configuration options
func (c *ClientImpl) validateOptions() error {
	if c.options == nil {
		return nil // Nil options are acceptable (use defaults)
	}

	// Validate working directory
	if c.options.Cwd != nil {
		if _, err := os.Stat(*c.options.Cwd); os.IsNotExist(err) {
			return fmt.Errorf("working directory does not exist: %s", *c.options.Cwd)
		}
	}

	// Validate max turns
	if c.options.MaxTurns < 0 {
		return fmt.Errorf("max_turns must be non-negative, got: %d", c.options.MaxTurns)
	}

	// Validate permission mode
	if c.options.PermissionMode != nil {
		validModes := map[PermissionMode]bool{
			PermissionModeDefault:           true,
			PermissionModeAcceptEdits:       true,
			PermissionModePlan:              true,
			PermissionModeBypassPermissions: true,
		}
		if !validModes[*c.options.PermissionMode] {
			return fmt.Errorf("invalid permission mode: %s", string(*c.options.PermissionMode))
		}
	}

	return nil
}

// Connect establishes a connection to the Claude Code CLI.
func (c *ClientImpl) Connect(ctx context.Context, _ ...StreamMessage) error {
	// Check context before acquiring lock
	if ctx.Err() != nil {
		return ctx.Err()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check context again after acquiring lock
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Validate configuration before connecting
	if err := c.validateOptions(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// If CanUseTool callback is set, automatically set permission_prompt_tool_name="stdio"
	// This enables the control protocol for tool permission callbacks
	if c.options.CanUseTool != nil {
		stdio := "stdio"
		c.options.PermissionPromptToolName = &stdio
	}

	// Use custom transport if provided, otherwise create default
	if c.customTransport != nil {
		c.transport = c.customTransport
	} else {
		// Create default subprocess transport directly (like Python SDK)
		cliPath, err := cli.FindCLI()
		if err != nil {
			return fmt.Errorf("claude CLI not found: %w", err)
		}

		// Create subprocess transport for streaming mode (closeStdin=false)
		c.transport = subprocess.New(cliPath, c.options, false, "sdk-go-client")
	}

	// Connect the transport
	if err := c.transport.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect transport: %w", err)
	}

	// Get message channels
	rawMsgChan, rawErrChan := c.transport.ReceiveMessages(ctx)

	// Create filtered channels for control protocol handling
	// Control responses are intercepted and routed to pending manager
	// All other messages pass through to client consumers
	filteredMsgChan := make(chan Message, 10)
	filteredErrChan := make(chan error, 10)

	c.msgChan = filteredMsgChan
	c.errChan = filteredErrChan

	// Start goroutine to route control responses
	go c.routeControlResponses(rawMsgChan, rawErrChan, filteredMsgChan, filteredErrChan)

	c.connected = true

	// Send Initialize request if hooks are configured
	if err := c.initializeHooks(ctx); err != nil {
		// Log error but don't fail connection - hooks are optional
		// In production, you might want to return this error instead
		_ = err
	}

	return nil
}

// Disconnect closes the connection to the Claude Code CLI.
func (c *ClientImpl) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Cancel all active queries before disconnecting
	c.queriesMu.Lock()
	for _, handle := range c.activeQueries {
		handle.Cancel()
	}
	c.activeQueries = make(map[string]*queryHandle)
	c.queriesMu.Unlock()

	if c.transport != nil && c.connected {
		if err := c.transport.Close(); err != nil {
			return fmt.Errorf("failed to close transport: %w", err)
		}
	}
	c.connected = false
	c.transport = nil
	c.msgChan = nil
	c.errChan = nil
	return nil
}

// Query sends a simple text query using the default session.
// This is equivalent to QueryWithSession(ctx, prompt, "default").
//
// Example:
//
//	client.Query(ctx, "What is Go?")
func (c *ClientImpl) Query(ctx context.Context, prompt string) error {
	return c.queryWithSession(ctx, prompt, defaultSessionID)
}

// QueryWithSession sends a simple text query using the specified session ID.
// Each session maintains its own conversation context, allowing for isolated
// conversations within the same client connection.
//
// If sessionID is empty, it defaults to "default".
//
// Example:
//
//	client.QueryWithSession(ctx, "Remember this", "my-session")
//	client.QueryWithSession(ctx, "What did I just say?", "my-session") // Remembers context
//	client.Query(ctx, "What did I just say?")                          // Won't remember, different session
func (c *ClientImpl) QueryWithSession(ctx context.Context, prompt string, sessionID string) error {
	// Use default session if empty session ID provided
	if sessionID == "" {
		sessionID = defaultSessionID
	}
	return c.queryWithSession(ctx, prompt, sessionID)
}

// QueryWithParentTool sends a query with ParentToolUseID for nested tool execution tracking.
// This is used when a tool needs to execute as a child of another tool.
func (c *ClientImpl) QueryWithParentTool(ctx context.Context, prompt string, parentToolUseID *string) error {
	return c.queryWithSessionAndParentTool(ctx, prompt, defaultSessionID, parentToolUseID)
}

// QueryWithSessionAndParentTool sends a query with both session ID and ParentToolUseID.
// This combines session management with tool hierarchy tracking.
func (c *ClientImpl) QueryWithSessionAndParentTool(ctx context.Context, prompt string, sessionID string, parentToolUseID *string) error {
	// Use default session if empty session ID provided
	if sessionID == "" {
		sessionID = defaultSessionID
	}
	return c.queryWithSessionAndParentTool(ctx, prompt, sessionID, parentToolUseID)
}

// queryWithSession is the internal implementation for sending queries with session management.
func (c *ClientImpl) queryWithSession(ctx context.Context, prompt string, sessionID string) error {
	// Check context before proceeding
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	// Check context again after acquiring connection info
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Create user message in Python SDK compatible format
	// Note: session_id is included in JSON for backward compatibility and testing,
	// but the actual CLI requires it as a --session-id command-line flag which must
	// be set via WithSessionID option when creating the client.
	streamMsg := StreamMessage{
		Type: "user",
		Message: map[string]interface{}{
			"role":    "user",
			"content": prompt,
		},
		ParentToolUseID: nil,
		SessionID:       sessionID,
	}

	// Send message via transport (without holding mutex to avoid blocking other operations)
	return transport.SendMessage(ctx, streamMsg)
}

// queryWithSessionAndParentTool is the internal implementation for sending queries with session management and ParentToolUseID.
func (c *ClientImpl) queryWithSessionAndParentTool(ctx context.Context, prompt string, sessionID string, parentToolUseID *string) error {
	// Check context before proceeding
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	// Check context again after acquiring connection info
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Create user message in Python SDK compatible format
	// Note: session_id is included in JSON for backward compatibility and testing,
	// but the actual CLI requires it as a --session-id command-line flag which must
	// be set via WithSessionID option when creating the client.
	streamMsg := StreamMessage{
		Type: "user",
		Message: map[string]interface{}{
			"role":    "user",
			"content": prompt,
		},
		ParentToolUseID: parentToolUseID,
		SessionID:       sessionID,
	}

	// Send message via transport (without holding mutex to avoid blocking other operations)
	return transport.SendMessage(ctx, streamMsg)
}

// QueryStream sends a stream of messages.
func (c *ClientImpl) QueryStream(ctx context.Context, messages <-chan StreamMessage) error {
	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	// Send messages from channel in a goroutine
	go func() {
		for {
			select {
			case msg, ok := <-messages:
				if !ok {
					return // Channel closed
				}
				if err := transport.SendMessage(ctx, msg); err != nil {
					// Log error but continue processing
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// ReceiveMessages returns a channel of incoming messages.
func (c *ClientImpl) ReceiveMessages(_ context.Context) <-chan Message {
	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	msgChan := c.msgChan
	c.mu.RUnlock()

	if !connected || msgChan == nil {
		// Return closed channel if not connected
		closedChan := make(chan Message)
		close(closedChan)
		return closedChan
	}

	// Return the transport's message channel directly
	return msgChan
}

// ReceiveMessagesWithErrors returns both message and error channels from the transport.
// This exposes the underlying transport error channel which was previously hidden,
// allowing callers to handle transport-level errors that occur during message streaming.
//
// The error channel will receive errors from:
//   - Transport I/O failures (stdout/stderr read errors)
//   - Message parsing errors
//   - Process termination errors
//
// Example usage:
//
//	msgChan, errChan := client.ReceiveMessagesWithErrors(ctx)
//	for {
//	    select {
//	    case msg, ok := <-msgChan:
//	        if !ok {
//	            return // Channel closed, done
//	        }
//	        // Process message
//	    case err, ok := <-errChan:
//	        if !ok {
//	            return // Error channel closed
//	        }
//	        // Handle error
//	        log.Printf("Transport error: %v", err)
//	    case <-ctx.Done():
//	        return
//	    }
//	}
func (c *ClientImpl) ReceiveMessagesWithErrors(_ context.Context) (<-chan Message, <-chan error) {
	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	msgChan := c.msgChan
	errChan := c.errChan
	c.mu.RUnlock()

	if !connected || msgChan == nil {
		// Return closed channels if not connected
		closedMsgChan := make(chan Message)
		closedErrChan := make(chan error)
		close(closedMsgChan)
		close(closedErrChan)
		return closedMsgChan, closedErrChan
	}

	// Return both transport channels directly
	return msgChan, errChan
}

// ReceiveResponse returns an iterator for the response messages.
func (c *ClientImpl) ReceiveResponse(_ context.Context) MessageIterator {
	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	msgChan := c.msgChan
	errChan := c.errChan
	c.mu.RUnlock()

	if !connected || msgChan == nil {
		return nil
	}

	// Create a simple iterator over the message channel
	return &clientIterator{
		msgChan: msgChan,
		errChan: errChan,
	}
}

// Interrupt sends an interrupt signal to stop the current operation.
func (c *ClientImpl) Interrupt(ctx context.Context) error {
	// Check context before proceeding
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	return transport.Interrupt(ctx)
}

// SetPermissionMode changes the permission mode at runtime using the control protocol.
// This allows dynamic switching between different permission policies during a conversation.
//
// Valid modes:
//   - "default": Standard interactive permission prompts (default behavior)
//   - "acceptEdits": Automatically accept file edit operations without prompts
//   - "plan": Planning mode - analyze and explain actions without executing them
//   - "bypassPermissions": Bypass all permission checks (use with caution)
//
// The mode change takes effect immediately for future tool calls but does not affect
// in-flight operations. If the CLI rejects the mode (e.g., invalid mode name), an error
// is returned and the previous mode remains active.
//
// This method requires an active bidirectional connection (Client mode) and will fail
// if called in one-shot Query mode.
//
// Example usage:
//
//	client.Connect(ctx)
//	defer client.Disconnect()
//
//	// Start in default mode with manual review
//	client.Query(ctx, "Review my code for issues")
//	// ... review Claude's suggestions ...
//
//	// Switch to acceptEdits for automated fixes
//	client.SetPermissionMode(ctx, "acceptEdits")
//	client.Query(ctx, "Apply the suggested fixes")
//
//	// Switch back to default for next round
//	client.SetPermissionMode(ctx, "default")
func (c *ClientImpl) SetPermissionMode(ctx context.Context, mode string) error {
	// Check context before proceeding
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	// Generate unique request ID
	requestID := GenerateRequestID()

	// Create SetPermissionMode request
	request := &SDKControlRequest{
		RequestID: requestID,
		Request: &SetPermissionModeRequest{
			Mode: mode,
		},
	}

	// Register for response with timeout
	respChan := c.pendingControlResponses.Register(requestID, DefaultControlRequestTimeout)

	// Send control request as a StreamMessage
	// The CLI expects control requests to be sent as StreamMessage with the request as Message field
	streamMsg := StreamMessage{
		Type:    MessageTypeSDKControlRequest,
		Message: request,
	}

	// Send the control request
	if err := transport.SendMessage(ctx, streamMsg); err != nil {
		c.pendingControlResponses.Unregister(requestID)
		return fmt.Errorf("failed to send control request: %w", err)
	}

	// Wait for response
	response, err := c.pendingControlResponses.Wait(requestID, respChan)
	if err != nil {
		return fmt.Errorf("control request timeout: %w", err)
	}

	// Parse response
	controlResp, ok := response.(*SDKControlResponse)
	if !ok {
		return fmt.Errorf("unexpected response type: %T", response)
	}

	// Check if request succeeded
	if !controlResp.Response.Success {
		if controlResp.Response.Error != "" {
			return fmt.Errorf("permission mode change failed: %s", controlResp.Response.Error)
		}
		return fmt.Errorf("permission mode change failed")
	}

	return nil
}

// SetModel switches the model mid-conversation using the control protocol.
// This allows changing the active model between turns while preserving conversation context.
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//   - model: Pointer to model ID string. Supported values:
//   - "claude-sonnet-4-5" - Claude Sonnet 4.5
//   - "claude-opus-4-1-20250805" - Claude Opus 4.1
//   - "claude-3-5-haiku-20241022" - Claude 3.5 Haiku
//   - nil - Use default model
//   - Any CLI-recognized model string
//
// Timing Requirements:
//   - Must be called between turns (after receiving response, before next query)
//   - Pattern: Query() → ReceiveMessages() → SetModel() → Query()
//
// State Preservation:
//   - Conversation context fully preserved across model changes
//   - Session ID and permissions unchanged
//
// Errors:
//   - Returns error if client not connected
//   - Returns error if model switch fails (e.g., invalid model)
//   - Returns error on context cancellation or timeout (60s default)
//
// Example - Switch to Opus for complex task:
//
//	// Start with default model
//	client.Query(ctx, "Analyze this codebase")
//	// ... process responses ...
//
//	// Switch to Opus for implementation
//	opus := "claude-opus-4-1-20250805"
//	client.SetModel(ctx, &opus)
//
//	client.Query(ctx, "Implement the solution")
func (c *ClientImpl) SetModel(ctx context.Context, model *string) error {
	// Check context before proceeding
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	// Determine model string for request (empty string for nil = default)
	modelStr := ""
	if model != nil {
		modelStr = *model
	}

	// Generate unique request ID
	requestID := GenerateRequestID()

	// Create SetModel control request
	request := &SDKControlRequest{
		RequestID: requestID,
		Request: &SetModelRequest{
			Model: modelStr,
		},
	}

	// Register for response with 60s timeout
	respChan := c.pendingControlResponses.Register(requestID, DefaultControlRequestTimeout)

	// Send control request as StreamMessage
	// The CLI expects control requests as StreamMessage with request as Message field
	streamMsg := StreamMessage{
		Type:    MessageTypeSDKControlRequest,
		Message: request,
	}

	// Send the control request
	if err := transport.SendMessage(ctx, streamMsg); err != nil {
		c.pendingControlResponses.Unregister(requestID)
		return fmt.Errorf("failed to send control request: %w", err)
	}

	// Wait for response
	response, err := c.pendingControlResponses.Wait(requestID, respChan)
	if err != nil {
		return fmt.Errorf("control request timeout: %w", err)
	}

	// Parse response
	controlResp, ok := response.(*SDKControlResponse)
	if !ok {
		return fmt.Errorf("unexpected response type: %T", response)
	}

	// Check if request succeeded
	if !controlResp.Response.Success {
		if controlResp.Response.Error != "" {
			return fmt.Errorf("model change failed: %s", controlResp.Response.Error)
		}
		return fmt.Errorf("model change failed")
	}

	return nil
}

// clientIterator implements MessageIterator for client message reception
type clientIterator struct {
	msgChan <-chan Message
	errChan <-chan error
	closed  bool
}

func (ci *clientIterator) Next(ctx context.Context) (Message, error) {
	if ci.closed {
		return nil, ErrNoMoreMessages
	}

	select {
	case msg, ok := <-ci.msgChan:
		if !ok {
			ci.closed = true
			return nil, ErrNoMoreMessages
		}
		return msg, nil
	case err := <-ci.errChan:
		ci.closed = true
		return nil, err
	case <-ctx.Done():
		ci.closed = true
		return nil, ctx.Err()
	}
}

func (ci *clientIterator) Close() error {
	ci.closed = true
	return nil
}

// GetStreamIssues returns validation issues found in the message stream.
// This can help diagnose problems like missing tool results or incomplete streams.
func (c *ClientImpl) GetStreamIssues() []StreamIssue {
	c.mu.RLock()
	transport := c.transport
	c.mu.RUnlock()

	if transport == nil {
		return nil
	}

	validator := transport.GetValidator()
	if validator == nil {
		return nil
	}

	return validator.GetIssues()
}

// GetStreamStats returns statistics about the message stream.
// This includes counts of tools requested/received and pending tools.
func (c *ClientImpl) GetStreamStats() StreamStats {
	c.mu.RLock()
	transport := c.transport
	c.mu.RUnlock()

	if transport == nil {
		return StreamStats{}
	}

	validator := transport.GetValidator()
	if validator == nil {
		return StreamStats{}
	}

	return validator.GetStats()
}

// routeControlResponses monitors incoming messages and routes control protocol responses
// to the pending control responses manager while passing other messages through to consumers.
// It also handles incoming control requests from the CLI (like can_use_tool).
func (c *ClientImpl) routeControlResponses(rawMsgChan <-chan Message, rawErrChan <-chan error,
	filteredMsgChan chan<- Message, filteredErrChan chan<- error) {

	defer close(filteredMsgChan)
	defer close(filteredErrChan)

	for {
		select {
		case msg, ok := <-rawMsgChan:
			if !ok {
				// Channel closed, exit goroutine
				return
			}

			// Check if this is a control response
			if controlResp, ok := msg.(*SDKControlResponse); ok {
				// Route to pending control responses manager
				c.pendingControlResponses.Resolve(controlResp.RequestID, controlResp)
			} else if controlReq, ok := msg.(*SDKControlRequest); ok {
				// Handle incoming control request from CLI
				c.handleControlRequest(controlReq)
			} else {
				// Pass through to filtered channel for normal consumers
				select {
				case filteredMsgChan <- msg:
				default:
					// Channel full, drop message (shouldn't happen with proper buffering)
				}
			}

		case err, ok := <-rawErrChan:
			if !ok {
				// Channel closed, exit goroutine
				return
			}

			// Pass through errors
			select {
			case filteredErrChan <- err:
			default:
				// Channel full, drop error (shouldn't happen with proper buffering)
			}
		}
	}
}

// handleControlRequest processes incoming control requests from the CLI.
func (c *ClientImpl) handleControlRequest(request *SDKControlRequest) {
	// Create context with timeout for callback execution
	ctx, cancel := context.WithTimeout(context.Background(), DefaultControlRequestTimeout)
	defer cancel()

	var response *SDKControlResponse

	switch req := request.Request.(type) {
	case *CanUseToolRequest:
		response = c.handleCanUseToolRequest(ctx, request.RequestID, req)
	case *HookCallbackRequest:
		response = c.handleHookCallbackRequest(ctx, request.RequestID, req)
	default:
		// Unknown request type - send error response
		response = &SDKControlResponse{
			RequestID: request.RequestID,
			Response: ControlResponseData{
				Success: false,
				Error:   fmt.Sprintf("unknown control request type: %T", request.Request),
			},
		}
	}

	// Send response back to CLI
	if response != nil {
		c.sendControlResponse(ctx, response)
	}
}

// handleCanUseToolRequest processes can_use_tool requests from the CLI.
func (c *ClientImpl) handleCanUseToolRequest(ctx context.Context, requestID string, req *CanUseToolRequest) *SDKControlResponse {
	// Check if callback is configured
	if c.options.CanUseTool == nil {
		return &SDKControlResponse{
			RequestID: requestID,
			Response: ControlResponseData{
				Success: false,
				Error:   "CanUseTool callback not configured",
			},
		}
	}

	// Parse suggestions from the request
	var suggestions []PermissionUpdate
	if req.Suggestions != nil {
		// req.Suggestions is []string in the request, but we need []PermissionUpdate
		// For now, we'll leave suggestions empty since the CLI sends them in a different format
		// In a real implementation, you'd need to parse the suggestion strings
		suggestions = []PermissionUpdate{}
	}

	// Create tool permission context
	toolContext := ToolPermissionContext{
		Suggestions: suggestions,
	}

	// Call the user's callback
	result, err := c.options.CanUseTool(ctx, req.ToolName, req.Input, toolContext)
	if err != nil {
		return &SDKControlResponse{
			RequestID: requestID,
			Response: ControlResponseData{
				Success: false,
				Error:   fmt.Sprintf("callback error: %v", err),
			},
		}
	}

	// Build response based on permission result
	responseData := make(map[string]interface{})
	responseData["behavior"] = result.Behavior()

	switch r := result.(type) {
	case *PermissionResultAllow:
		if r.UpdatedInput != nil {
			responseData["updated_input"] = r.UpdatedInput
		}
		if r.UpdatedPermissions != nil {
			responseData["updated_permissions"] = r.UpdatedPermissions
		}
	case *PermissionResultDeny:
		if r.Message != "" {
			responseData["message"] = r.Message
		}
		if r.Interrupt {
			responseData["interrupt"] = r.Interrupt
		}
	}

	return &SDKControlResponse{
		RequestID: requestID,
		Response: ControlResponseData{
			Success: true,
			Result:  responseData,
		},
	}
}

// sendControlResponse sends a control protocol response to the CLI.
func (c *ClientImpl) sendControlResponse(ctx context.Context, response *SDKControlResponse) {
	c.mu.RLock()
	transport := c.transport
	connected := c.connected
	c.mu.RUnlock()

	if !connected || transport == nil {
		return
	}

	// Create stream message with control response
	streamMsg := StreamMessage{
		Type:    MessageTypeSDKControlResponse,
		Message: response,
	}

	// Send the response (errors are logged but not returned since this is async)
	_ = transport.SendMessage(ctx, streamMsg)
}

// initializeHooks sends an Initialize request to register hooks with the CLI.
func (c *ClientImpl) initializeHooks(ctx context.Context) error {
	// Check if any hooks are configured
	if c.options == nil || c.options.Hooks == nil || len(c.options.Hooks) == 0 {
		return nil // No hooks to register
	}

	// Build list of hook event names and callbacks
	var hookNames []string
	var callbacks []CallbackInfo

	for eventType, matcher := range c.options.Hooks {
		if matcher == nil || len(matcher.Hooks) == 0 {
			continue
		}

		// Add event type to hooks list
		hookNames = append(hookNames, string(eventType))

		// Register callbacks and get their IDs
		callbackIDs := c.hookRegistry.RegisterHooks(eventType, matcher)

		// Add callback info
		for _, id := range callbackIDs {
			callbacks = append(callbacks, CallbackInfo{
				ID:   id,
				Name: fmt.Sprintf("%s_callback", eventType),
			})
		}
	}

	// If no hooks were actually registered, return early
	if len(hookNames) == 0 {
		return nil
	}

	// Create Initialize request
	initRequest := &InitializeRequest{
		Hooks:     hookNames,
		Callbacks: callbacks,
	}

	// Send control request
	request := &SDKControlRequest{
		RequestID: GenerateRequestID(),
		Request:   initRequest,
	}

	// Register pending response
	respChan := c.pendingControlResponses.Register(request.RequestID, DefaultControlRequestTimeout)

	// Send request
	streamMsg := StreamMessage{
		Type:    MessageTypeSDKControlRequest,
		Message: request,
	}

	c.mu.RLock()
	transport := c.transport
	c.mu.RUnlock()

	if err := transport.SendMessage(ctx, streamMsg); err != nil {
		c.pendingControlResponses.Unregister(request.RequestID)
		return fmt.Errorf("failed to send initialize request: %w", err)
	}

	// Wait for response
	response, err := c.pendingControlResponses.Wait(request.RequestID, respChan)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	// Check response
	controlResp, ok := response.(*SDKControlResponse)
	if !ok {
		return fmt.Errorf("unexpected response type: %T", response)
	}

	if !controlResp.Response.Success {
		return fmt.Errorf("initialize request failed: %s", controlResp.Response.Error)
	}

	return nil
}

// handleHookCallbackRequest processes hook_callback requests from the CLI.
func (c *ClientImpl) handleHookCallbackRequest(ctx context.Context, requestID string, req *HookCallbackRequest) *SDKControlResponse {
	// Get the callback from registry
	callback := c.hookRegistry.GetCallback(req.CallbackID)
	if callback == nil {
		return &SDKControlResponse{
			RequestID: requestID,
			Response: ControlResponseData{
				Success: false,
				Error:   fmt.Sprintf("hook callback not found: %s", req.CallbackID),
			},
		}
	}

	// Parse HookInput from the request input
	var hookInput HookInput

	// Convert map[string]interface{} to HookInput
	// The CLI sends the hook input as a map, we need to parse it properly
	if req.Input != nil {
		// Marshal and unmarshal to convert map to struct
		inputBytes, err := json.Marshal(req.Input)
		if err != nil {
			return &SDKControlResponse{
				RequestID: requestID,
				Response: ControlResponseData{
					Success: false,
					Error:   fmt.Sprintf("failed to marshal hook input: %v", err),
				},
			}
		}

		if err := json.Unmarshal(inputBytes, &hookInput); err != nil {
			return &SDKControlResponse{
				RequestID: requestID,
				Response: ControlResponseData{
					Success: false,
					Error:   fmt.Sprintf("failed to unmarshal hook input: %v", err),
				},
			}
		}
	}

	// Parse tool use ID
	var toolUseID *string
	if req.ToolUseID != "" {
		toolUseID = &req.ToolUseID
	}

	// Create hook context
	hookContext := HookContext{
		AdditionalData: make(map[string]any),
	}

	// Call the user's hook callback
	result, err := callback(ctx, hookInput, toolUseID, hookContext)
	if err != nil {
		return &SDKControlResponse{
			RequestID: requestID,
			Response: ControlResponseData{
				Success: false,
				Error:   fmt.Sprintf("hook callback error: %v", err),
			},
		}
	}

	// Return the hook output as the result
	return &SDKControlResponse{
		RequestID: requestID,
		Response: ControlResponseData{
			Success: true,
			Result:  result,
		},
	}
}
