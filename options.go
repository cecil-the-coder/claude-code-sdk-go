package claudecode

import (
	"github.com/severity1/claude-code-sdk-go/internal/shared"
)

// Options contains configuration for Claude Code CLI interactions.
type Options = shared.Options

// PermissionMode defines the permission handling mode.
type PermissionMode = shared.PermissionMode

// McpServerType defines the type of MCP server.
type McpServerType = shared.McpServerType

// McpServerConfig represents an MCP server configuration.
type McpServerConfig = shared.McpServerConfig

// McpStdioServerConfig represents a stdio MCP server configuration.
type McpStdioServerConfig = shared.McpStdioServerConfig

// McpSSEServerConfig represents an SSE MCP server configuration.
type McpSSEServerConfig = shared.McpSSEServerConfig

// McpHTTPServerConfig represents an HTTP MCP server configuration.
type McpHTTPServerConfig = shared.McpHTTPServerConfig

// AgentDefinition defines a custom agent with specific prompts, tools, and model.
type AgentDefinition = shared.AgentDefinition

// Hook System Types
type HookEventName = shared.HookEventName
type HookCallback = shared.HookCallback
type HookMatcher = shared.HookMatcher
type HookInput = shared.HookInput
type HookContext = shared.HookContext

// Re-export constants
const (
	PermissionModeDefault           = shared.PermissionModeDefault
	PermissionModeAcceptEdits       = shared.PermissionModeAcceptEdits
	PermissionModePlan              = shared.PermissionModePlan
	PermissionModeBypassPermissions = shared.PermissionModeBypassPermissions
	McpServerTypeStdio              = shared.McpServerTypeStdio
	McpServerTypeSSE                = shared.McpServerTypeSSE
	McpServerTypeHTTP               = shared.McpServerTypeHTTP
	HookEventPreToolUse             = shared.HookEventPreToolUse
	HookEventPostToolUse            = shared.HookEventPostToolUse
	HookEventUserPromptSubmit       = shared.HookEventUserPromptSubmit
	HookEventStop                   = shared.HookEventStop
	HookEventSubagentStop           = shared.HookEventSubagentStop
	HookEventPreCompact             = shared.HookEventPreCompact
	DefaultHookTimeout              = shared.DefaultHookTimeout
)

// Option configures Options using the functional options pattern.
type Option func(*Options)

// WithAllowedTools sets the allowed tools list.
func WithAllowedTools(tools ...string) Option {
	return func(o *Options) {
		o.AllowedTools = tools
	}
}

// WithDisallowedTools sets the disallowed tools list.
func WithDisallowedTools(tools ...string) Option {
	return func(o *Options) {
		o.DisallowedTools = tools
	}
}

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(o *Options) {
		o.SystemPrompt = &prompt
	}
}

// WithAppendSystemPrompt sets the append system prompt.
func WithAppendSystemPrompt(prompt string) Option {
	return func(o *Options) {
		o.AppendSystemPrompt = &prompt
	}
}

// WithModel sets the model to use.
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = &model
	}
}

// WithMaxThinkingTokens sets the maximum thinking tokens.
func WithMaxThinkingTokens(tokens int) Option {
	return func(o *Options) {
		o.MaxThinkingTokens = tokens
	}
}

// WithPermissionMode sets the permission mode.
func WithPermissionMode(mode PermissionMode) Option {
	return func(o *Options) {
		o.PermissionMode = &mode
	}
}

// WithPermissionPromptToolName sets the permission prompt tool name.
func WithPermissionPromptToolName(toolName string) Option {
	return func(o *Options) {
		o.PermissionPromptToolName = &toolName
	}
}

