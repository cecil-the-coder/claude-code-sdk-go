# Hooks System Demo

This example demonstrates the comprehensive Hook System for lifecycle events in the Claude Code SDK for Go.

## Overview

The Hook System allows you to intercept and respond to various events during the SDK's operation:

- **PreToolUse**: Before tool execution (validate, log, modify inputs, control permissions)
- **PostToolUse**: After tool execution (add context, log results, collect metrics)
- **UserPromptSubmit**: When user submits a prompt (track interactions, analytics)
- **Stop**: When execution stops (cleanup, summaries, save state)
- **SubagentStop**: When a subagent completes (agent orchestration)
- **PreCompact**: Before compacting conversation history (control compaction behavior)

## Features Demonstrated

1. **PreToolUse Hook** - Logging and Validation
   - Log all tool usage
   - Block dangerous commands
   - Validate tool inputs

2. **PostToolUse Hook** - Inject Additional Context
   - Add context after tool execution
   - Log tool results
   - Pattern matching for specific tools

3. **UserPromptSubmit Hook** - Track User Interactions
   - Monitor user prompts
   - Analytics and logging
   - Session tracking

4. **Stop Hook** - Cleanup and Summary
   - Perform cleanup operations
   - Generate summaries
   - Save conversation state

5. **Multiple Hooks** - Combined Usage
   - Use multiple hooks together
   - Shared state between hooks
   - Tool usage counting

6. **Pattern Matching** - Tool-Specific Hooks
   - Match specific tools with regex
   - Pipe patterns (e.g., "Bash|Edit|Write")
   - Exact matching

## Hook Callback Signature

```go
type HookCallback func(
    ctx context.Context,
    input HookInput,
    toolUseID *string,
    context HookContext,
) (map[string]any, error)
```

## Hook Input Types

Each hook type receives different input fields:

### PreToolUse
- `ToolName`: Name of the tool being executed
- `ToolInput`: Tool input parameters
- `SessionID`, `TranscriptPath`, `Cwd`, `PermissionMode`

### PostToolUse
- Same as PreToolUse, plus:
- `ToolResponse`: The tool's response

### UserPromptSubmit
- `Prompt`: User's prompt text
- `SessionID`, `TranscriptPath`, `Cwd`

### Stop / SubagentStop
- `StopHookActive`: Whether stop hook is active
- `SessionID`, `TranscriptPath`, `Cwd`

### PreCompact
- `Trigger`: What triggered compaction (e.g., "max_turns")
- `CustomInstructions`: Compaction instructions
- `SessionID`, `TranscriptPath`, `Cwd`

## Hook Output Format

Hooks return a `map[string]any` with control fields:

### Control Fields
- `continue` (bool): Continue execution
- `suppressOutput` (bool): Suppress tool output
- `stopReason` (string): Reason for stopping

### Decision Fields (PreToolUse)
- `decision` (string): "allow" or "deny"
- `systemMessage` (string): Message to show user
- `reason` (string): Reason for decision

### Async Fields
- `async` (bool): Execute asynchronously
- `asyncTimeout` (float64): Timeout in seconds

### Hook-Specific Output
- `additionalContext` (string): Extra context to inject

## Pattern Matching

Hooks support flexible pattern matching for tool names:

```go
Matcher: nil                    // Match all tools
Matcher: strPtr("Bash")         // Exact match
Matcher: strPtr("Bash|Edit")    // Pipe pattern (OR)
Matcher: strPtr("^(Bash|Edit|Write)$")  // Regex pattern
```

## Usage Examples

### Basic Hook

```go
claudecode.WithHooks(map[claudecode.HookEventName]*claudecode.HookMatcher{
    claudecode.HookEventPreToolUse: {
        Matcher: nil, // match all tools
        Hooks: []claudecode.HookCallback{
            func(ctx context.Context, input claudecode.HookInput, toolUseID *string, hookCtx claudecode.HookContext) (map[string]any, error) {
                fmt.Printf("Tool: %s\n", *input.ToolName)
                return map[string]any{"continue": true}, nil
            },
        },
    },
})
```

### Multiple Hooks

```go
claudecode.WithHooks(map[claudecode.HookEventName]*claudecode.HookMatcher{
    claudecode.HookEventPreToolUse: {...},
    claudecode.HookEventPostToolUse: {...},
    claudecode.HookEventStop: {...},
})
```

### Custom Timeout

```go
&claudecode.HookMatcher{
    Matcher: strPtr("Bash"),
    Timeout: floatPtr(120.0), // 2 minutes
    Hooks: []claudecode.HookCallback{...},
}
```

## Running the Example

```bash
go run main.go
```

## Integration with Other Features

The Hook System integrates seamlessly with:
- **Tool Permission Callbacks** (`WithCanUseTool`)
- **Permission Modes** (`WithPermissionMode`)
- **Session Management** (`WithSessionID`)
- **Agent Definitions** (`WithAgents`)

## Best Practices

1. **Keep hooks fast** - Avoid long-running operations in hook callbacks
2. **Use pattern matching** - Target specific tools instead of matching all
3. **Handle errors gracefully** - Return proper error messages
4. **Use async for expensive operations** - Set async: true for long tasks
5. **Clean up state** - Use Stop hooks for cleanup operations
6. **Log strategically** - Use hooks for audit trails and debugging

## Python SDK Parity

This implementation provides 100% feature parity with the Python SDK's hook system:
- All 6 hook event types
- Pattern matching with regex
- Async hook execution
- Custom timeouts
- Complete hook input/output structures
