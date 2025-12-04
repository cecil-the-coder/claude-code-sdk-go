# Claude Code SDK Enhancements for Message Queueing Support

**Date**: 2025-11-10
**SDK Repository**: `/tmp/claude-code-sdk-go`
**Current Version**: v0.2.5
**Status**: Proposal for SDK Enhancement

---

## Executive Summary

The current `claude-code-sdk-go` provides **synchronous, blocking message sending** with no built-in queueing support. To enable application-level message queueing similar to Claude CLI's h2A queue, the SDK needs enhancements in the following areas:

1. **Non-blocking message submission** API
2. **Message queue management** primitives
3. **Session isolation** improvements
4. **Interruption/cancellation** support
5. **Queue state inspection** APIs

This document outlines specific SDK enhancements needed, along with API proposals and implementation guidance.

---

## Current SDK Limitations

### 1. Blocking Message Submission

**Current Implementation (`client.go:293-327`):**
```go
func (c *ClientImpl) QueryWithSession(ctx context.Context, prompt string, sessionID string) error {
    // Creates message
    streamMsg := StreamMessage{
        Type: "user",
        Message: map[string]interface{}{
            "role":    "user",
            "content": prompt,
        },
        SessionID: sessionID,
    }

    // BLOCKS until message is sent
    if err := c.transport.SendMessage(ctx, streamMsg); err != nil {
        return err
    }

    // BLOCKS until all responses are received
    for {
        select {
        case msg := <-c.msgChan:
            // Process message
        case err := <-c.errChan:
            return err
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

**Problem:** Cannot submit a new message while previous query is processing.

### 2. No Queue Primitives

**Current Client Structure:**
```go
type ClientImpl struct {
    transport       Transport
    msgChan         <-chan Message
    errChan         <-chan error
    // No queue fields!
}
```

**Problem:** Applications must implement their own queue management with no SDK support.

### 3. Unreliable Session Isolation

**Testing Shows:** (from `/home/vscode/goagent/webserver/server.go:351-357`)
```go
// CRITICAL: QueryWithSession() does NOT properly isolate conversations:
// - Created "Blue" conversation, told it "My favorite color is blue"
// - Created "Red" conversation, told it "My favorite color is red"
// - Asked "Blue" conversation "What's my favorite color?" → answered "Red" (WRONG!)
```

**Problem:** Session IDs are sent to CLI but subprocess shares state.

### 4. No Interruption Support

**Current Transport (`internal/subprocess/transport.go:224-265`):**
```go
func (t *Transport) SendMessage(ctx context.Context, message shared.StreamMessage) error {
    data, err := json.Marshal(message)
    _, err = t.stdin.Write(append(data, '\n'))
    return nil
}
```

**Problem:** Once a message is sent, there's no way to interrupt or cancel processing.

---

## Proposed SDK Enhancements

### Enhancement 1: Non-Blocking Message API

#### Proposed API

```go
// QueryAsync submits a message without blocking
// Returns a QueryHandle that can be used to monitor/control the query
func (c *ClientImpl) QueryAsync(ctx context.Context, prompt string) (*QueryHandle, error)

// QueryWithSessionAsync submits a message to a specific session without blocking
func (c *ClientImpl) QueryWithSessionAsync(ctx context.Context, prompt string, sessionID string) (*QueryHandle, error)

// QueryHandle represents an in-flight query
type QueryHandle struct {
    ID        string
    SessionID string
    Status    QueryStatus
    Messages  <-chan Message
    Errors    <-chan error
    Cancel    context.CancelFunc
}

// QueryStatus represents the current state of a query
type QueryStatus int

const (
    QueryStatusQueued QueryStatus = iota
    QueryStatusProcessing
    QueryStatusCompleted
    QueryStatusFailed
    QueryStatusCancelled
)

