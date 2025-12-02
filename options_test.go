package claudecode

import (
	"context"
	"testing"
)

// T015: Default Options Creation - Test functional options integration
func TestDefaultOptions(t *testing.T) {
	// Test that NewOptions() creates proper defaults via shared package
	options := NewOptions()

	// Verify that functional options work with shared types
	assertOptionsMaxThinkingTokens(t, options, 8000)

	// Test that we can apply functional options
	optionsWithPrompt := NewOptions(WithSystemPrompt("test prompt"))
	assertOptionsSystemPrompt(t, optionsWithPrompt, "test prompt")
}

// T016: Options with Tools
func TestOptionsWithTools(t *testing.T) {
	// Test Options with allowed_tools and disallowed_tools to match Python SDK
	options := NewOptions(
		WithAllowedTools("Read", "Write", "Edit"),
		WithDisallowedTools("Bash"),
	)

	// Verify allowed tools
	expectedAllowed := []string{"Read", "Write", "Edit"}
	assertOptionsStringSlice(t, options.AllowedTools, expectedAllowed, "AllowedTools")

	// Verify disallowed tools
	expectedDisallowed := []string{"Bash"}
	assertOptionsStringSlice(t, options.DisallowedTools, expectedDisallowed, "DisallowedTools")

	// Test with empty tools
	emptyOptions := NewOptions(
		WithAllowedTools(),
		WithDisallowedTools(),
	)
	assertOptionsStringSlice(t, emptyOptions.AllowedTools, []string{}, "AllowedTools")
	assertOptionsStringSlice(t, emptyOptions.DisallowedTools, []string{}, "DisallowedTools")
}

// T017: Permission Mode Options
func TestPermissionModeOptions(t *testing.T) {
	// Test all permission modes using table-driven approach
	tests := []struct {
		name string
		mode PermissionMode
	}{
		{"default", PermissionModeDefault},
		{"accept_edits", PermissionModeAcceptEdits},
		{"plan", PermissionModePlan},
		{"bypass_permissions", PermissionModeBypassPermissions},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := NewOptions(WithPermissionMode(test.mode))
			assertOptionsPermissionMode(t, options, test.mode)
		})
	}
}

// T018: System Prompt Options
func TestSystemPromptOptions(t *testing.T) {
	// Test system_prompt and append_system_prompt
	systemPrompt := "You are a helpful assistant."
	appendPrompt := "Be concise."

	options := NewOptions(
		WithSystemPrompt(systemPrompt),
		WithAppendSystemPrompt(appendPrompt),
	)

	// Verify system prompt is set
	assertOptionsSystemPrompt(t, options, systemPrompt)
	assertOptionsAppendSystemPrompt(t, options, appendPrompt)

	// Test with only system prompt
	systemOnlyOptions := NewOptions(WithSystemPrompt("Only system prompt"))
	assertOptionsSystemPrompt(t, systemOnlyOptions, "Only system prompt")
	assertOptionsAppendSystemPromptNil(t, systemOnlyOptions)

	// Test with only append prompt
	appendOnlyOptions := NewOptions(WithAppendSystemPrompt("Only append prompt"))
	assertOptionsAppendSystemPrompt(t, appendOnlyOptions, "Only append prompt")
	assertOptionsSystemPromptNil(t, appendOnlyOptions)
}

// T019: Session Continuation Options
func TestSessionContinuationOptions(t *testing.T) {
	// Test continue_conversation and resume options
	sessionID := "session-123"

	options := NewOptions(
		WithContinueConversation(true),
		WithResume(sessionID),
	)

	// Verify continue conversation is set
	assertOptionsContinueConversation(t, options, true)
	assertOptionsResume(t, options, sessionID)

	// Test with continue_conversation false
	falseOptions := NewOptions(WithContinueConversation(false))
	assertOptionsContinueConversation(t, falseOptions, false)
	assertOptionsResumeNil(t, falseOptions)

	// Test with only resume
	resumeOnlyOptions := NewOptions(WithResume("another-session"))
	assertOptionsResume(t, resumeOnlyOptions, "another-session")
	assertOptionsContinueConversation(t, resumeOnlyOptions, false) // default
}

// T020: Model Specification Options
func TestModelSpecificationOptions(t *testing.T) {
	// Test model and permission_prompt_tool_name
	model := "claude-3-5-sonnet-20241022"
	toolName := "CustomTool"

	options := NewOptions(
		WithModel(model),
		WithPermissionPromptToolName(toolName),
	)

	// Verify model and tool name are set
	assertOptionsModel(t, options, model)
	assertOptionsPermissionPromptToolName(t, options, toolName)

	// Test with only model
	modelOnlyOptions := NewOptions(WithModel("claude-opus-4"))
	assertOptionsModel(t, modelOnlyOptions, "claude-opus-4")
	assertOptionsPermissionPromptToolNameNil(t, modelOnlyOptions)

	// Test with only permission prompt tool name
	toolOnlyOptions := NewOptions(WithPermissionPromptToolName("OnlyTool"))
	assertOptionsPermissionPromptToolName(t, toolOnlyOptions, "OnlyTool")
	assertOptionsModelNil(t, toolOnlyOptions)
}

// T021: Functional Options Pattern
func TestFunctionalOptionsPattern(t *testing.T) {
	// Test chaining multiple functional options to create a fluent API
	options := NewOptions(
		WithSystemPrompt("You are a helpful assistant"),
		WithAllowedTools("Read", "Write"),
		WithDisallowedTools("Bash"),
		WithPermissionMode(PermissionModeAcceptEdits),
		WithModel("claude-3-5-sonnet-20241022"),
		WithContinueConversation(true),
		WithResume("session-456"),
		WithCwd("/tmp/test"),
		WithAddDirs("/tmp/dir1", "/tmp/dir2"),
		WithMaxThinkingTokens(10000),
		WithPermissionPromptToolName("CustomPermissionTool"),
	)

	// Verify all options are correctly applied
	if options.SystemPrompt == nil || *options.SystemPrompt != "You are a helpful assistant" {
		t.Errorf("Expected SystemPrompt = %q, got %v", "You are a helpful assistant", options.SystemPrompt)
	}

	expectedAllowed := []string{"Read", "Write"}
	if len(options.AllowedTools) != len(expectedAllowed) {
		t.Errorf("Expected AllowedTools length = %d, got %d", len(expectedAllowed), len(options.AllowedTools))
	}

	expectedDisallowed := []string{"Bash"}
	if len(options.DisallowedTools) != len(expectedDisallowed) {
		t.Errorf("Expected DisallowedTools length = %d, got %d", len(expectedDisallowed), len(options.DisallowedTools))
	}

	if options.PermissionMode == nil || *options.PermissionMode != PermissionModeAcceptEdits {
		t.Errorf("Expected PermissionMode = %q, got %v", PermissionModeAcceptEdits, options.PermissionMode)
	}

	if options.Model == nil || *options.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected Model = %q, got %v", "claude-3-5-sonnet-20241022", options.Model)
	}

	if options.ContinueConversation != true {
		t.Errorf("Expected ContinueConversation = true, got %v", options.ContinueConversation)
	}

	if options.Resume == nil || *options.Resume != "session-456" {
		t.Errorf("Expected Resume = %q, got %v", "session-456", options.Resume)
	}

	if options.Cwd == nil || *options.Cwd != "/tmp/test" {
		t.Errorf("Expected Cwd = %q, got %v", "/tmp/test", options.Cwd)
	}

	expectedAddDirs := []string{"/tmp/dir1", "/tmp/dir2"}
	if len(options.AddDirs) != len(expectedAddDirs) {
		t.Errorf("Expected AddDirs length = %d, got %d", len(expectedAddDirs), len(options.AddDirs))
	}

	if options.MaxThinkingTokens != 10000 {
		t.Errorf("Expected MaxThinkingTokens = 10000, got %d", options.MaxThinkingTokens)
	}

	if options.PermissionPromptToolName == nil || *options.PermissionPromptToolName != "CustomPermissionTool" {
		t.Errorf("Expected PermissionPromptToolName = %q, got %v", "CustomPermissionTool", options.PermissionPromptToolName)
	}
}

