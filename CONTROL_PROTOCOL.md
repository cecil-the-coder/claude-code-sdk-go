# Control Protocol API Documentation

The Control Protocol enables bidirectional communication between the SDK and Claude Code CLI, allowing for hook callbacks, permission requests, and dynamic configuration changes.

## Overview

The control protocol provides:
- **Hook Callbacks**: Register and execute custom hooks during tool execution
- **Permission Requests**: Dynamic permission checks via `can_use_tool` callbacks
- **Runtime Configuration**: Change permission modes and models mid-conversation
- **MCP Message Routing**: Route messages to MCP servers
- **Operation Control**: Interrupt and cancel operations

## Message Types

### SDKControlRequest

Represents a control protocol request from SDK to CLI.

```go
type SDKControlRequest struct {
    RequestID string         // Format: req_{counter}_{8-char-hex-random}
    Request   ControlRequest // Discriminated union of request types
}
```

### SDKControlResponse

Represents a control protocol response from CLI to SDK.

```go
type SDKControlResponse struct {
    RequestID string
    Response  ControlResponseData
}

type ControlResponseData struct {
    Success bool
    Result  map[string]interface{} // Present on success
    Error   string                 // Present on error
}
```

## Request Types

### 1. InitializeRequest

Register hooks and callbacks with the CLI.

```go
request := &claudecode.InitializeRequest{
    Hooks: []string{"can_use_tool"},
    Callbacks: []claudecode.CallbackInfo{
        {ID: "cb_1", Name: "permission_handler"},
    },
}
```

**Fields:**
- `Hooks`: Array of hook names to register (e.g., `"can_use_tool"`)
- `Callbacks`: Array of callback information

### 2. CanUseToolRequest

Request permission to use a specific tool.

```go
request := &claudecode.CanUseToolRequest{
    ToolName: "Read",
    Input: map[string]interface{}{
        "file_path": "/etc/passwd",
    },
    Suggestions: []string{"allow", "deny"},
}
```

**Fields:**
- `ToolName`: Name of the tool requiring permission
- `Input`: Tool input parameters
- `Suggestions`: Optional array of suggested actions

### 3. HookCallbackRequest

Execute a registered hook callback.

```go
request := &claudecode.HookCallbackRequest{
    CallbackID: "cb_1",
    Input: map[string]interface{}{
        "tool": "Read",
        "params": map[string]interface{}{},
    },
    ToolUseID: "tool_abc123",
}
```

**Fields:**
- `CallbackID`: ID of the registered callback
- `Input`: Callback input data
- `ToolUseID`: Associated tool use ID

### 4. SetPermissionModeRequest

Change the permission mode at runtime.

```go
request := &claudecode.SetPermissionModeRequest{
    Mode: "bypass_permissions",
}
```

**Valid Modes:**
- `"default"` - Default permission behavior
- `"accept_edits"` - Auto-accept edit operations
- `"plan"` - Planning mode
- `"bypass_permissions"` - Bypass all permission checks

### 5. SetModelRequest

Switch the model mid-conversation.

```go
request := &claudecode.SetModelRequest{
    Model: "claude-opus-4-5-20251101",
}
```

**Fields:**
- `Model`: Model identifier to switch to

### 6. InterruptRequest

Cancel the current operation.

```go
request := &claudecode.InterruptRequest{}
```

### 7. MCPMessageRequest

Route messages to MCP servers.

```go
request := &claudecode.MCPMessageRequest{
    ServerName: "filesystem",
    Message: map[string]interface{}{
        "method": "read_file",
        "params": map[string]interface{}{
            "path": "/data/file.txt",
        },
    },
}
```

**Fields:**
- `ServerName`: Name of the MCP server
- `Message`: Message payload to send

## Request/Response Flow

### Basic Pattern

```go
import (
    "context"
    "time"

    "github.com/severity1/claude-code-sdk-go"
)

func handleControlProtocol(ctx context.Context) error {
    // 1. Create pending response manager
    mgr := claudecode.NewPendingControlResponses()

    // 2. Generate unique request ID
    requestID := claudecode.GenerateRequestID()

    // 3. Register pending request with timeout
    respChan := mgr.Register(requestID, claudecode.DefaultControlRequestTimeout)

    // 4. Create and send control request
    request := &claudecode.SDKControlRequest{
        RequestID: requestID,
        Request: &claudecode.CanUseToolRequest{
            ToolName: "Read",
            Input: map[string]interface{}{
                "file_path": "/test.txt",
            },
        },
    }

    // Send request via transport (implementation specific)
    // transport.SendControlRequest(ctx, request)

    // 5. Wait for response with timeout
    response, err := mgr.Wait(requestID, respChan)
    if err != nil {
        return err // Timeout or other error
    }

    // 6. Handle response
    respData := response.(*claudecode.ControlResponseData)
    if !respData.Success {
        return fmt.Errorf("request failed: %s", respData.Error)
    }

    // Process successful result
    allowed := respData.Result["allowed"].(bool)
    return nil
}
```

## Request ID Format

Request IDs follow the format: `req_{counter}_{8-char-hex-random}`

- `counter`: Monotonically increasing atomic counter
- `8-char-hex-random`: 8 hexadecimal characters from 4 random bytes

**Example:** `req_42_a1b2c3d4`

### Generation

```go
requestID := claudecode.GenerateRequestID()
// Returns: "req_1_f3e2d1c0"
```