// Methods on QueryHandle
func (qh *QueryHandle) Wait() error
func (qh *QueryHandle) GetStatus() QueryStatus
func (qh *QueryHandle) Cancel() error
```

#### Implementation Example

```go
// client.go - New async methods
func (c *ClientImpl) QueryWithSessionAsync(ctx context.Context, prompt string, sessionID string) (*QueryHandle, error) {
    if sessionID == "" {
        sessionID = defaultSessionID
    }

    // Create handle with channels
    handle := &QueryHandle{
        ID:        generateID(),
        SessionID: sessionID,
        Status:    QueryStatusQueued,
        Messages:  make(chan Message, 100),
        Errors:    make(chan error, 1),
    }

    // Create cancelable context
    handleCtx, cancel := context.WithCancel(ctx)
    handle.Cancel = cancel

    // Start query in goroutine (non-blocking)
    go c.executeQueryAsync(handleCtx, prompt, sessionID, handle)

    return handle, nil
}

func (c *ClientImpl) executeQueryAsync(ctx context.Context, prompt string, sessionID string, handle *QueryHandle) {
    // Update status
    handle.Status = QueryStatusProcessing

    // Execute query (same logic as current Query)
    streamMsg := StreamMessage{
        Type: "user",
        Message: map[string]interface{}{
            "role":    "user",
            "content": prompt,
        },
        SessionID: sessionID,
    }

    if err := c.transport.SendMessage(ctx, streamMsg); err != nil {
        handle.Status = QueryStatusFailed
        handle.Errors <- err
        return
    }

    // Stream responses
    for {
        select {
        case msg := <-c.msgChan:
            handle.Messages <- msg
            if msg.Type == "done" {
                handle.Status = QueryStatusCompleted
                close(handle.Messages)
                return
            }
        case err := <-c.errChan:
            handle.Status = QueryStatusFailed
            handle.Errors <- err
            return
        case <-ctx.Done():
            handle.Status = QueryStatusCancelled
            return
        }
    }
}
```

#### Usage Example

```go
// Application code
client := claudecode.NewClient(...)

// Submit multiple queries without blocking
handle1, _ := client.QueryAsync(ctx, "What is 2+2?")
handle2, _ := client.QueryAsync(ctx, "What is 3+3?")
handle3, _ := client.QueryAsync(ctx, "What is 4+4?")

// Process responses as they arrive
for msg := range handle1.Messages {
    fmt.Println("Response 1:", msg)
}

// Or wait for completion
if err := handle2.Wait(); err != nil {
    log.Fatal(err)
}
```

---

### Enhancement 2: Built-In Queue Manager

#### Proposed API

```go
// QueueManager manages message queues for multiple sessions
type QueueManager struct {
    queues map[string]*MessageQueue
    mu     sync.RWMutex
}

// MessageQueue represents a queue for a specific session
type MessageQueue struct {
    sessionID   string
    messages    []QueuedMessage
    processing  *QueuedMessage
    mu          sync.RWMutex
}

// QueuedMessage represents a message in the queue
type QueuedMessage struct {
    ID        string
    Content   string
    Priority  int
    Timestamp time.Time
    Handle    *QueryHandle
}

// Queue management methods
func (qm *QueueManager) Enqueue(sessionID, message string) (*QueueHandle, error)
func (qm *QueueManager) GetQueueStatus(sessionID string) (*QueueStatus, error)
func (qm *QueueManager) GetQueueLength(sessionID string) int
func (qm *QueueManager) ClearQueue(sessionID string) error
func (qm *QueueManager) RemoveFromQueue(sessionID, messageID string) error

// QueueHandle represents a queued message
type QueueHandle struct {
    MessageID string
    Position  int
    EstimatedWait time.Duration
    Status    MessageStatus
}

type MessageStatus int

const (
    MessageStatusQueued MessageStatus = iota
    MessageStatusProcessing
    MessageStatusCompleted
    MessageStatusFailed
    MessageStatusCancelled
)
```

#### Implementation Example

```go
// queue.go - New file in SDK
package claudecode

import (
    "context"
    "sync"
    "time"
)

type QueueManager struct {
    client *ClientImpl
    queues map[string]*MessageQueue
    mu     sync.RWMutex
}

func NewQueueManager(client *ClientImpl) *QueueManager {
    return &QueueManager{
        client: client,
        queues: make(map[string]*MessageQueue),
    }
}

