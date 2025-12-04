# Async Queue Implementation Plan

**Date**: 2025-11-11
**Repository**: `/tmp/claude-code-sdk-go`
**Approach**: Test-Driven Development (TDD) with Subagent Assistance

---

## Overview

Implement non-blocking message submission API with queue management primitives for the Claude Code SDK Go. This enables applications to submit multiple messages without blocking and manage message queues before they are sent to the CLI.

## Architecture: Two-Layer Design

### Layer 1: QueryAsync API (P0 - Foundation)
**Purpose**: Basic async query execution without blocking
**Key Feature**: Messages sent immediately, cannot be removed
**Use Case**: Simple non-blocking queries

```go
handle, err := client.QueryAsync(ctx, "query")
// Message sent immediately to CLI
// Can cancel via handle.Cancel() but message already in CLI
```

### Layer 2: QueueManager (P1 - Queue Manipulation)
**Purpose**: SDK-side queue with deferred sending
**Key Feature**: Messages stay in queue until sent, can be removed
**Use Case**: Full queue management (add, remove, reorder, priority)

```go
handle, err := queueManager.Enqueue(ctx, sessionID, "query")
// Message stays in SDK queue
queueManager.RemoveFromQueue(sessionID, handle.MessageID) // ✅ Can remove before sending
```

---

## Implementation Phases

### Phase 1: QueryAsync API (P0)
1. Create types and interfaces (`query_handle.go`)
2. Write failing tests (`client_async_test.go`)
3. Implement async query methods (`client_async.go`)
4. Extend ClientImpl with query tracking (`client.go`)
5. Verify tests pass

### Phase 2: QueueManager (P1)
1. Create queue types (`queue.go`)
2. Write failing tests (`queue_test.go`)
3. Implement queue manager with deferred sending
4. Implement queue manipulation methods (Remove, Clear, Reorder)
5. Verify tests pass

---

## Phase 1: QueryAsync API Implementation

### Files to Create

#### 1. `query_handle.go` - Core Types and Interfaces

```go
package claudecode

import (
    "context"
    "sync"
)

// QueryHandle represents a non-blocking query execution
type QueryHandle interface {
    // ID returns the unique identifier for this query
    ID() string

    // SessionID returns the session ID for this query
    SessionID() string

    // Status returns the current status of the query
    Status() QueryStatus

    // Messages returns a channel for receiving response messages
    Messages() <-chan Message

    // Errors returns a channel for receiving errors
    Errors() <-chan error

    // Wait blocks until the query completes or fails
    Wait() error

    // Cancel cancels the query execution
    Cancel()

    // Done returns a channel that closes when the query completes
    Done() <-chan struct{}
}

// QueryStatus represents the state of an async query
type QueryStatus int

const (
    QueryStatusQueued QueryStatus = iota
    QueryStatusProcessing
    QueryStatusCompleted
    QueryStatusFailed
    QueryStatusCancelled
)

func (qs QueryStatus) String() string {
    switch qs {
    case QueryStatusQueued:
        return "queued"
    case QueryStatusProcessing:
        return "processing"
    case QueryStatusCompleted:
        return "completed"
    case QueryStatusFailed:
        return "failed"
    case QueryStatusCancelled:
        return "cancelled"
    default:
        return "unknown"
    }
}

// queryHandle implements QueryHandle interface
type queryHandle struct {
    id        string
    sessionID string

    // Status tracking
    statusMu sync.RWMutex
    status   QueryStatus

    // Communication channels (buffered for non-blocking sends)
    messages chan Message
    errors   chan error
    done     chan struct{}

    // Cancellation
    ctx       context.Context
    cancel    context.CancelFunc
    cancelMu  sync.Mutex
    cancelled bool

    // Completion tracking
    waitOnce sync.Once
    waitErr  error
    waitDone chan struct{}
}

// Constructor
func newQueryHandle(parentCtx context.Context, sessionID string) *queryHandle

// Interface implementations
func (qh *queryHandle) ID() string
func (qh *queryHandle) SessionID() string
func (qh *queryHandle) Status() QueryStatus
func (qh *queryHandle) Messages() <-chan Message
func (qh *queryHandle) Errors() <-chan error
func (qh *queryHandle) Done() <-chan struct{}
func (qh *queryHandle) Wait() error
func (qh *queryHandle) Cancel()

// Internal methods
func (qh *queryHandle) setStatus(status QueryStatus)
func (qh *queryHandle) complete(err error)

// Helper functions
func generateQueryID() string
func isDoneMessage(msg Message) bool
```