## Timeouts

The default timeout for control requests is **60 seconds** (`DefaultControlRequestTimeout`).

Custom timeouts can be specified when registering a request:

```go
mgr := claudecode.NewPendingControlResponses()
respChan := mgr.Register(requestID, 30*time.Second) // 30 second timeout
```

## Concurrent Requests

The control protocol supports concurrent requests through request ID matching:

```go
mgr := claudecode.NewPendingControlResponses()

// Send multiple requests concurrently
for i := 0; i < 5; i++ {
    go func(idx int) {
        requestID := claudecode.GenerateRequestID()
        respChan := mgr.Register(requestID, claudecode.DefaultControlRequestTimeout)

        // Send request and wait for response
        // ...
    }(i)
}
```

## PendingControlResponses Manager

Thread-safe manager for tracking in-flight control requests.

### Methods

#### Register
Register a new pending request with a timeout.

```go
func (p *PendingControlResponses) Register(requestID string, timeout time.Duration) chan interface{}
```

**Returns:** Channel that will receive the response or be closed on timeout.

#### Resolve
Resolve a pending request with a response.

```go
func (p *PendingControlResponses) Resolve(requestID string, response interface{})
```

#### Wait
Wait for a response on the given channel.

```go
func (p *PendingControlResponses) Wait(requestID string, respChan chan interface{}) (interface{}, error)
```

**Returns:** Response data or timeout error.

#### Unregister
Remove a pending request without resolving it (cleanup/cancellation).

```go
func (p *PendingControlResponses) Unregister(requestID string)
```

#### Len
Return the number of pending requests.

```go
func (p *PendingControlResponses) Len() int
```

## Message Parsing

Control protocol messages are automatically parsed by the SDK's message parser when received from the CLI:

```go
// Parser automatically discriminates control protocol messages
parser := parser.New()
messages, err := parser.ProcessLine(jsonLine)

for _, msg := range messages {
    switch m := msg.(type) {
    case *claudecode.SDKControlRequest:
        // Handle incoming control request
        handleControlRequest(m)
    case *claudecode.SDKControlResponse:
        // Handle control response
        handleControlResponse(m)
    }
}
```

## Constants

### Message Type Constants

```go
const (
    MessageTypeSDKControlRequest  = "sdk_control_request"
    MessageTypeSDKControlResponse = "sdk_control_response"
)
```

### Control Request Type Constants

```go
const (
    ControlRequestTypeInitialize        = "initialize"
    ControlRequestTypeCanUseTool        = "can_use_tool"
    ControlRequestTypeHookCallback      = "hook_callback"
    ControlRequestTypeSetPermissionMode = "set_permission_mode"
    ControlRequestTypeSetModel          = "set_model"
    ControlRequestTypeInterrupt         = "interrupt"
    ControlRequestTypeMCPMessage        = "mcp_message"
)
```

### Timeout Constants

```go
const DefaultControlRequestTimeout = 60 * time.Second
```

## Error Handling

### Timeout Errors

```go
response, err := mgr.Wait(requestID, respChan)
if err != nil {
    if strings.Contains(err.Error(), "timeout") {
        // Handle timeout
        log.Printf("Control request timed out: %s", requestID)
    }
}
```

### Response Errors

```go
respData := response.(*claudecode.ControlResponseData)
if !respData.Success {
    log.Printf("Control request failed: %s", respData.Error)
    return fmt.Errorf("control request error: %s", respData.Error)
}
```

## Thread Safety

All control protocol types and managers are thread-safe:

- `GenerateRequestID()` uses atomic counter for concurrent safety
- `PendingControlResponses` uses mutex protection for concurrent access
- Request/response channels are goroutine-safe

## Best Practices

1. **Always use unique request IDs**: Use `GenerateRequestID()` for guaranteed uniqueness
2. **Set appropriate timeouts**: Consider operation complexity when setting timeouts
3. **Clean up on cancellation**: Use `Unregister()` when cancelling operations
4. **Handle all response types**: Check both success and error responses
5. **Monitor pending requests**: Use `Len()` to track in-flight requests for debugging

## Example: Permission Hook

Complete example of implementing a permission hook:

```go
func setupPermissionHook(ctx context.Context, client claudecode.Client) error {
    mgr := claudecode.NewPendingControlResponses()

    // 1. Initialize hook
    initID := claudecode.GenerateRequestID()
    initChan := mgr.Register(initID, claudecode.DefaultControlRequestTimeout)

    initReq := &claudecode.SDKControlRequest{
        RequestID: initID,
        Request: &claudecode.InitializeRequest{
            Hooks: []string{"can_use_tool"},
            Callbacks: []claudecode.CallbackInfo{
                {ID: "perm_check", Name: "permission_handler"},
            },
        },
    }

    // Send initialization request
    // (implementation depends on transport layer)

    // Wait for confirmation
    _, err := mgr.Wait(initID, initChan)
    if err != nil {
        return fmt.Errorf("failed to initialize hook: %w", err)
    }

    // 2. Hook is now active - handle can_use_tool callbacks
    // (received as SDKControlRequest messages)

    return nil
}
```

## Integration with Transport Layer

The control protocol integrates with the Transport interface for sending/receiving control messages. Implementation details depend on the specific transport implementation (subprocess, mock, etc.).

For transport integration details, see the Transport interface documentation and implementation in `internal/subprocess/transport.go`.