func (qm *QueueManager) Enqueue(ctx context.Context, sessionID, message string) (*QueueHandle, error) {
    qm.mu.Lock()
    queue, exists := qm.queues[sessionID]
    if !exists {
        queue = &MessageQueue{
            sessionID: sessionID,
            messages:  make([]QueuedMessage, 0),
        }
        qm.queues[sessionID] = queue

        // Start processor goroutine
        go qm.processQueue(ctx, queue)
    }
    qm.mu.Unlock()

    // Add message to queue
    queuedMsg := QueuedMessage{
        ID:        generateID(),
        Content:   message,
        Timestamp: time.Now(),
    }

    queue.mu.Lock()
    queue.messages = append(queue.messages, queuedMsg)
    position := len(queue.messages)
    queue.mu.Unlock()

    return &QueueHandle{
        MessageID: queuedMsg.ID,
        Position:  position,
        Status:    MessageStatusQueued,
    }, nil
}

func (qm *QueueManager) processQueue(ctx context.Context, queue *MessageQueue) {
    for {
        queue.mu.Lock()
        if len(queue.messages) == 0 {
            queue.mu.Unlock()
            time.Sleep(100 * time.Millisecond)
            continue
        }

        // Get next message
        msg := queue.messages[0]
        queue.messages = queue.messages[1:]
        queue.processing = &msg
        queue.mu.Unlock()

        // Process message
        handle, err := qm.client.QueryWithSessionAsync(ctx, msg.Content, queue.sessionID)
        if err != nil {
            log.Printf("Error processing queued message: %v", err)
            continue
        }

        // Wait for completion
        handle.Wait()

        queue.mu.Lock()
        queue.processing = nil
        queue.mu.Unlock()
    }
}

func (qm *QueueManager) GetQueueLength(sessionID string) int {
    qm.mu.RLock()
    defer qm.mu.RUnlock()

    queue, exists := qm.queues[sessionID]
    if !exists {
        return 0
    }

    queue.mu.RLock()
    defer queue.mu.RUnlock()

    return len(queue.messages)
}
```

#### Usage Example

```go
// Application code
client := claudecode.NewClient(...)
queueManager := claudecode.NewQueueManager(client)

// Enqueue messages (non-blocking)
handle1, _ := queueManager.Enqueue(ctx, "session-1", "Message 1")
handle2, _ := queueManager.Enqueue(ctx, "session-1", "Message 2")
handle3, _ := queueManager.Enqueue(ctx, "session-1", "Message 3")

// Check queue status
queueLength := queueManager.GetQueueLength("session-1")
fmt.Printf("Queue length: %d\n", queueLength)

// Messages are processed sequentially in the background
```

---

### Enhancement 3: Session Isolation Fix

#### Current Problem

Session IDs are sent to Claude CLI subprocess, but the subprocess doesn't properly isolate state between sessions.

#### Root Cause Analysis

```go
// Current implementation sends session ID but subprocess ignores it
streamMsg := StreamMessage{
    Type: "user",
    Message: map[string]interface{}{
        "role":    "user",
        "content": prompt,
    },
    SessionID: sessionID,  // ← Sent but not respected by subprocess
}
```

#### Proposed Solutions

**Option A: Multiple Subprocesses (Recommended)**

Create one Claude CLI subprocess per session:

```go
type ClientImpl struct {
    transport       Transport
    msgChan         <-chan Message
    errChan         <-chan error
    sessionID       string  // NEW: Dedicated session
}

// Factory method to create client per session
func NewClientForSession(options ClientOptions, sessionID string) (Client, error) {
    // Each client gets its own subprocess
    transport := subprocess.NewTransport(options)

    return &ClientImpl{
        transport: transport,
        msgChan:   transport.MessageChan(),
        errChan:   transport.ErrorChan(),
        sessionID: sessionID,
    }, nil
}
```

**Option B: Session Manager**

Manage multiple clients internally:

```go
type SessionManager struct {
    clients map[string]*ClientImpl
    mu      sync.RWMutex
}

func (sm *SessionManager) GetClient(sessionID string) (*ClientImpl, error) {
    sm.mu.RLock()
    client, exists := sm.clients[sessionID]
    sm.mu.RUnlock()

    if exists {
        return client, nil
    }

    // Create new client for session
    sm.mu.Lock()
    defer sm.mu.Unlock()

    newClient, err := NewClientForSession(sm.options, sessionID)
    if err != nil {
        return nil, err
    }

    sm.clients[sessionID] = newClient
    return newClient, nil
}