// WithCanUseTool sets the tool permission callback for dynamic permission decisions.
// The callback is invoked when Claude wants to use a tool, allowing you to:
//   - Allow the tool use (optionally modifying inputs or updating permissions)
//   - Deny the tool use (optionally interrupting the conversation)
//
// When this callback is set, it automatically sets permission_prompt_tool_name="stdio"
// for control protocol communication. This option is mutually exclusive with
// WithPermissionPromptToolName - setting both will cause validation to fail.
//
// The callback receives context for cancellation, tool name, input parameters,
// and additional context including CLI suggestions.
//
// Example - Block dangerous commands:
//
//	WithCanUseTool(func(ctx context.Context, toolName string, input map[string]interface{}, context ToolPermissionContext) (PermissionResult, error) {
//	    if toolName == "Bash" {
//	        command := input["command"].(string)
//	        if strings.Contains(command, "rm -rf") {
//	            return &PermissionResultDeny{
//	                Message: "Dangerous command blocked",
//	            }, nil
//	        }
//	    }
//	    return &PermissionResultAllow{}, nil
//	})
//
// Example - Redirect file writes to safe directory:
//
//	WithCanUseTool(func(ctx context.Context, toolName string, input map[string]interface{}, context ToolPermissionContext) (PermissionResult, error) {
//	    if toolName == "Write" {
//	        filePath := input["file_path"].(string)
//	        if !strings.HasPrefix(filePath, "/tmp/") {
//	            safePath := filepath.Join("/tmp", filepath.Base(filePath))
//	            modifiedInput := make(map[string]interface{})
//	            for k, v := range input {
//	                modifiedInput[k] = v
//	            }
//	            modifiedInput["file_path"] = safePath
//	            return &PermissionResultAllow{
//	                UpdatedInput: modifiedInput,
//	            }, nil
//	        }
//	    }
//	    return &PermissionResultAllow{}, nil
//	})
func WithCanUseTool(callback CanUseToolFunc) Option {
	return func(o *Options) {
		o.CanUseTool = callback
	}
}

// WithContinueConversation enables conversation continuation.
func WithContinueConversation(continueConversation bool) Option {
	return func(o *Options) {
		o.ContinueConversation = continueConversation
	}
}

// WithResume sets the session ID to resume.
func WithResume(sessionID string) Option {
	return func(o *Options) {
		o.Resume = &sessionID
	}
}

// WithSessionID sets the session ID for conversation isolation.
func WithSessionID(sessionID string) Option {
	return func(o *Options) {
		o.SessionID = &sessionID
	}
}

// WithCwd sets the working directory.
func WithCwd(cwd string) Option {
	return func(o *Options) {
		o.Cwd = &cwd
	}
}

// WithAddDirs adds directories to the context.
func WithAddDirs(dirs ...string) Option {
	return func(o *Options) {
		o.AddDirs = dirs
	}
}

// WithMcpServers sets the MCP server configurations.
func WithMcpServers(servers map[string]McpServerConfig) Option {
	return func(o *Options) {
		o.McpServers = servers
	}
}

// WithAgents sets the custom agent definitions.
func WithAgents(agents map[string]AgentDefinition) Option {
	return func(o *Options) {
		o.Agents = agents
	}
}

// WithHooks sets hook callbacks for lifecycle events.
// Hooks allow you to intercept and respond to events like tool execution,
// user prompts, conversation stops, and history compaction.
//
// Example - Log all tool usage:
//
//	WithHooks(map[HookEventName]*HookMatcher{
//	    HookEventPreToolUse: {
//	        Matcher: nil, // matches all tools
//	        Hooks: []HookCallback{
//	            func(ctx context.Context, input HookInput, toolUseID *string, context HookContext) (map[string]any, error) {
//	                log.Printf("Tool: %s, Input: %v", *input.ToolName, input.ToolInput)
//	                return map[string]any{"continue": true}, nil
//	            },
//	        },
//	    },
//	})
//
// Example - Block dangerous bash commands:
//
//	WithHooks(map[HookEventName]*HookMatcher{
//	    HookEventPreToolUse: {
//	        Matcher: strPtr("Bash"),
//	        Hooks: []HookCallback{
//	            func(ctx context.Context, input HookInput, toolUseID *string, context HookContext) (map[string]any, error) {
//	                command := input.ToolInput["command"].(string)
//	                if strings.Contains(command, "rm -rf") {
//	                    return map[string]any{
//	                        "decision": "deny",
//	                        "systemMessage": "Dangerous command blocked",
//	                    }, nil
//	                }
//	                return map[string]any{"continue": true}, nil
//	            },
//	        },
//	    },
//	})
//
// Example - Add context after tool execution:
//
//	WithHooks(map[HookEventName]*HookMatcher{
//	    HookEventPostToolUse: {
//	        Matcher: strPtr("Bash|Edit|Write"),
//	        Hooks: []HookCallback{
//	            func(ctx context.Context, input HookInput, toolUseID *string, context HookContext) (map[string]any, error) {
//	                return map[string]any{
//	                    "continue": true,
//	                    "additionalContext": "Tool executed successfully",
//	                }, nil
//	            },
//	        },
//	    },
//	})
func WithHooks(hooks map[HookEventName]*HookMatcher) Option {
	return func(o *Options) {
		if o.Hooks == nil {
			o.Hooks = make(map[HookEventName]*HookMatcher)
		}
		for k, v := range hooks {
			o.Hooks[k] = v
		}
	}
}