**Buffer Sizes**:
- `messages`: 100 (typical response is 5-20 messages)
- `errors`: 1 (only first error matters)
- `done`: unbuffered (signaling channel)

#### 2. `client_async.go` - Async Query Methods

```go
package claudecode

import (
    "context"
    "fmt"
)

// QueryAsync submits a query without blocking, using default session.
// Returns a handle that can be used to monitor/control the query.
// The message is sent immediately to the CLI subprocess.
func (c *ClientImpl) QueryAsync(ctx context.Context, prompt string) (QueryHandle, error)

// QueryWithSessionAsync submits a query without blocking to a specific session.
// Each session maintains its own conversation context.
// The message is sent immediately to the CLI subprocess.
func (c *ClientImpl) QueryWithSessionAsync(ctx context.Context, prompt string, sessionID string) (QueryHandle, error)

// executeAsyncQuery runs in a goroutine to execute the query asynchronously
func (c *ClientImpl) executeAsyncQuery(handle *queryHandle, prompt string, sessionID string, transport Transport)

// Query tracking methods
func (c *ClientImpl) trackQuery(handle *queryHandle)
func (c *ClientImpl) untrackQuery(queryID string)
func (c *ClientImpl) getActiveQueryCount() int
```

**Implementation Pattern**:
1. Validate connection state
2. Create queryHandle
3. Track query
4. Start goroutine for execution
5. Return handle immediately

**Goroutine Pattern**:
1. Update status to processing
2. Create StreamMessage
3. Send via transport
4. Stream responses to handle channels
5. Complete on done/error/cancellation
6. Untrack query in defer

#### 3. `client_async_test.go` - Comprehensive Tests

```go
package claudecode

import (
    "context"
    "fmt"
    "testing"
    "time"
)

// Test functions first (primary purpose)
func TestClientQueryAsync(t *testing.T)
func TestClientQueryWithSessionAsync(t *testing.T)
func TestQueryAsyncNotConnected(t *testing.T)
func TestQueryHandleWait(t *testing.T)
func TestQueryHandleCancellation(t *testing.T)
func TestQueryHandleErrorPropagation(t *testing.T)
func TestConcurrentAsyncQueries(t *testing.T)
func TestQueryHandleMessageStreaming(t *testing.T)
func TestQueryHandleStatusTransitions(t *testing.T)
func TestQueryTrackingCleanup(t *testing.T)
func TestQueryHandleInterfaceCompliance(t *testing.T)

// Mock implementations (supporting types)
type mockTransportForAsync struct {
    // ... fields for async testing
}

// Helper functions (utilities)
func setupAsyncTestClient(t *testing.T) (*ClientImpl, *mockTransportForAsync)
func createTestMessage(content string) Message
func verifyQueryCompleted(t *testing.T, handle QueryHandle)
```

**Test Categories**:
1. **Basic async query** - Submit and receive
2. **Session-specific async** - Verify session ID propagation
3. **Connection validation** - Error when not connected
4. **Wait functionality** - Blocking until completion
5. **Cancellation** - Cancel via handle.Cancel()
6. **Error propagation** - Errors flow through error channel
7. **Concurrent queries** - Multiple simultaneous queries
8. **Message streaming** - Messages arrive in order
9. **Status transitions** - Queued → Processing → Completed/Failed/Cancelled
10. **Tracking cleanup** - Active queries map cleaned up
11. **Interface compliance** - Verify QueryHandle interface

### Files to Modify

#### 4. `client.go` - Add Query Tracking

```go
type ClientImpl struct {
    mu              sync.RWMutex
    transport       Transport
    customTransport Transport
    options         *Options
    connected       bool
    msgChan         <-chan Message
    errChan         <-chan error

    // NEW: Active query tracking
    activeQueries   map[string]*queryHandle
    queriesMu       sync.RWMutex
}

// Initialize in NewClient
func NewClient(options ...Option) Client {
    // ... existing code ...
    return &ClientImpl{
        // ... existing fields ...
        activeQueries: make(map[string]*queryHandle),
    }
}

// Clean up in Disconnect
func (c *ClientImpl) Disconnect() error {
    // ... existing code ...

    // Cancel all active queries
    c.queriesMu.Lock()
    for _, handle := range c.activeQueries {
        handle.Cancel()
    }
    c.activeQueries = make(map[string]*queryHandle)
    c.queriesMu.Unlock()

    // ... rest of disconnect logic ...
}
```