func (sm *SessionManager) Query(ctx context.Context, sessionID, prompt string) error {
    client, err := sm.GetClient(sessionID)
    if err != nil {
        return err
    }

    return client.Query(ctx, prompt)
}
```

#### Implementation Changes Required

**File: `client.go`**
```go
// Add session-specific client creation
func NewClientForSession(options ClientOptions, sessionID string) (Client, error) {
    // Implementation...
}

// Add SessionManager
type SessionManager struct {
    options ClientOptions
    clients map[string]*ClientImpl
    mu      sync.RWMutex
}

func NewSessionManager(options ClientOptions) *SessionManager {
    return &SessionManager{
        options: options,
        clients: make(map[string]*ClientImpl),
    }
}
```

**File: `types.go`**
```go
// Add session configuration
type SessionConfig struct {
    ID              string
    IsolationMode   SessionIsolationMode
    AutoCleanup     bool
    IdleTimeout     time.Duration
}

type SessionIsolationMode int

const (
    SessionIsolationShared SessionIsolationMode = iota  // Current behavior
    SessionIsolationProcess                               // One subprocess per session
)
```

---

### Enhancement 4: Interruption Support

#### Proposed API

```go
// InterruptQuery interrupts the currently processing query for a session
func (c *ClientImpl) InterruptQuery(sessionID string) error

// SendInterrupt sends interrupt signal to Claude CLI subprocess
func (t *Transport) SendInterrupt() error
```

#### Implementation

**File: `internal/subprocess/transport.go`**

```go
// Add interrupt support to transport
type Transport struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout io.ReadCloser
    stderr io.ReadCloser

    // NEW: Signal channel for interrupts
    interrupt chan struct{}
}

func (t *Transport) SendInterrupt() error {
    // Send Ctrl+C to subprocess
    if t.cmd != nil && t.cmd.Process != nil {
        return t.cmd.Process.Signal(os.Interrupt)
    }
    return fmt.Errorf("no active process to interrupt")
}

// Or send special message to stdin
func (t *Transport) SendInterruptMessage() error {
    interruptMsg := map[string]interface{}{
        "type": "interrupt",
    }

    data, err := json.Marshal(interruptMsg)
    if err != nil {
        return err
    }

    _, err = t.stdin.Write(append(data, '\n'))
    return err
}
```

**File: `client.go`**

```go
func (c *ClientImpl) InterruptQuery(sessionID string) error {
    // Send interrupt to transport
    return c.transport.SendInterrupt()
}
```

#### Usage Example

```go
// Start query
handle, _ := client.QueryAsync(ctx, "Long running task...")

// User presses ESC or cancel button
go func() {
    time.Sleep(2 * time.Second)
    handle.Cancel()  // Cancels via context
}()

// Or use client-level interrupt
client.InterruptQuery("session-id")
```

---

### Enhancement 5: Queue State Inspection

#### Proposed API

```go
// GetActiveQueries returns all active queries across all sessions
func (c *ClientImpl) GetActiveQueries() []QueryInfo

// GetSessionQueries returns queries for a specific session
func (c *ClientImpl) GetSessionQueries(sessionID string) []QueryInfo

// QueryInfo provides information about a query
type QueryInfo struct {
    ID        string
    SessionID string
    Status    QueryStatus
    Submitted time.Time
    Started   *time.Time
    Completed *time.Time
    Duration  time.Duration
}
```

#### Implementation

```go
// client.go - Add query tracking
type ClientImpl struct {
    transport       Transport
    msgChan         <-chan Message
    errChan         <-chan error

    // NEW: Query tracking
    activeQueries   map[string]*QueryInfo
    queriesMutex    sync.RWMutex
}

func (c *ClientImpl) trackQuery(handle *QueryHandle) {
    c.queriesMutex.Lock()
    defer c.queriesMutex.Unlock()

    c.activeQueries[handle.ID] = &QueryInfo{
        ID:        handle.ID,
        SessionID: handle.SessionID,
        Status:    handle.Status,
        Submitted: time.Now(),
    }
}