// T022: MCP Server Configuration
func TestMcpServerConfiguration(t *testing.T) {
	// Test all three MCP server configuration types: stdio, SSE, HTTP

	// Create MCP server configurations
	stdioConfig := &McpStdioServerConfig{
		Type:    McpServerTypeStdio,
		Command: "python",
		Args:    []string{"-m", "my_mcp_server"},
		Env:     map[string]string{"DEBUG": "1"},
	}

	sseConfig := &McpSSEServerConfig{
		Type:    McpServerTypeSSE,
		URL:     "http://localhost:8080/sse",
		Headers: map[string]string{"Authorization": "Bearer token123"},
	}

	httpConfig := &McpHTTPServerConfig{
		Type:    McpServerTypeHTTP,
		URL:     "http://localhost:8080/mcp",
		Headers: map[string]string{"Content-Type": "application/json"},
	}

	servers := map[string]McpServerConfig{
		"stdio_server": stdioConfig,
		"sse_server":   sseConfig,
		"http_server":  httpConfig,
	}

	options := NewOptions(WithMcpServers(servers))

	// Verify MCP servers are set
	if options.McpServers == nil {
		t.Error("Expected McpServers to be set, got nil")
	}

	if len(options.McpServers) != 3 {
		t.Errorf("Expected 3 MCP servers, got %d", len(options.McpServers))
	}

	// Test stdio server configuration
	stdioServer, exists := options.McpServers["stdio_server"]
	if !exists {
		t.Error("Expected stdio_server to exist")
	}
	if stdioServer.GetType() != McpServerTypeStdio {
		t.Errorf("Expected stdio server type = %q, got %q", McpServerTypeStdio, stdioServer.GetType())
	}

	stdioTyped, ok := stdioServer.(*McpStdioServerConfig)
	if !ok {
		t.Errorf("Expected *McpStdioServerConfig, got %T", stdioServer)
	} else {
		if stdioTyped.Command != "python" {
			t.Errorf("Expected Command = %q, got %q", "python", stdioTyped.Command)
		}
		if len(stdioTyped.Args) != 2 || stdioTyped.Args[0] != "-m" {
			t.Errorf("Expected Args = [-m my_mcp_server], got %v", stdioTyped.Args)
		}
		if stdioTyped.Env["DEBUG"] != "1" {
			t.Errorf("Expected Env[DEBUG] = %q, got %q", "1", stdioTyped.Env["DEBUG"])
		}
	}

	// Test SSE server configuration
	sseServer, exists := options.McpServers["sse_server"]
	if !exists {
		t.Error("Expected sse_server to exist")
	}
	if sseServer.GetType() != McpServerTypeSSE {
		t.Errorf("Expected SSE server type = %q, got %q", McpServerTypeSSE, sseServer.GetType())
	}

	sseTyped, ok := sseServer.(*McpSSEServerConfig)
	if !ok {
		t.Errorf("Expected *McpSSEServerConfig, got %T", sseServer)
	} else {
		if sseTyped.URL != "http://localhost:8080/sse" {
			t.Errorf("Expected URL = %q, got %q", "http://localhost:8080/sse", sseTyped.URL)
		}
		if sseTyped.Headers["Authorization"] != "Bearer token123" {
			t.Errorf("Expected Headers[Authorization] = %q, got %q", "Bearer token123", sseTyped.Headers["Authorization"])
		}
	}

	// Test HTTP server configuration
	httpServer, exists := options.McpServers["http_server"]
	if !exists {
		t.Error("Expected http_server to exist")
	}
	if httpServer.GetType() != McpServerTypeHTTP {
		t.Errorf("Expected HTTP server type = %q, got %q", McpServerTypeHTTP, httpServer.GetType())
	}

	httpTyped, ok := httpServer.(*McpHTTPServerConfig)
	if !ok {
		t.Errorf("Expected *McpHTTPServerConfig, got %T", httpServer)
	} else {
		if httpTyped.URL != "http://localhost:8080/mcp" {
			t.Errorf("Expected URL = %q, got %q", "http://localhost:8080/mcp", httpTyped.URL)
		}
		if httpTyped.Headers["Content-Type"] != "application/json" {
			t.Errorf("Expected Headers[Content-Type] = %q, got %q", "application/json", httpTyped.Headers["Content-Type"])
		}
	}
}

// T023: Extra Args Support
func TestExtraArgsSupport(t *testing.T) {
	// Test arbitrary CLI flag support via ExtraArgs map[string]*string

	// Create extra args - nil values represent boolean flags, non-nil represent flags with values
	debugFlag := "verbose"
	extraArgs := map[string]*string{
		"--debug":   &debugFlag,        // Flag with value: --debug=verbose
		"--verbose": nil,               // Boolean flag: --verbose
		"--output":  stringPtr("json"), // Flag with value: --output=json
		"--quiet":   nil,               // Boolean flag: --quiet
	}

	options := NewOptions(WithExtraArgs(extraArgs))

	// Verify extra args are set
	if options.ExtraArgs == nil {
		t.Error("Expected ExtraArgs to be set, got nil")
	}

	if len(options.ExtraArgs) != 4 {
		t.Errorf("Expected 4 extra args, got %d", len(options.ExtraArgs))
	}

	// Test flag with value
	debugValue, exists := options.ExtraArgs["--debug"]
	if !exists {
		t.Error("Expected --debug flag to exist")
	}
	if debugValue == nil {
		t.Error("Expected --debug to have a value, got nil")
		return
	}
	if *debugValue != "verbose" {
		t.Errorf("Expected --debug = %q, got %q", "verbose", *debugValue)
	}

	// Test boolean flag
	verboseValue, exists := options.ExtraArgs["--verbose"]
	if !exists {
		t.Error("Expected --verbose flag to exist")
	}
	if verboseValue != nil {
		t.Errorf("Expected --verbose to be boolean flag (nil), got %v", verboseValue)
	}

	// Test another flag with value
	outputValue, exists := options.ExtraArgs["--output"]
	if !exists {
		t.Error("Expected --output flag to exist")
	}
	if outputValue == nil {
		t.Error("Expected --output to have a value, got nil")
		return
	}
	if *outputValue != "json" {
		t.Errorf("Expected --output = %q, got %q", "json", *outputValue)
	}

	// Test another boolean flag
	quietValue, exists := options.ExtraArgs["--quiet"]
	if !exists {
		t.Error("Expected --quiet flag to exist")
	}
	if quietValue != nil {
		t.Errorf("Expected --quiet to be boolean flag (nil), got %v", quietValue)
	}

	// Test empty extra args
	emptyOptions := NewOptions(WithExtraArgs(map[string]*string{}))
	if emptyOptions.ExtraArgs == nil {
		t.Error("Expected ExtraArgs to be initialized, got nil")
	}
	if len(emptyOptions.ExtraArgs) != 0 {
		t.Errorf("Expected empty ExtraArgs, got %v", emptyOptions.ExtraArgs)
	}
}