#### 5. `types.go` - Export QueryHandle Interface

```go
package claudecode

// Export QueryHandle and QueryStatus for public API
type (
    // QueryHandle is exported from query_handle.go
    QueryHandle = QueryHandle

    // QueryStatus is exported from query_handle.go
    QueryStatus = QueryStatus
)

// Export QueryStatus constants
const (
    QueryStatusQueued     = QueryStatusQueued
    QueryStatusProcessing = QueryStatusProcessing
    QueryStatusCompleted  = QueryStatusCompleted
    QueryStatusFailed     = QueryStatusFailed
    QueryStatusCancelled  = QueryStatusCancelled
)
```

---

## Phase 2: QueueManager Implementation

### Files to Create

#### 6. `queue.go` - Queue Manager with Deferred Sending

```go
package claudecode

import (
    "context"
    "fmt"
    "sync"
    "time"
)

// QueueManager manages message queues for multiple sessions
// Messages stay in SDK-side queue until processed
// Supports removal, reordering, and priority before sending
type QueueManager struct {
    client *Client
    queues map[string]*MessageQueue
    mu     sync.RWMutex
    ctx    context.Context
    cancel context.CancelFunc
}

// MessageQueue represents a queue for a specific session
type MessageQueue struct {
    sessionID   string
    messages    []QueuedMessage
    processing  *QueuedMessage
    mu          sync.RWMutex
    pauseChan   chan struct{}
    paused      bool
}

// QueuedMessage represents a message in the queue
type QueuedMessage struct {
    ID        string
    Content   string
    Priority  int
    Timestamp time.Time
    Handle    QueryHandle
    Status    MessageStatus
}

// MessageStatus represents the state of a queued message
type MessageStatus int

const (
    MessageStatusQueued MessageStatus = iota
    MessageStatusProcessing
    MessageStatusCompleted
    MessageStatusFailed
    MessageStatusCancelled
    MessageStatusRemoved
)

// Constructor
func NewQueueManager(client Client) *QueueManager

// Core queue operations
func (qm *QueueManager) Enqueue(ctx context.Context, sessionID, message string) (*QueuedMessage, error)
func (qm *QueueManager) RemoveFromQueue(sessionID, messageID string) error
func (qm *QueueManager) ClearQueue(sessionID string) error
func (qm *QueueManager) GetQueueStatus(sessionID string) (*QueueStatus, error)
func (qm *QueueManager) GetQueueLength(sessionID string) int

// Advanced operations
func (qm *QueueManager) PauseQueue(sessionID string) error
func (qm *QueueManager) ResumeQueue(sessionID string) error
func (qm *QueueManager) ReorderQueue(sessionID string, messageIDs []string) error
func (qm *QueueManager) SetPriority(sessionID, messageID string, priority int) error

// Internal processing
func (qm *QueueManager) processQueue(ctx context.Context, queue *MessageQueue)
func (qm *QueueManager) getQueue(sessionID string, create bool) (*MessageQueue, error)

// Shutdown
func (qm *QueueManager) Close() error
```

**Key Implementation Details**:

1. **Enqueue**: Adds message to queue, starts processor if needed
2. **RemoveFromQueue**: Removes from `queue.messages` slice if not processing
3. **processQueue**: Goroutine that dequeues and calls `client.QueryWithSessionAsync()`
4. **Pause/Resume**: Temporarily stop processing without clearing queue
5. **Reorder**: Change message order in queue
6. **Priority**: Sort queue by priority field

#### 7. `queue_test.go` - Queue Manager Tests