func (c *ClientImpl) GetActiveQueries() []QueryInfo {
    c.queriesMutex.RLock()
    defer c.queriesMutex.RUnlock()

    queries := make([]QueryInfo, 0, len(c.activeQueries))
    for _, info := range c.activeQueries {
        queries = append(queries, *info)
    }

    return queries
}

func (c *ClientImpl) GetSessionQueries(sessionID string) []QueryInfo {
    c.queriesMutex.RLock()
    defer c.queriesMutex.RUnlock()

    queries := make([]QueryInfo, 0)
    for _, info := range c.activeQueries {
        if info.SessionID == sessionID {
            queries = append(queries, *info)
        }
    }

    return queries
}
```

---

## Complete SDK Enhancement Summary

### New Files to Add

```
claude-code-sdk-go/
├── queue.go           # QueueManager implementation
├── session.go         # SessionManager for isolation
├── handle.go          # QueryHandle and async operations
└── types.go           # Enhanced with new types
```

### Modified Files

```
claude-code-sdk-go/
├── client.go                            # Add async methods
├── internal/subprocess/transport.go     # Add interrupt support
└── README.md                            # Document new APIs
```

### New Types to Add

```go
// Async Operations
type QueryHandle
type QueryStatus
type QueryInfo

// Queue Management
type QueueManager
type MessageQueue
type QueuedMessage
type QueueHandle
type MessageStatus

// Session Management
type SessionManager
type SessionConfig
type SessionIsolationMode

// Interruption
type InterruptSignal
```

### New Methods to Add

```go
// ClientImpl
func (c *ClientImpl) QueryAsync(ctx, prompt) (*QueryHandle, error)
func (c *ClientImpl) QueryWithSessionAsync(ctx, prompt, sessionID) (*QueryHandle, error)
func (c *ClientImpl) InterruptQuery(sessionID) error
func (c *ClientImpl) GetActiveQueries() []QueryInfo
func (c *ClientImpl) GetSessionQueries(sessionID) []QueryInfo

// QueueManager
func NewQueueManager(client) *QueueManager
func (qm *QueueManager) Enqueue(ctx, sessionID, message) (*QueueHandle, error)
func (qm *QueueManager) GetQueueLength(sessionID) int
func (qm *QueueManager) GetQueueStatus(sessionID) (*QueueStatus, error)
func (qm *QueueManager) ClearQueue(sessionID) error
func (qm *QueueManager) RemoveFromQueue(sessionID, messageID) error

// SessionManager
func NewSessionManager(options) *SessionManager
func (sm *SessionManager) GetClient(sessionID) (*ClientImpl, error)
func (sm *SessionManager) Query(ctx, sessionID, prompt) error
func (sm *SessionManager) CloseSession(sessionID) error

// Transport
func (t *Transport) SendInterrupt() error
func (t *Transport) SendInterruptMessage() error

// QueryHandle
func (qh *QueryHandle) Wait() error
func (qh *QueryHandle) GetStatus() QueryStatus
func (qh *QueryHandle) Cancel() error
```

---

## Backward Compatibility

### Approach: Add, Don't Break

All enhancements should be **additive** - existing APIs continue to work:

```go
// Existing API - still works
client.Query(ctx, "prompt")
client.QueryWithSession(ctx, "prompt", "session-id")

// New API - added alongside
client.QueryAsync(ctx, "prompt")
client.QueryWithSessionAsync(ctx, "prompt", "session-id")
```

### Migration Path

**Phase 1:** Add new APIs alongside existing ones
**Phase 2:** Deprecate old synchronous APIs (with warnings)
**Phase 3:** (Future) Remove old APIs in v2.0.0

---

## Testing Requirements

### Unit Tests

```go
// queue_test.go
func TestQueueEnqueue(t *testing.T)
func TestQueueProcessing(t *testing.T)
func TestQueueConcurrency(t *testing.T)

// session_test.go
func TestSessionIsolation(t *testing.T)
func TestMultipleSessions(t *testing.T)