// T024: Options Validation
func TestOptionsValidationIntegration(t *testing.T) {
	// Test that validation works through functional options API (detailed tests in internal/shared)
	validOptions := NewOptions(
		WithAllowedTools("Read", "Write"),
		WithMaxThinkingTokens(8000),
		WithSystemPrompt("Valid prompt"),
	)
	assertOptionsValidationError(t, validOptions, false, "valid options should pass validation")

	// Test that functional options can create invalid options that validation catches
	invalidOptions := NewOptions(WithMaxThinkingTokens(-100))
	assertOptionsValidationError(t, invalidOptions, true, "negative max thinking tokens should fail validation")
}

// T025: NewOptions Constructor
func TestNewOptionsConstructor(t *testing.T) {
	// Test Options creation with functional options applied correctly with defaults

	// Test NewOptions with no arguments should return defaults
	defaultOptions := NewOptions()
	assertOptionsMaxThinkingTokens(t, defaultOptions, 8000)
	assertOptionsStringSlice(t, defaultOptions.AllowedTools, []string{}, "AllowedTools")

	// Test NewOptions with single functional option
	singleOptionOptions := NewOptions(WithSystemPrompt("Single option test"))
	assertOptionsSystemPrompt(t, singleOptionOptions, "Single option test")
	// Should still have defaults for other fields
	assertOptionsMaxThinkingTokens(t, singleOptionOptions, 8000)

	// Test NewOptions with multiple functional options applied in order
	multipleOptions := NewOptions(
		WithMaxThinkingTokens(5000),               // Override default
		WithAllowedTools("Read"),                  // Add tools
		WithSystemPrompt("First prompt"),          // Set system prompt
		WithMaxThinkingTokens(12000),              // Override again (should win)
		WithAllowedTools("Read", "Write", "Edit"), // Override tools (should win)
		WithSystemPrompt("Second prompt"),         // Override again (should win)
		WithDisallowedTools("Bash"),
		WithPermissionMode(PermissionModeAcceptEdits),
		WithContinueConversation(true),
		WithMaxTurns(5),                        // Test WithMaxTurns
		WithSettings("/path/to/settings.json"), // Test WithSettings
	)

	// Verify options are applied in order (later options override earlier ones)
	assertOptionsMaxThinkingTokens(t, multipleOptions, 12000) // final override
	assertOptionsStringSlice(t, multipleOptions.AllowedTools, []string{"Read", "Write", "Edit"}, "AllowedTools")
	assertOptionsSystemPrompt(t, multipleOptions, "Second prompt") // final override
	assertOptionsStringSlice(t, multipleOptions.DisallowedTools, []string{"Bash"}, "DisallowedTools")
	assertOptionsPermissionMode(t, multipleOptions, PermissionModeAcceptEdits)
	assertOptionsContinueConversation(t, multipleOptions, true)
	assertOptionsMaxTurns(t, multipleOptions, 5)
	assertOptionsSettings(t, multipleOptions, "/path/to/settings.json")

	// Test that unmodified fields retain defaults
	assertOptionsResumeNil(t, multipleOptions)
	assertOptionsCwdNil(t, multipleOptions)

	// Test that maps are properly initialized even with options
	if multipleOptions.McpServers == nil {
		t.Error("Expected McpServers to be initialized, got nil")
	} else {
		assertOptionsMapInitialized(t, len(multipleOptions.McpServers), "McpServers")
	}

	if multipleOptions.ExtraArgs == nil {
		t.Error("Expected ExtraArgs to be initialized, got nil")
	} else {
		assertOptionsMapInitialized(t, len(multipleOptions.ExtraArgs), "ExtraArgs")
	}
}

// TestWithCLIPath tests the WithCLIPath option function
func TestWithCLIPath(t *testing.T) {
	tests := []struct {
		name     string
		cliPath  string
		expected *string
	}{
		{
			name:     "valid_cli_path",
			cliPath:  "/usr/local/bin/claude",
			expected: stringPtr("/usr/local/bin/claude"),
		},
		{
			name:     "relative_cli_path",
			cliPath:  "./claude",
			expected: stringPtr("./claude"),
		},
		{
			name:     "empty_cli_path",
			cliPath:  "",
			expected: stringPtr(""),
		},
		{
			name:     "windows_cli_path",
			cliPath:  "C:\\Program Files\\Claude\\claude.exe",
			expected: stringPtr("C:\\Program Files\\Claude\\claude.exe"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := NewOptions(WithCLIPath(test.cliPath))

			if options.CLIPath == nil && test.expected != nil {
				t.Errorf("Expected CLIPath to be set to %q, got nil", *test.expected)
			}

			if options.CLIPath != nil && test.expected == nil {
				t.Errorf("Expected CLIPath to be nil, got %q", *options.CLIPath)
			}

			if options.CLIPath != nil && test.expected != nil && *options.CLIPath != *test.expected {
				t.Errorf("Expected CLIPath %q, got %q", *test.expected, *options.CLIPath)
			}
		})
	}

	// Test integration with other options
	t.Run("cli_path_with_other_options", func(t *testing.T) {
		options := NewOptions(
			WithCLIPath("/custom/claude"),
			WithSystemPrompt("Test system prompt"),
			WithModel("claude-sonnet-3-5-20241022"),
		)

		if options.CLIPath == nil || *options.CLIPath != "/custom/claude" {
			t.Errorf("Expected CLIPath to be preserved with other options")
		}

		assertOptionsSystemPrompt(t, options, "Test system prompt")
		assertOptionsModel(t, options, "claude-sonnet-3-5-20241022")
	})
}