// WithHook adds a single hook for a specific event type.
// This is a convenience method for adding individual hooks.
//
// Example:
//
//	WithHook(HookEventPreToolUse, &HookMatcher{
//	    Matcher: strPtr("Bash"),
//	    Hooks: []HookCallback{myCallback},
//	})
func WithHook(eventType HookEventName, matcher *HookMatcher) Option {
	return func(o *Options) {
		if o.Hooks == nil {
			o.Hooks = make(map[HookEventName]*HookMatcher)
		}
		o.Hooks[eventType] = matcher
	}
}

// WithMaxTurns sets the maximum number of conversation turns.
func WithMaxTurns(turns int) Option {
	return func(o *Options) {
		o.MaxTurns = turns
	}
}

// WithSettings sets the settings file path or JSON string.
func WithSettings(settings string) Option {
	return func(o *Options) {
		o.Settings = &settings
	}
}

// WithExtraArgs sets arbitrary CLI flags via ExtraArgs.
func WithExtraArgs(args map[string]*string) Option {
	return func(o *Options) {
		o.ExtraArgs = args
	}
}

// WithExtraFlag adds a boolean flag to ExtraArgs.
// This is a convenience helper for adding flags without values (nil value).
// Example: WithExtraFlag("fork-session") instead of manually creating map[string]*string{"fork-session": nil}
func WithExtraFlag(name string) Option {
	return func(o *Options) {
		if o.ExtraArgs == nil {
			o.ExtraArgs = make(map[string]*string)
		}
		o.ExtraArgs[name] = nil
	}
}

// WithExtraArg adds a flag with a value to ExtraArgs.
// This is a convenience helper for adding flags with values.
// Example: WithExtraArg("setting", "value") instead of manually creating map[string]*string with pointer.
func WithExtraArg(name, value string) Option {
	return func(o *Options) {
		if o.ExtraArgs == nil {
			o.ExtraArgs = make(map[string]*string)
		}
		valueCopy := value
		o.ExtraArgs[name] = &valueCopy
	}
}

// WithForkSession enables fork-session mode when resuming a conversation.
// Creates a new session ID while preserving the conversation history from the resumed session.
// This allows branching conversations to explore different paths while maintaining the original context.
//
// Must be used with WithResume() to specify which session to fork from.
//
// Use cases:
//   - Exploring alternative approaches without affecting the original conversation
//   - Creating "what-if" scenarios from a specific conversation state
//   - Parallel experimentation with different prompts from the same starting point
//   - Recovery scenarios where you want to retry from a known good state
//
// Example:
//
//	// Fork from an existing session to explore an alternative approach
//	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
//	    return client.Query(ctx, "Let's try a different approach")
//	}, claudecode.WithResume(originalSessionID),
//	   claudecode.WithForkSession())
func WithForkSession() Option {
	return WithExtraFlag("fork-session")
}

// WithCLIPath sets a custom CLI path.
func WithCLIPath(path string) Option {
	return func(o *Options) {
		o.CLIPath = &path
	}
}

// WithEnv sets environment variables for the subprocess.
// Multiple calls to WithEnv or WithEnvVar merge the values.
// Later calls override earlier ones for the same key.
func WithEnv(env map[string]string) Option {
	return func(o *Options) {
		if o.ExtraEnv == nil {
			o.ExtraEnv = make(map[string]string)
		}
		// Merge pattern - idiomatic Go
		for k, v := range env {
			o.ExtraEnv[k] = v
		}
	}
}

// WithEnvVar sets a single environment variable for the subprocess.
// This is a convenience method for setting individual variables.
func WithEnvVar(key, value string) Option {
	return func(o *Options) {
		if o.ExtraEnv == nil {
			o.ExtraEnv = make(map[string]string)
		}
		o.ExtraEnv[key] = value
	}
}

const customTransportMarker = "custom_transport"

// WithTransport sets a custom transport for testing.
// Since Transport is not part of Options struct, this is handled in client creation.
func WithTransport(_ Transport) Option {
	return func(o *Options) {
		// This will be handled in client implementation
		// For now, we'll use a special marker in ExtraArgs
		if o.ExtraArgs == nil {
			o.ExtraArgs = make(map[string]*string)
		}
		marker := customTransportMarker
		o.ExtraArgs["__transport_marker__"] = &marker
	}
}

// NewOptions creates Options with default values using functional options pattern.
func NewOptions(opts ...Option) *Options {
	// Create options with defaults from shared package
	options := shared.NewOptions()

	// Apply functional options
	for _, opt := range opts {
		opt(options)
	}

	return options
}