// async_test.go
func TestAsyncQuery(t *testing.T)
func TestConcurrentQueries(t *testing.T)
func TestQueryCancellation(t *testing.T)
```

### Integration Tests

```go
// integration_test.go
func TestQueueWithRealClaude(t *testing.T)
func TestSessionIsolationWithRealClaude(t *testing.T)
func TestInterruptWithRealClaude(t *testing.T)
```

### Benchmarks

```go
// benchmark_test.go
func BenchmarkQueueThroughput(b *testing.B)
func BenchmarkConcurrentSessions(b *testing.B)
func BenchmarkAsyncQueries(b *testing.B)
```

---

## Documentation Requirements

### API Documentation

Update README.md with:
- Async query examples
- Queue management examples
- Session isolation examples
- Interruption examples

### Migration Guide

Create MIGRATION.md:
- How to migrate from sync to async
- How to implement queueing in applications
- How to handle session isolation

### Architecture Guide

Create ARCHITECTURE.md:
- Explain queue design
- Explain session management
- Explain transport layer

---

## Priority Ranking

### P0 - Critical (Required for basic queueing)
1. **Non-blocking message API** (`QueryAsync`)
2. **Session isolation fix** (one subprocess per session)

### P1 - High (Required for production queueing)
3. **Built-in Queue Manager**
4. **Queue state inspection**

### P2 - Medium (Nice to have)
5. **Interruption support**
6. **Advanced queue operations** (priority, reordering)

### P3 - Low (Future enhancements)
7. **Queue persistence**
8. **Queue metrics and monitoring**
9. **Smart queue optimization**

---

## Alternative: Application-Level Implementation

If SDK enhancements are not feasible, applications can implement queueing using current SDK:

### Workaround Pattern

```go
// Application-level queue manager
type AppQueueManager struct {
    client    claudecode.Client
    queues    map[string]chan string
    mu        sync.RWMutex
}

func (aqm *AppQueueManager) Enqueue(sessionID, message string) error {
    aqm.mu.RLock()
    queue, exists := aqm.queues[sessionID]
    aqm.mu.RUnlock()

    if !exists {
        aqm.mu.Lock()
        queue = make(chan string, 100)
        aqm.queues[sessionID] = queue
        aqm.mu.Unlock()

        go aqm.processQueue(sessionID, queue)
    }

    queue <- message
    return nil
}

func (aqm *AppQueueManager) processQueue(sessionID string, queue chan string) {
    for message := range queue {
        // Create new client per message for isolation
        client := claudecode.NewClient(...)
        err := client.Query(context.Background(), message)
        client.Disconnect()

        if err != nil {
            log.Printf("Error: %v", err)
        }
    }
}
```

**Pros:**
- Can implement immediately
- No SDK changes needed
- Full control over behavior

**Cons:**
- Each application reimplements same logic
- No standard API
- Less efficient (creates many subprocesses)

---

## Recommendation

### Short Term: Application-Level Implementation
Use the workaround pattern above in CodeCrucible while SDK enhancements are developed.

### Long Term: SDK Enhancement
Submit PR to `claude-code-sdk-go` with:
1. `QueryAsync` methods (P0)
2. Session isolation fix (P0)
3. Built-in `QueueManager` (P1)

This benefits the entire Claude Code ecosystem, not just CodeCrucible.

---

## Next Steps

1. **Review this proposal** with team
2. **Decide on approach:**
   - Application-level workaround only
   - Contribute to SDK
   - Both (workaround now, SDK later)
3. **Create GitHub issue** in claude-code-sdk-go repository
4. **Begin implementation** based on decision

---

## References

### SDK Files
- `/tmp/claude-code-sdk-go/client.go` - Main client implementation
- `/tmp/claude-code-sdk-go/query.go` - Query methods
- `/tmp/claude-code-sdk-go/types.go` - Type definitions
- `/tmp/claude-code-sdk-go/internal/subprocess/transport.go` - Transport layer

### Related Issues
- Session isolation bug (needs GitHub issue)
- Async query support (needs GitHub issue)

### Similar Implementations
- Go `database/sql` - Connection pooling pattern
- Go `net/http` - Request/response async pattern
- Python asyncio - Queue management pattern

---

**Document Version**: 1.0
**Last Updated**: 2025-11-10
**Author**: CodeCrucible Development Team