// TestWithTransport tests the WithTransport option function
func TestWithTransport(t *testing.T) {
	// Create a mock transport for testing
	mockTransport := &mockTransportForOptions{}

	t.Run("transport_marker_in_extra_args", func(t *testing.T) {
		options := NewOptions(WithTransport(mockTransport))

		if options.ExtraArgs == nil {
			t.Fatal("Expected ExtraArgs to be initialized")
		}

		marker, exists := options.ExtraArgs["__transport_marker__"]
		if !exists {
			t.Error("Expected transport marker to be set in ExtraArgs")
		}

		if marker == nil || *marker != customTransportMarker {
			t.Errorf("Expected transport marker value 'custom_transport', got %v", marker)
		}
	})

	t.Run("transport_with_existing_extra_args", func(t *testing.T) {
		options := NewOptions(
			WithExtraArgs(map[string]*string{"existing": stringPtr("value")}),
			WithTransport(mockTransport),
		)

		if options.ExtraArgs == nil {
			t.Fatal("Expected ExtraArgs to be preserved")
		}

		// Check existing arg is preserved
		existing, exists := options.ExtraArgs["existing"]
		if !exists || existing == nil || *existing != "value" {
			t.Error("Expected existing ExtraArgs to be preserved")
		}

		// Check transport marker is added
		marker, exists := options.ExtraArgs["__transport_marker__"]
		if !exists || marker == nil || *marker != customTransportMarker {
			t.Error("Expected transport marker to be added to existing ExtraArgs")
		}
	})

	t.Run("transport_with_nil_extra_args", func(t *testing.T) {
		// Create options with nil ExtraArgs
		options := &Options{}

		// Apply WithTransport option
		WithTransport(mockTransport)(options)

		if options.ExtraArgs == nil {
			t.Error("Expected ExtraArgs to be initialized")
		}

		marker, exists := options.ExtraArgs["__transport_marker__"]
		if !exists || marker == nil || *marker != customTransportMarker {
			t.Error("Expected transport marker to be set when ExtraArgs was nil")
		}
	})

	t.Run("multiple_transport_calls", func(t *testing.T) {
		anotherMockTransport := &mockTransportForOptions{}

		options := NewOptions(
			WithTransport(mockTransport),
			WithTransport(anotherMockTransport), // Should overwrite
		)

		// Should only have one transport marker (last one wins)
		marker, exists := options.ExtraArgs["__transport_marker__"]
		if !exists || marker == nil || *marker != customTransportMarker {
			t.Error("Expected last transport to set the marker")
		}
	})
}

// Helper Functions - following client_test.go patterns

// assertOptionsMaxThinkingTokens verifies MaxThinkingTokens value
func assertOptionsMaxThinkingTokens(t *testing.T, options *Options, expected int) {
	t.Helper()
	if options.MaxThinkingTokens != expected {
		t.Errorf("Expected MaxThinkingTokens = %d, got %d", expected, options.MaxThinkingTokens)
	}
}

// assertOptionsSystemPrompt verifies SystemPrompt value
func assertOptionsSystemPrompt(t *testing.T, options *Options, expected string) {
	t.Helper()
	if options.SystemPrompt == nil {
		t.Error("Expected SystemPrompt to be set, got nil")
		return
	}
	actual := *options.SystemPrompt
	if actual != expected {
		t.Errorf("Expected SystemPrompt = %q, got %q", expected, actual)
	}
}

// assertOptionsSystemPromptNil verifies SystemPrompt is nil
func assertOptionsSystemPromptNil(t *testing.T, options *Options) {
	t.Helper()
	if options.SystemPrompt != nil {
		t.Errorf("Expected SystemPrompt = nil, got %v", *options.SystemPrompt)
	}
}

// assertOptionsAppendSystemPrompt verifies AppendSystemPrompt value
func assertOptionsAppendSystemPrompt(t *testing.T, options *Options, expected string) {
	t.Helper()
	if options.AppendSystemPrompt == nil {
		t.Error("Expected AppendSystemPrompt to be set, got nil")
		return
	}
	if *options.AppendSystemPrompt != expected {
		t.Errorf("Expected AppendSystemPrompt = %q, got %q", expected, *options.AppendSystemPrompt)
	}
}

// assertOptionsAppendSystemPromptNil verifies AppendSystemPrompt is nil
func assertOptionsAppendSystemPromptNil(t *testing.T, options *Options) {
	t.Helper()
	if options.AppendSystemPrompt != nil {
		t.Errorf("Expected AppendSystemPrompt = nil, got %v", *options.AppendSystemPrompt)
	}
}

// assertOptionsStringSlice verifies string slice values
func assertOptionsStringSlice(t *testing.T, actual, expected []string, fieldName string) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Errorf("Expected %s length = %d, got %d", fieldName, len(expected), len(actual))
		return
	}
	for i, expectedVal := range expected {
		if i >= len(actual) || actual[i] != expectedVal {
			t.Errorf("Expected %s[%d] = %q, got %q", fieldName, i, expectedVal, actual[i])
		}
	}
}

// assertOptionsPermissionMode verifies PermissionMode value
func assertOptionsPermissionMode(t *testing.T, options *Options, expected PermissionMode) {
	t.Helper()
	if options.PermissionMode == nil {
		t.Error("Expected PermissionMode to be set, got nil")
		return
	}
	if *options.PermissionMode != expected {
		t.Errorf("Expected PermissionMode = %q, got %q", expected, *options.PermissionMode)
	}
}

// assertOptionsContinueConversation verifies ContinueConversation value
func assertOptionsContinueConversation(t *testing.T, options *Options, expected bool) {
	t.Helper()
	if options.ContinueConversation != expected {
		t.Errorf("Expected ContinueConversation = %v, got %v", expected, options.ContinueConversation)
	}
}

// assertOptionsResume verifies Resume value
func assertOptionsResume(t *testing.T, options *Options, expected string) {
	t.Helper()
	if options.Resume == nil {
		t.Error("Expected Resume to be set, got nil")
		return
	}
	if *options.Resume != expected {
		t.Errorf("Expected Resume = %q, got %q", expected, *options.Resume)
	}
}

// assertOptionsResumeNil verifies Resume is nil
func assertOptionsResumeNil(t *testing.T, options *Options) {
	t.Helper()
	if options.Resume != nil {
		t.Errorf("Expected Resume = nil, got %v", *options.Resume)
	}
}

// assertOptionsModel verifies Model value
func assertOptionsModel(t *testing.T, options *Options, expected string) {
	t.Helper()
	if options.Model == nil {
		t.Error("Expected Model to be set, got nil")
		return
	}
	if *options.Model != expected {
		t.Errorf("Expected Model = %q, got %q", expected, *options.Model)
	}
}

// assertOptionsModelNil verifies Model is nil
func assertOptionsModelNil(t *testing.T, options *Options) {
	t.Helper()
	if options.Model != nil {
		t.Errorf("Expected Model = nil, got %v", *options.Model)
	}
}

// assertOptionsPermissionPromptToolName verifies PermissionPromptToolName value
func assertOptionsPermissionPromptToolName(t *testing.T, options *Options, expected string) {
	t.Helper()
	if options.PermissionPromptToolName == nil {
		t.Error("Expected PermissionPromptToolName to be set, got nil")
		return
	}
	if *options.PermissionPromptToolName != expected {
		t.Errorf("Expected PermissionPromptToolName = %q, got %q", expected, *options.PermissionPromptToolName)
	}
}

