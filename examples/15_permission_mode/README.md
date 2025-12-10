# Permission Mode Example

This example demonstrates dynamic permission mode changes for runtime permission control using `SetPermissionMode()`.

## What It Shows

The example implements a complete workflow showing how to:
1. **Review Code**: Use default mode for manual permission approval during code review
2. **Apply Fixes**: Switch to `acceptEdits` mode to automatically apply suggested improvements
3. **Plan Changes**: Use `plan` mode to analyze further optimizations without executing them
4. **Safe Defaults**: Return to default mode after completing automated operations

## Permission Modes

- **`default`**: Standard interactive permission prompts (default behavior)
- **`acceptEdits`**: Automatically accept file edit operations without prompts
- **`plan`**: Planning mode - analyze and explain actions without executing them
- **`bypassPermissions`**: Bypass all permission checks (use with caution, not shown in example)

## Key Concepts

### Timing
Permission mode changes take effect **between turns**:
```go
// 1. Send query and wait for response
client.Query(ctx, "Review my code")
for msg := range client.ReceiveMessages(ctx) {
    // Process messages...
}

// 2. Change mode between turns
client.SetPermissionMode(ctx, "acceptEdits")

// 3. Next query uses new mode
client.Query(ctx, "Apply the fixes")
```

### State Preservation
Mode changes preserve conversation context:
- Session ID remains the same
- Conversation history is maintained
- Only permission policy changes

### Error Handling
Always handle mode change errors:
```go
if err := client.SetPermissionMode(ctx, "acceptEdits"); err != nil {
    // Previous mode stays active if change fails
    return fmt.Errorf("mode change failed: %w", err)
}
```

## Running the Example

```bash
cd examples/15_permission_mode
go run main.go
```

## Expected Behavior

The example will:
1. Connect to Claude Code CLI
2. Ask Claude to review sample code (with manual permissions)
3. Switch to `acceptEdits` mode
4. Ask Claude to apply suggested fixes automatically
5. Switch to `plan` mode
6. Ask Claude to plan performance optimizations
7. Return to `default` mode
8. Disconnect and exit

## Use Cases

This pattern is useful for:
- **Code Review Workflows**: Review first, then apply approved changes
- **Automated Refactoring**: Switch to `acceptEdits` after manual approval
- **Safe Exploration**: Use `plan` mode to understand what would happen
- **CI/CD Integration**: Dynamic permissions based on environment

## Requirements

- Connected Client (streaming mode)
- Context with appropriate timeout
- Valid permission mode strings

## Related Examples

- `02_client_streaming` - Basic client streaming
- `03_client_multi_turn` - Multi-turn conversations
- `10_context_manager` - Resource management patterns

## See Also

- **Control Protocol Documentation**: `CONTROL_PROTOCOL.md`
- **Client API**: `client.go` - `SetPermissionMode()` method
- **Permission Options**: `options.go` - Initial permission mode configuration