```go
package claudecode

import (
    "context"
    "testing"
    "time"
)

// Test functions
func TestQueueManagerEnqueue(t *testing.T)
func TestQueueManagerRemoveFromQueue(t *testing.T)
func TestQueueManagerClearQueue(t *testing.T)
func TestQueueManagerGetQueueStatus(t *testing.T)
func TestQueueManagerConcurrentEnqueue(t *testing.T)
func TestQueueManagerRemoveDuringProcessing(t *testing.T)
func TestQueueManagerPauseResume(t *testing.T)
func TestQueueManagerReorder(t *testing.T)
func TestQueueManagerPriority(t *testing.T)
func TestQueueManagerMultipleSessions(t *testing.T)
func TestQueueManagerProcessingOrder(t *testing.T)

// Mock implementations
type mockClientForQueue struct {
    // ... fields
}

// Helper functions
func setupQueueTestClient(t *testing.T) (*QueueManager, *mockClientForQueue)
func verifyQueueLength(t *testing.T, qm *QueueManager, sessionID string, expected int)
func verifyMessageRemoved(t *testing.T, qm *QueueManager, sessionID, messageID string)
```

---

## TDD Workflow with Subagents

### Step 1: Write Tests (RED Phase)

**Subagent Task**: "Write comprehensive tests for async query API"
- Input: This plan document + `client_test.go` (reference pattern)
- Output: Complete `client_async_test.go` with failing tests
- Focus: Follow `client_test.go` patterns exactly

### Step 2: Implement Code (GREEN Phase)

**Subagent Task**: "Implement async query API to pass tests"
- Input: Failing tests + this plan document
- Output: Implementation of `query_handle.go` + `client_async.go`
- Focus: Make tests pass with minimal code

### Step 3: Refactor (BLUE Phase)

**Subagent Task**: "Refactor async implementation for optimization"
- Input: Passing tests + implementation
- Output: Optimized code (better error handling, documentation, edge cases)
- Focus: Maintain test pass, improve code quality

### Step 4: Queue Manager Tests (RED Phase)

**Subagent Task**: "Write tests for queue manager"
- Input: This plan + Phase 2 requirements
- Output: Complete `queue_test.go` with failing tests

### Step 5: Queue Manager Implementation (GREEN Phase)

**Subagent Task**: "Implement queue manager to pass tests"
- Input: Failing queue tests + this plan
- Output: `queue.go` implementation
- Focus: Make all queue tests pass

### Step 6: Integration Tests (Validation)

**Subagent Task**: "Write integration tests for full async + queue workflow"
- Input: All implementations
- Output: End-to-end tests
- Focus: Real-world usage scenarios

---

## Thread Safety Requirements

### Lock Hierarchy (Must acquire in this order)
1. `ClientImpl.mu` (client state)
2. `ClientImpl.queriesMu` (query tracking)
3. `QueueManager.mu` (queue manager state)
4. `MessageQueue.mu` (individual queue state)
5. `queryHandle.statusMu` (query status)
6. `queryHandle.cancelMu` (cancellation)

### Race Condition Prevention
- Always use RWMutex for read-heavy operations
- Minimize lock duration
- Copy values before releasing lock
- Use buffered channels to prevent goroutine blocking
- Use sync.Once for one-time operations

---

## Testing Strategy

### Unit Tests
- Test each component in isolation
- Use mocks for dependencies
- Test thread safety with race detector: `go test -race`
- Test error conditions exhaustively

### Integration Tests
- Test real subprocess communication
- Test with actual Claude CLI
- Test concurrent scenarios
- Test resource cleanup

### Performance Tests
- Benchmark query throughput: `go test -bench=.`
- Test memory usage: `go test -benchmem`
- Test with 100+ concurrent queries
- Verify no goroutine leaks

### Test Commands
```bash
# Run all async tests
go test -v -run TestClient.*Async

# Run with race detector
go test -race -v -run TestClient.*Async

# Run queue tests
go test -v -run TestQueue

# Benchmark async queries
go test -bench=BenchmarkAsync -benchmem

# Test consistency (run 10 times)
go test -count=10 -run TestConcurrentAsyncQueries
```

---

## Success Criteria

### Phase 1 (P0) Complete When:
- [ ] All tests in `client_async_test.go` pass
- [ ] `go test -race` passes with no warnings
- [ ] Can submit multiple async queries without blocking
- [ ] Cancellation works correctly
- [ ] Error propagation works correctly
- [ ] Query tracking cleanup works correctly
- [ ] Documentation complete with examples
- [ ] Backward compatibility maintained (existing tests still pass)