// assertOptionsPermissionPromptToolNameNil verifies PermissionPromptToolName is nil
func assertOptionsPermissionPromptToolNameNil(t *testing.T, options *Options) {
	t.Helper()
	if options.PermissionPromptToolName != nil {
		t.Errorf("Expected PermissionPromptToolName = nil, got %v", *options.PermissionPromptToolName)
	}
}

// assertOptionsCwdNil verifies Cwd is nil
func assertOptionsCwdNil(t *testing.T, options *Options) {
	t.Helper()
	if options.Cwd != nil {
		t.Errorf("Expected Cwd = nil, got %v", *options.Cwd)
	}
}

// assertOptionsMaxTurns verifies MaxTurns value
func assertOptionsMaxTurns(t *testing.T, options *Options, expected int) {
	t.Helper()
	if options.MaxTurns != expected {
		t.Errorf("Expected MaxTurns = %d, got %d", expected, options.MaxTurns)
	}
}

// assertOptionsSettings verifies Settings value
func assertOptionsSettings(t *testing.T, options *Options, expected string) {
	t.Helper()
	if options.Settings == nil {
		t.Error("Expected Settings to be set, got nil")
		return
	}
	if *options.Settings != expected {
		t.Errorf("Expected Settings = %q, got %q", expected, *options.Settings)
	}
}

// assertOptionsMapInitialized verifies a map field is initialized but empty
func assertOptionsMapInitialized(t *testing.T, actualLen int, fieldName string) {
	t.Helper()
	if actualLen != 0 {
		t.Errorf("Expected %s = {} (empty but initialized), got length %d", fieldName, actualLen)
	}
}

// assertOptionsValidationError verifies validation returns error
func assertOptionsValidationError(t *testing.T, options *Options, shouldError bool, description string) {
	t.Helper()
	err := options.Validate()
	if shouldError && err == nil {
		t.Errorf("%s: expected validation error, got nil", description)
	}
	if !shouldError && err != nil {
		t.Errorf("%s: expected no validation error, got: %v", description, err)
	}
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}

// mockTransportForOptions is a minimal mock transport for testing options
type mockTransportForOptions struct{}

func (m *mockTransportForOptions) Connect(_ context.Context) error { return nil }
func (m *mockTransportForOptions) SendMessage(_ context.Context, _ StreamMessage) error {
	return nil
}

func (m *mockTransportForOptions) ReceiveMessages(_ context.Context) (<-chan Message, <-chan error) {
	return nil, nil
}
func (m *mockTransportForOptions) Interrupt(_ context.Context) error { return nil }
func (m *mockTransportForOptions) Close() error                      { return nil }
func (m *mockTransportForOptions) GetValidator() *StreamValidator    { return &StreamValidator{} }

// TestWithEnvOptions tests environment variable functional options following table-driven pattern
func TestWithEnvOptions(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() *Options
		expected  map[string]string
		wantPanic bool
	}{
		{
			name: "single_env_var",
			setup: func() *Options {
				return NewOptions(WithEnvVar("DEBUG", "1"))
			},
			expected: map[string]string{"DEBUG": "1"},
		},
		{
			name: "multiple_env_vars",
			setup: func() *Options {
				return NewOptions(WithEnv(map[string]string{
					"HTTP_PROXY": "http://proxy:8080",
					"CUSTOM_VAR": "value",
				}))
			},
			expected: map[string]string{
				"HTTP_PROXY": "http://proxy:8080",
				"CUSTOM_VAR": "value",
			},
		},
		{
			name: "merge_with_env_and_envvar",
			setup: func() *Options {
				return NewOptions(
					WithEnv(map[string]string{"VAR1": "val1"}),
					WithEnvVar("VAR2", "val2"),
				)
			},
			expected: map[string]string{
				"VAR1": "val1",
				"VAR2": "val2",
			},
		},
		{
			name: "override_existing",
			setup: func() *Options {
				return NewOptions(
					WithEnvVar("KEY", "original"),
					WithEnvVar("KEY", "updated"),
				)
			},
			expected: map[string]string{"KEY": "updated"},
		},
		{
			name: "empty_env_map",
			setup: func() *Options {
				return NewOptions(WithEnv(map[string]string{}))
			},
			expected: map[string]string{},
		},
		{
			name: "nil_env_map_initializes",
			setup: func() *Options {
				opts := &Options{} // ExtraEnv is nil
				WithEnvVar("TEST", "value")(opts)
				return opts
			},
			expected: map[string]string{"TEST": "value"},
		},
		{
			name: "proxy_configuration_example",
			setup: func() *Options {
				return NewOptions(
					WithEnv(map[string]string{
						"HTTP_PROXY":  "http://proxy.example.com:8080",
						"HTTPS_PROXY": "http://proxy.example.com:8080",
						"NO_PROXY":    "localhost,127.0.0.1",
					}),
				)
			},
			expected: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"HTTPS_PROXY": "http://proxy.example.com:8080",
				"NO_PROXY":    "localhost,127.0.0.1",
			},
		},
		{
			name: "path_override_example",
			setup: func() *Options {
				return NewOptions(
					WithEnvVar("PATH", "/custom/bin:/usr/bin"),
				)
			},
			expected: map[string]string{
				"PATH": "/custom/bin:/usr/bin",
			},
		},
		{
			name: "nil_env_map_to_WithEnv",
			setup: func() *Options {
				opts := &Options{} // ExtraEnv is nil
				WithEnv(map[string]string{"TEST": "value"})(opts)
				return opts
			},
			expected: map[string]string{"TEST": "value"},
		},
		{
			name: "nil_map_passed_to_WithEnv",
			setup: func() *Options {
				return NewOptions(WithEnv(nil))
			},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := tt.setup()
			assertEnvVars(t, options.ExtraEnv, tt.expected)
		})
	}
}

