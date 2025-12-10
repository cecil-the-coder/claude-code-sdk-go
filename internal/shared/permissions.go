// Package shared provides permission-related types for tool permission callbacks.
package shared

import (
	"context"
	"encoding/json"
)

// PermissionBehavior represents the behavior for a permission rule.
type PermissionBehavior string

const (
	// PermissionBehaviorAllow allows the operation.
	PermissionBehaviorAllow PermissionBehavior = "allow"
	// PermissionBehaviorDeny denies the operation.
	PermissionBehaviorDeny PermissionBehavior = "deny"
	// PermissionBehaviorAsk prompts the user for permission.
	PermissionBehaviorAsk PermissionBehavior = "ask"
)

// PermissionUpdateDestination represents where permission updates should be saved.
type PermissionUpdateDestination string

const (
	// PermissionUpdateDestinationUserSettings saves to user settings.
	PermissionUpdateDestinationUserSettings PermissionUpdateDestination = "userSettings"
	// PermissionUpdateDestinationProjectSettings saves to project settings.
	PermissionUpdateDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	// PermissionUpdateDestinationLocalSettings saves to local settings.
	PermissionUpdateDestinationLocalSettings PermissionUpdateDestination = "localSettings"
	// PermissionUpdateDestinationSession saves to session only.
	PermissionUpdateDestinationSession PermissionUpdateDestination = "session"
)

// PermissionUpdateType represents the type of permission update.
type PermissionUpdateType string

const (
	// PermissionUpdateTypeAddRules adds new permission rules.
	PermissionUpdateTypeAddRules PermissionUpdateType = "addRules"
	// PermissionUpdateTypeReplaceRules replaces existing permission rules.
	PermissionUpdateTypeReplaceRules PermissionUpdateType = "replaceRules"
	// PermissionUpdateTypeRemoveRules removes permission rules.
	PermissionUpdateTypeRemoveRules PermissionUpdateType = "removeRules"
	// PermissionUpdateTypeSetMode sets the permission mode.
	PermissionUpdateTypeSetMode PermissionUpdateType = "setMode"
	// PermissionUpdateTypeAddDirectories adds allowed directories.
	PermissionUpdateTypeAddDirectories PermissionUpdateType = "addDirectories"
	// PermissionUpdateTypeRemoveDirectories removes allowed directories.
	PermissionUpdateTypeRemoveDirectories PermissionUpdateType = "removeDirectories"
)

// PermissionRuleValue represents a permission rule.
type PermissionRuleValue struct {
	ToolName    string  `json:"tool_name"`
	RuleContent *string `json:"rule_content,omitempty"`
}

// PermissionUpdate represents a permission update configuration.
type PermissionUpdate struct {
	Type        PermissionUpdateType         `json:"type"`
	Rules       []PermissionRuleValue        `json:"rules,omitempty"`
	Behavior    *PermissionBehavior          `json:"behavior,omitempty"`
	Mode        *PermissionMode              `json:"mode,omitempty"`
	Directories []string                     `json:"directories,omitempty"`
	Destination *PermissionUpdateDestination `json:"destination,omitempty"`
}

// ToolPermissionContext provides context information for tool permission callbacks.
type ToolPermissionContext struct {
	// Signal is reserved for future abort signal support.
	Signal interface{} `json:"signal,omitempty"`
	// Suggestions contains permission suggestions from the CLI.
	Suggestions []PermissionUpdate `json:"suggestions,omitempty"`
}

// PermissionResult is the interface for permission callback results.
type PermissionResult interface {
	Behavior() string
}

// PermissionResultAllow represents an allow permission result.
type PermissionResultAllow struct {
	// UpdatedInput contains optional modified input for the tool.
	UpdatedInput map[string]interface{} `json:"updated_input,omitempty"`
	// UpdatedPermissions contains optional new permission rules.
	UpdatedPermissions []PermissionUpdate `json:"updated_permissions,omitempty"`
}

// Behavior returns "allow" for PermissionResultAllow.
func (r *PermissionResultAllow) Behavior() string {
	return "allow"
}

// MarshalJSON implements custom JSON marshaling for PermissionResultAllow.
func (r *PermissionResultAllow) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"behavior": "allow",
	}
	if r.UpdatedInput != nil {
		data["updated_input"] = r.UpdatedInput
	}
	if r.UpdatedPermissions != nil {
		data["updated_permissions"] = r.UpdatedPermissions
	}
	return json.Marshal(data)
}

// PermissionResultDeny represents a deny permission result.
type PermissionResultDeny struct {
	// Message provides the reason for denial.
	Message string `json:"message,omitempty"`
	// Interrupt indicates whether to stop the conversation.
	Interrupt bool `json:"interrupt,omitempty"`
}

// Behavior returns "deny" for PermissionResultDeny.
func (r *PermissionResultDeny) Behavior() string {
	return "deny"
}

// MarshalJSON implements custom JSON marshaling for PermissionResultDeny.
func (r *PermissionResultDeny) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"behavior": "deny",
	}
	if r.Message != "" {
		data["message"] = r.Message
	}
	if r.Interrupt {
		data["interrupt"] = r.Interrupt
	}
	return json.Marshal(data)
}

// CanUseToolFunc is the callback function type for tool permission decisions.
// It is invoked when Claude wants to use a tool, allowing the SDK user to:
// - Allow the tool use (optionally modifying inputs or updating permissions)
// - Deny the tool use (optionally interrupting the conversation)
//
// The callback receives:
// - ctx: Context for cancellation
// - toolName: Name of the tool Claude wants to use
// - input: Input parameters for the tool
// - context: Additional context including CLI suggestions
//
// Returns:
// - PermissionResult: Either PermissionResultAllow or PermissionResultDeny
// - error: Any error that occurred during the callback
type CanUseToolFunc func(ctx context.Context, toolName string, input map[string]interface{}, context ToolPermissionContext) (PermissionResult, error)
