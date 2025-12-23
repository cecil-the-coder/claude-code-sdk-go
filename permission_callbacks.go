package claudecode

import (
	"context"
	"fmt"
	"time"
)

// PermissionBehavior represents permission behavior
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionUpdateDestination represents where permission updates are stored
type PermissionUpdateDestination string

const (
	PermissionUpdateDestinationUserSettings    PermissionUpdateDestination = "userSettings"
	PermissionUpdateDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	PermissionUpdateDestinationLocalSettings   PermissionUpdateDestination = "localSettings"
	PermissionUpdateDestinationSessionSettings PermissionUpdateDestination = "sessionSettings"
)

// PermissionRuleValue represents a permission rule
type PermissionRuleValue struct {
	ToolName    string `json:"tool_name"`
	RuleContent string `json:"rule_content,omitempty"`
}

// PermissionUpdate represents a permission update request
type PermissionUpdate struct {
	Type        PermissionUpdateDestination `json:"type"`
	Rules       []PermissionRuleValue     `json:"rules,omitempty"`
	Behavior    *PermissionBehavior        `json:"behavior,omitempty"`
	Mode        *PermissionMode           `json:"mode,omitempty"`
	Directories []string                   `json:"directories,omitempty"`
	Destination *PermissionUpdateDestination `json:"destination,omitempty"`
}

// ToolPermissionContext provides context information for tool permission callbacks
type ToolPermissionContext struct {
	Signal      any              `json:"signal,omitempty"`
	Suggestions []PermissionUpdate `json:"suggestions,omitempty"`
}

// PermissionResult represents the result of a tool permission check
type PermissionResult interface {
	Behavior() PermissionBehavior
	UpdatedInput() map[string]any
	UpdatedPermissions() []PermissionUpdate
	Message() string
	ShouldInterrupt() bool
}

// PermissionResultAllow implements PermissionResult for allowed operations
type PermissionResultAllow struct {
	updatedInput      map[string]any    `json:"-"`
	updatedPermissions []PermissionUpdate `json:"-"`
}

// Behavior returns "allow"
func (p *PermissionResultAllow) Behavior() PermissionBehavior {
	return PermissionBehaviorAllow
}

// UpdatedInput returns any modified input data
func (p *PermissionResultAllow) UpdatedInput() map[string]any {
	return p.updatedInput
}

// UpdatedPermissions returns any permission updates
func (p *PermissionResultAllow) UpdatedPermissions() []PermissionUpdate {
	return p.updatedPermissions
}

// Message returns an empty message for allow results
func (p *PermissionResultAllow) Message() string {
	return ""
}

// ShouldInterrupt returns false for allow results
func (p *PermissionResultAllow) ShouldInterrupt() bool {
	return false
}

// WithInput returns a new PermissionResultAllow with updated input
func (p *PermissionResultAllow) WithInput(input map[string]any) *PermissionResultAllow {
	p.updatedInput = input
	return p
}

// WithPermissions returns a new PermissionResultAllow with permission updates
func (p *PermissionResultAllow) WithPermissions(updates []PermissionUpdate) *PermissionResultAllow {
	p.updatedPermissions = updates
	return p
}

// NewPermissionResultAllow creates a new allow result
func NewPermissionResultAllow() *PermissionResultAllow {
	return &PermissionResultAllow{}
}

// PermissionResultDeny implements PermissionResult for denied operations
type PermissionResultDeny struct {
	message   string  `json:"-"`
	interrupt bool    `json:"-"`
}

// Behavior returns "deny"
func (p *PermissionResultDeny) Behavior() PermissionBehavior {
	return PermissionBehaviorDeny
}

// UpdatedInput returns nil for deny results
func (p *PermissionResultDeny) UpdatedInput() map[string]any {
	return nil
}

// UpdatedPermissions returns nil for deny results
func (p *PermissionResultDeny) UpdatedPermissions() []PermissionUpdate {
	return nil
}

// Message returns the denial message
func (p *PermissionResultDeny) Message() string {
	return p.message
}

// ShouldInterrupt returns the interrupt flag
func (p *PermissionResultDeny) ShouldInterrupt() bool {
	return p.interrupt
}

// WithInterrupt returns a new PermissionResultDeny with interrupt flag
func (p *PermissionResultDeny) WithInterrupt() *PermissionResultDeny {
	p.interrupt = true
	return p
}

// NewPermissionResultDeny creates a new deny result
func NewPermissionResultDeny(message string) *PermissionResultDeny {
	return &PermissionResultDeny{
		message: message,
	}
}

// CanUseToolFunc is the callback function type for tool permission checks
type CanUseToolFunc func(
	ctx context.Context,
	toolName string,
	input map[string]any,
	permContext ToolPermissionContext,
) (PermissionResult, error)

// PermissionManager manages tool permission callbacks
type PermissionManager interface {
	// SetPermissionCallback sets the global permission callback
	SetPermissionCallback(callback CanUseToolFunc)

	// CheckPermission calls the current permission callback
	CheckPermission(ctx context.Context, toolName string, input map[string]any, permContext ToolPermissionContext) (PermissionResult, error)

	// HasCallback returns true if a permission callback is set
	HasCallback() bool
}

// permissionManager implements PermissionManager
type permissionManager struct {
	callback CanUseToolFunc
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager() PermissionManager {
	return &permissionManager{}
}

// SetPermissionCallback sets the permission callback
func (pm *permissionManager) SetPermissionCallback(callback CanUseToolFunc) {
	pm.callback = callback
}

// CheckPermission executes the permission callback if set
func (pm *permissionManager) CheckPermission(ctx context.Context, toolName string, input map[string]any, permContext ToolPermissionContext) (PermissionResult, error) {
	if pm.callback == nil {
		// Default: allow all operations when no callback is set
		return NewPermissionResultAllow(), nil
	}

	// Execute callback with timeout to prevent blocking
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resultChan := make(chan PermissionResult, 1)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Send panic as error
				select {
				case errChan <- fmt.Errorf("callback panic: %v", r):
				default:
				}
			}
		}()

		result, err := pm.callback(timeoutCtx, toolName, input, permContext)
		if err != nil {
			select {
			case errChan <- err:
			default:
			}
		} else {
			select {
			case resultChan <- result:
			default:
			}
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return NewPermissionResultDeny("Callback failed"), fmt.Errorf("permission callback failed: %w", err)
	case <-timeoutCtx.Done():
		return NewPermissionResultDeny("Callback timeout"), nil
	}
}

// HasCallback returns true if a callback is set
func (pm *permissionManager) HasCallback() bool {
	return pm.callback != nil
}