// TestWithExtraFlag tests the WithExtraFlag convenience helper
func TestWithExtraFlag(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Options
		flagName string
		wantNil  bool
	}{
		{
			name: "single_flag",
			setup: func() *Options {
				return NewOptions(WithExtraFlag("fork-session"))
			},
			flagName: "fork-session",
			wantNil:  true,
		},
		{
			name: "multiple_flags",
			setup: func() *Options {
				return NewOptions(
					WithExtraFlag("verbose"),
					WithExtraFlag("debug"),
				)
			},
			flagName: "verbose",
			wantNil:  true,
		},
		{
			name: "flag_with_dashes",
			setup: func() *Options {
				return NewOptions(WithExtraFlag("--fork-session"))
			},
			flagName: "--fork-session",
			wantNil:  true,
		},
		{
			name: "flag_on_nil_extraargs",
			setup: func() *Options {
				opts := &Options{} // ExtraArgs is nil
				WithExtraFlag("test-flag")(opts)
				return opts
			},
			flagName: "test-flag",
			wantNil:  true,
		},
		{
			name: "flag_with_existing_args",
			setup: func() *Options {
				return NewOptions(
					WithExtraArg("setting", "value"),
					WithExtraFlag("bool-flag"),
				)
			},
			flagName: "bool-flag",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := tt.setup()

			if options.ExtraArgs == nil {
				t.Fatal("Expected ExtraArgs to be initialized")
			}

			value, exists := options.ExtraArgs[tt.flagName]
			if !exists {
				t.Errorf("Expected flag %q to exist", tt.flagName)
			}

			if tt.wantNil && value != nil {
				t.Errorf("Expected flag %q to be nil (boolean flag), got %v", tt.flagName, value)
			}

			if !tt.wantNil && value == nil {
				t.Errorf("Expected flag %q to have value, got nil", tt.flagName)
			}
		})
	}

	// Test multiple flags are preserved
	t.Run("multiple_flags_preserved", func(t *testing.T) {
		options := NewOptions(
			WithExtraFlag("flag1"),
			WithExtraFlag("flag2"),
			WithExtraFlag("flag3"),
		)

		if len(options.ExtraArgs) != 3 {
			t.Errorf("Expected 3 flags, got %d", len(options.ExtraArgs))
		}

		for _, flag := range []string{"flag1", "flag2", "flag3"} {
			if value, exists := options.ExtraArgs[flag]; !exists || value != nil {
				t.Errorf("Expected flag %q to exist as boolean flag", flag)
			}
		}
	})
}

// TestWithExtraArg tests the WithExtraArg convenience helper
func TestWithExtraArg(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() *Options
		argName   string
		wantValue string
	}{
		{
			name: "single_arg",
			setup: func() *Options {
				return NewOptions(WithExtraArg("output", "json"))
			},
			argName:   "output",
			wantValue: "json",
		},
		{
			name: "multiple_args",
			setup: func() *Options {
				return NewOptions(
					WithExtraArg("format", "xml"),
					WithExtraArg("level", "debug"),
				)
			},
			argName:   "format",
			wantValue: "xml",
		},
		{
			name: "arg_with_dashes",
			setup: func() *Options {
				return NewOptions(WithExtraArg("--output-file", "/tmp/out.txt"))
			},
			argName:   "--output-file",
			wantValue: "/tmp/out.txt",
		},
		{
			name: "arg_on_nil_extraargs",
			setup: func() *Options {
				opts := &Options{} // ExtraArgs is nil
				WithExtraArg("test-arg", "test-value")(opts)
				return opts
			},
			argName:   "test-arg",
			wantValue: "test-value",
		},
		{
			name: "arg_with_existing_flag",
			setup: func() *Options {
				return NewOptions(
					WithExtraFlag("bool-flag"),
					WithExtraArg("setting", "value"),
				)
			},
			argName:   "setting",
			wantValue: "value",
		},
		{
			name: "empty_value",
			setup: func() *Options {
				return NewOptions(WithExtraArg("empty", ""))
			},
			argName:   "empty",
			wantValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := tt.setup()

			if options.ExtraArgs == nil {
				t.Fatal("Expected ExtraArgs to be initialized")
			}

			value, exists := options.ExtraArgs[tt.argName]
			if !exists {
				t.Errorf("Expected arg %q to exist", tt.argName)
				return
			}

			if value == nil {
				t.Errorf("Expected arg %q to have value, got nil", tt.argName)
				return
			}

			if *value != tt.wantValue {
				t.Errorf("Expected arg %q value = %q, got %q", tt.argName, tt.wantValue, *value)
			}
		})
	}

	// Test multiple args are preserved
	t.Run("multiple_args_preserved", func(t *testing.T) {
		options := NewOptions(
			WithExtraArg("arg1", "value1"),
			WithExtraArg("arg2", "value2"),
			WithExtraArg("arg3", "value3"),
		)

		if len(options.ExtraArgs) != 3 {
			t.Errorf("Expected 3 args, got %d", len(options.ExtraArgs))
		}

		expected := map[string]string{
			"arg1": "value1",
			"arg2": "value2",
			"arg3": "value3",
		}

		for name, wantValue := range expected {
			value, exists := options.ExtraArgs[name]
			if !exists || value == nil || *value != wantValue {
				t.Errorf("Expected arg %q = %q", name, wantValue)
			}
		}
	})
}

// TestExtraHelpersIntegration tests WithExtraFlag and WithExtraArg together
func TestExtraHelpersIntegration(t *testing.T) {
	t.Run("mixed_flags_and_args", func(t *testing.T) {
		options := NewOptions(
			WithExtraFlag("verbose"),
			WithExtraArg("output", "json"),
			WithExtraFlag("debug"),
			WithExtraArg("level", "info"),
		)

		// Check flags
		if value, exists := options.ExtraArgs["verbose"]; !exists || value != nil {
			t.Error("Expected verbose to be boolean flag")
		}
		if value, exists := options.ExtraArgs["debug"]; !exists || value != nil {
			t.Error("Expected debug to be boolean flag")
		}

		// Check args
		if value, exists := options.ExtraArgs["output"]; !exists || value == nil || *value != "json" {
			t.Error("Expected output = json")
		}
		if value, exists := options.ExtraArgs["level"]; !exists || value == nil || *value != "info" {
			t.Error("Expected level = info")
		}

		if len(options.ExtraArgs) != 4 {
			t.Errorf("Expected 4 total args, got %d", len(options.ExtraArgs))
		}
	})

	t.Run("with_other_options", func(t *testing.T) {
		options := NewOptions(
			WithSystemPrompt("Test prompt"),
			WithExtraFlag("fork-session"),
			WithModel("claude-sonnet-3-5-20241022"),
			WithExtraArg("custom", "value"),
		)

		// Check other options are preserved
		assertOptionsSystemPrompt(t, options, "Test prompt")
		assertOptionsModel(t, options, "claude-sonnet-3-5-20241022")

		// Check extra args
		if value, exists := options.ExtraArgs["fork-session"]; !exists || value != nil {
			t.Error("Expected fork-session to be boolean flag")
		}
		if value, exists := options.ExtraArgs["custom"]; !exists || value == nil || *value != "value" {
			t.Error("Expected custom = value")
		}
	})

	t.Run("override_with_WithExtraArgs", func(t *testing.T) {
		// Test that WithExtraArgs replaces individual flags
		options := NewOptions(
			WithExtraFlag("flag1"),
			WithExtraArg("arg1", "value1"),
			WithExtraArgs(map[string]*string{
				"new-flag": nil,
				"new-arg":  stringPtr("new-value"),
			}),
		)

		// Previous flags should be replaced
		if _, exists := options.ExtraArgs["flag1"]; exists {
			t.Error("Expected flag1 to be replaced")
		}
		if _, exists := options.ExtraArgs["arg1"]; exists {
			t.Error("Expected arg1 to be replaced")
		}

		// New flags should exist
		if value, exists := options.ExtraArgs["new-flag"]; !exists || value != nil {
			t.Error("Expected new-flag to be boolean flag")
		}
		if value, exists := options.ExtraArgs["new-arg"]; !exists || value == nil || *value != "new-value" {
			t.Error("Expected new-arg = new-value")
		}
	})

	t.Run("add_after_WithExtraArgs", func(t *testing.T) {
		// Test that individual helpers work after WithExtraArgs
		options := NewOptions(
			WithExtraArgs(map[string]*string{
				"initial-flag": nil,
			}),
			WithExtraFlag("added-flag"),
			WithExtraArg("added-arg", "added-value"),
		)

		// All flags should exist
		if value, exists := options.ExtraArgs["initial-flag"]; !exists || value != nil {
			t.Error("Expected initial-flag to be boolean flag")
		}
		if value, exists := options.ExtraArgs["added-flag"]; !exists || value != nil {
			t.Error("Expected added-flag to be boolean flag")
		}
		if value, exists := options.ExtraArgs["added-arg"]; !exists || value == nil || *value != "added-value" {
			t.Error("Expected added-arg = added-value")
		}

		if len(options.ExtraArgs) != 3 {
			t.Errorf("Expected 3 args, got %d", len(options.ExtraArgs))
		}
	})

	t.Run("real_world_example_fork_session", func(t *testing.T) {
		// Example from issue: WithExtraFlag('fork-session')
		options := NewOptions(
			WithSystemPrompt("You are a helpful assistant"),
			WithExtraFlag("fork-session"),
			WithModel("claude-sonnet-3-5-20241022"),
		)

		// Verify fork-session flag
		if value, exists := options.ExtraArgs["fork-session"]; !exists || value != nil {
			t.Error("Expected fork-session to be boolean flag")
		}

		// Verify other options
		assertOptionsSystemPrompt(t, options, "You are a helpful assistant")
		assertOptionsModel(t, options, "claude-sonnet-3-5-20241022")
	})
}