### Phase 2 (P1) Complete When:
- [ ] All tests in `queue_test.go` pass
- [ ] Can enqueue messages without sending
- [ ] Can remove messages from queue before sending
- [ ] Can pause/resume queue processing
- [ ] Can reorder queue
- [ ] Multiple sessions isolated correctly
- [ ] All queue operations thread-safe
- [ ] Documentation complete with examples

---

## Example Usage Patterns

### Pattern 1: Basic Async Query
```go
client := claudecode.NewClient()
defer client.Disconnect()

ctx := context.Background()
client.Connect(ctx)

handle, err := client.QueryAsync(ctx, "What is 2+2?")
if err != nil {
    log.Fatal(err)
}

for msg := range handle.Messages() {
    // Process message
}

if err := handle.Wait(); err != nil {
    log.Fatal(err)
}
```

### Pattern 2: Concurrent Queries
```go
handles := []QueryHandle{}
for i := 0; i < 10; i++ {
    handle, _ := client.QueryAsync(ctx, fmt.Sprintf("Query %d", i))
    handles = append(handles, handle)
}

for _, handle := range handles {
    handle.Wait()
}
```

### Pattern 3: Queue with Removal
```go
qm := claudecode.NewQueueManager(client)
defer qm.Close()

// Enqueue messages
h1, _ := qm.Enqueue(ctx, "session-1", "Message 1")
h2, _ := qm.Enqueue(ctx, "session-1", "Message 2")
h3, _ := qm.Enqueue(ctx, "session-1", "Message 3")

// User decides to remove message 2 (like pressing up arrow in CLI)
qm.RemoveFromQueue("session-1", h2.ID)

// Only messages 1 and 3 will be sent
```

### Pattern 4: Query with Timeout
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

handle, err := client.QueryAsync(ctx, "Long task...")
if err := handle.Wait(); err != nil {
    if err == context.DeadlineExceeded {
        fmt.Println("Query timed out")
    }
}
```

---

## Performance Characteristics

### Memory Usage
- Per QueryHandle: ~2KB (channels + metadata)
- Per QueuedMessage: ~500 bytes
- Message buffer: up to 100 messages × message size
- Expected for 100 concurrent queries: ~200KB overhead

### Goroutines
- One goroutine per async query (2KB stack)
- One goroutine per queue processor
- Expected for 10 sessions + 100 queries: ~200KB goroutine overhead

### Lock Contention
- RWMutex minimizes read contention
- Lock duration < 1μs for most operations
- No lock held during I/O operations

---

## Documentation Updates

### README.md
- Add async query examples
- Add queue manager examples
- Update API reference

### EXAMPLES.md
- Create comprehensive async examples
- Show queue manipulation patterns
- Show cancellation patterns

### MIGRATION.md (for users)
- How to migrate from sync to async
- Benefits of queue manager
- Breaking changes (none expected)

---

## Implementation Checklist

### Phase 1: QueryAsync API
- [ ] Create `query_handle.go` with types and interfaces
- [ ] Write tests in `client_async_test.go` (RED)
- [ ] Implement `query_handle.go` implementation (GREEN)
- [ ] Implement `client_async.go` methods (GREEN)
- [ ] Modify `client.go` for query tracking (GREEN)
- [ ] Verify all tests pass (GREEN)
- [ ] Run race detector
- [ ] Refactor for optimization (BLUE)
- [ ] Add documentation and examples
- [ ] Update README.md

### Phase 2: QueueManager
- [ ] Create `queue.go` types
- [ ] Write tests in `queue_test.go` (RED)
- [ ] Implement `queue.go` core operations (GREEN)
- [ ] Implement advanced queue operations (GREEN)
- [ ] Verify all tests pass (GREEN)
- [ ] Run race detector
- [ ] Test with real CLI
- [ ] Refactor for optimization (BLUE)
- [ ] Add documentation and examples
- [ ] Update README.md

---

## Notes for Subagents

1. **Reference Implementation**: Use `client_test.go` as the gold standard for test structure
2. **Test File Organization**: Tests first, mocks second, helpers third
3. **Context Management**: Only add context when actually needed for blocking operations
4. **Helper Functions**: Call `t.Helper()` in all test utilities
5. **Mock Design**: Thread-safe with proper mutex usage
6. **Error Messages**: Include context (file:line references)
7. **No Placeholders**: Never use dummy code or TODOs
8. **Self-Contained**: Each test file has its own helpers

---

## Version History

- **v1.0** (2025-11-11): Initial implementation plan created