// TestExtraHelperEdgeCases tests edge cases and error conditions
func TestExtraHelperEdgeCases(t *testing.T) {
	t.Run("empty_flag_name", func(t *testing.T) {
		options := NewOptions(WithExtraFlag(""))

		if _, exists := options.ExtraArgs[""]; !exists {
			t.Error("Expected empty flag name to be accepted")
		}
	})

	t.Run("empty_arg_name", func(t *testing.T) {
		options := NewOptions(WithExtraArg("", "value"))

		if _, exists := options.ExtraArgs[""]; !exists {
			t.Error("Expected empty arg name to be accepted")
		}
	})

	t.Run("duplicate_flag", func(t *testing.T) {
		options := NewOptions(
			WithExtraFlag("duplicate"),
			WithExtraFlag("duplicate"), // Should overwrite
		)

		if len(options.ExtraArgs) != 1 {
			t.Errorf("Expected 1 arg after duplicate, got %d", len(options.ExtraArgs))
		}
	})

	t.Run("duplicate_arg", func(t *testing.T) {
		options := NewOptions(
			WithExtraArg("duplicate", "first"),
			WithExtraArg("duplicate", "second"), // Should overwrite
		)

		if len(options.ExtraArgs) != 1 {
			t.Errorf("Expected 1 arg after duplicate, got %d", len(options.ExtraArgs))
		}

		if value, exists := options.ExtraArgs["duplicate"]; !exists || value == nil || *value != "second" {
			t.Error("Expected duplicate = second (last wins)")
		}
	})

	t.Run("flag_then_arg_same_name", func(t *testing.T) {
		options := NewOptions(
			WithExtraFlag("same"),
			WithExtraArg("same", "value"), // Should convert to valued arg
		)

		if len(options.ExtraArgs) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(options.ExtraArgs))
		}

		if value, exists := options.ExtraArgs["same"]; !exists || value == nil || *value != "value" {
			t.Error("Expected same = value (arg overwrites flag)")
		}
	})

	t.Run("arg_then_flag_same_name", func(t *testing.T) {
		options := NewOptions(
			WithExtraArg("same", "value"),
			WithExtraFlag("same"), // Should convert to boolean flag
		)

		if len(options.ExtraArgs) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(options.ExtraArgs))
		}

		if value, exists := options.ExtraArgs["same"]; !exists || value != nil {
			t.Error("Expected same to be boolean flag (flag overwrites arg)")
		}
	})

	t.Run("value_with_special_chars", func(t *testing.T) {
		specialValue := "path/to/file with spaces & special=chars"
		options := NewOptions(WithExtraArg("path", specialValue))

		if value, exists := options.ExtraArgs["path"]; !exists || value == nil || *value != specialValue {
			t.Errorf("Expected path = %q", specialValue)
		}
	})
}

// TestWithForkSession tests the WithForkSession convenience option
func TestWithForkSession(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Options
		validate func(*testing.T, *Options)
	}{
		{
			name: "fork_session_alone",
			setup: func() *Options {
				return NewOptions(WithForkSession())
			},
			validate: func(t *testing.T, opts *Options) {
				t.Helper()
				if opts.ExtraArgs == nil {
					t.Fatal("Expected ExtraArgs to be initialized")
				}
				value, exists := opts.ExtraArgs["fork-session"]
				if !exists {
					t.Error("Expected fork-session flag to exist")
				}
				if value != nil {
					t.Errorf("Expected fork-session to be nil (boolean flag), got %v", value)
				}
			},
		},
		{
			name: "fork_session_with_resume",
			setup: func() *Options {
				return NewOptions(
					WithResume("original-session-uuid"),
					WithForkSession(),
				)
			},
			validate: func(t *testing.T, opts *Options) {
				t.Helper()
				// Verify resume is set
				if opts.Resume == nil || *opts.Resume != "original-session-uuid" {
					t.Error("Expected Resume to be set to 'original-session-uuid'")
				}
				// Verify fork-session flag
				value, exists := opts.ExtraArgs["fork-session"]
				if !exists {
					t.Error("Expected fork-session flag to exist")
				}
				if value != nil {
					t.Errorf("Expected fork-session to be nil (boolean flag), got %v", value)
				}
			},
		},
		{
			name: "fork_session_with_other_options",
			setup: func() *Options {
				return NewOptions(
					WithSystemPrompt("Test prompt"),
					WithResume("session-123"),
					WithForkSession(),
					WithModel("claude-3-5-sonnet-20241022"),
					WithMaxTurns(10),
				)
			},
			validate: func(t *testing.T, opts *Options) {
				t.Helper()
				// Verify all options are preserved
				if opts.SystemPrompt == nil || *opts.SystemPrompt != "Test prompt" {
					t.Error("Expected SystemPrompt to be preserved")
				}
				if opts.Resume == nil || *opts.Resume != "session-123" {
					t.Error("Expected Resume to be preserved")
				}
				if opts.Model == nil || *opts.Model != "claude-3-5-sonnet-20241022" {
					t.Error("Expected Model to be preserved")
				}
				if opts.MaxTurns != 10 {
					t.Error("Expected MaxTurns to be preserved")
				}
				// Verify fork-session flag
				value, exists := opts.ExtraArgs["fork-session"]
				if !exists {
					t.Error("Expected fork-session flag to exist")
				}
				if value != nil {
					t.Errorf("Expected fork-session to be nil (boolean flag), got %v", value)
				}
			},
		},
		{
			name: "fork_session_with_other_extra_args",
			setup: func() *Options {
				return NewOptions(
					WithExtraFlag("verbose"),
					WithExtraArg("setting", "value"),
					WithForkSession(),
					WithExtraFlag("debug"),
				)
			},
			validate: func(t *testing.T, opts *Options) {
				t.Helper()
				if len(opts.ExtraArgs) != 4 {
					t.Errorf("Expected 4 ExtraArgs, got %d", len(opts.ExtraArgs))
				}
				// Verify all flags exist
				expectedFlags := map[string]bool{
					"verbose":      true,
					"fork-session": true,
					"debug":        true,
				}
				for flag := range expectedFlags {
					value, exists := opts.ExtraArgs[flag]
					if !exists {
						t.Errorf("Expected %q flag to exist", flag)
					}
					if value != nil {
						t.Errorf("Expected %q to be nil (boolean flag), got %v", flag, value)
					}
				}
				// Verify valued arg
				value, exists := opts.ExtraArgs["setting"]
				if !exists || value == nil || *value != "value" {
					t.Error("Expected setting arg to be 'value'")
				}
			},
		},
		{
			name: "fork_session_multiple_times",
			setup: func() *Options {
				return NewOptions(
					WithForkSession(),
					WithForkSession(), // Should be idempotent
				)
			},
			validate: func(t *testing.T, opts *Options) {
				t.Helper()
				if len(opts.ExtraArgs) != 1 {
					t.Errorf("Expected 1 ExtraArg, got %d", len(opts.ExtraArgs))
				}
				value, exists := opts.ExtraArgs["fork-session"]
				if !exists || value != nil {
					t.Error("Expected fork-session to be boolean flag")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := tt.setup()
			tt.validate(t, options)
		})
	}
}

// TestWithForkSessionIntegration tests WithForkSession in realistic scenarios
func TestWithForkSessionIntegration(t *testing.T) {
	t.Run("conversation_branching_scenario", func(t *testing.T) {
		// Simulate a conversation branching workflow
		originalSessionID := "uuid-original-session"

		// Create options for forking from original session
		forkOptions := NewOptions(
			WithResume(originalSessionID),
			WithForkSession(),
			WithSystemPrompt("Continue with alternative approach"),
		)

		// Verify configuration
		if forkOptions.Resume == nil || *forkOptions.Resume != originalSessionID {
			t.Error("Expected Resume to reference original session")
		}

		value, exists := forkOptions.ExtraArgs["fork-session"]
		if !exists || value != nil {
			t.Error("Expected fork-session flag to be set correctly")
		}

		if forkOptions.SystemPrompt == nil || *forkOptions.SystemPrompt != "Continue with alternative approach" {
			t.Error("Expected SystemPrompt to be preserved")
		}
	})

	t.Run("parallel_experimentation_scenario", func(t *testing.T) {
		// Simulate creating multiple forks for parallel experimentation
		baseSessionID := "uuid-base-session"

		// Fork 1: Optimization approach
		fork1 := NewOptions(
			WithResume(baseSessionID),
			WithForkSession(),
			WithExtraArg("experiment", "optimization"),
		)

		// Fork 2: Alternative approach
		fork2 := NewOptions(
			WithResume(baseSessionID),
			WithForkSession(),
			WithExtraArg("experiment", "alternative"),
		)

		// Both should reference the same base session
		if fork1.Resume == nil || *fork1.Resume != baseSessionID {
			t.Error("Fork 1 should reference base session")
		}
		if fork2.Resume == nil || *fork2.Resume != baseSessionID {
			t.Error("Fork 2 should reference base session")
		}

		// Both should have fork-session flag
		for i, fork := range []*Options{fork1, fork2} {
			value, exists := fork.ExtraArgs["fork-session"]
			if !exists || value != nil {
				t.Errorf("Fork %d should have fork-session flag", i+1)
			}
		}

		// Each should have unique experiment tags
		if exp1, exists := fork1.ExtraArgs["experiment"]; !exists || exp1 == nil || *exp1 != "optimization" {
			t.Error("Fork 1 should have optimization experiment tag")
		}
		if exp2, exists := fork2.ExtraArgs["experiment"]; !exists || exp2 == nil || *exp2 != "alternative" {
			t.Error("Fork 2 should have alternative experiment tag")
		}
	})

	t.Run("recovery_scenario", func(t *testing.T) {
		// Simulate recovery from a known good state
		knownGoodSessionID := "uuid-known-good-state"

		recoveryOptions := NewOptions(
			WithResume(knownGoodSessionID),
			WithForkSession(),
			WithSystemPrompt("Retry with corrected approach"),
			WithExtraArg("retry-attempt", "2"),
		)

		// Verify all components for recovery scenario
		if recoveryOptions.Resume == nil || *recoveryOptions.Resume != knownGoodSessionID {
			t.Error("Expected Resume to reference known good state")
		}

		value, exists := recoveryOptions.ExtraArgs["fork-session"]
		if !exists || value != nil {
			t.Error("Expected fork-session flag for recovery")
		}

		if retry, exists := recoveryOptions.ExtraArgs["retry-attempt"]; !exists || retry == nil || *retry != "2" {
			t.Error("Expected retry-attempt metadata")
		}
	})
}

// TestWithEnvIntegration tests environment variable options integration with other options
func TestWithEnvIntegration(t *testing.T) {
	options := NewOptions(
		WithSystemPrompt("You are a helpful assistant"),
		WithEnvVar("DEBUG", "1"),
		WithModel("claude-3-5-sonnet-20241022"),
		WithEnv(map[string]string{
			"HTTP_PROXY": "http://proxy:8080",
			"CUSTOM":     "value",
		}),
		WithEnvVar("OVERRIDE", "final"),
	)

	// Test that env vars are correctly set
	expected := map[string]string{
		"DEBUG":      "1",
		"HTTP_PROXY": "http://proxy:8080",
		"CUSTOM":     "value",
		"OVERRIDE":   "final",
	}
	assertEnvVars(t, options.ExtraEnv, expected)

	// Test that other options are preserved
	assertOptionsSystemPrompt(t, options, "You are a helpful assistant")
	assertOptionsModel(t, options, "claude-3-5-sonnet-20241022")
}

// Helper function following client_test.go patterns
func assertEnvVars(t *testing.T, actual, expected map[string]string) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Errorf("Expected %d env vars, got %d. Expected: %v, Actual: %v",
			len(expected), len(actual), expected, actual)
		return
	}
	for k, v := range expected {
		if actual[k] != v {
			t.Errorf("Expected %s=%s, got %s=%s", k, v, k, actual[k])
		}
	}
}
