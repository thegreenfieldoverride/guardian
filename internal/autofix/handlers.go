package autofix

import (
	"context"
	"fmt"
	"time"

	"liberation-guardian/pkg/types"
)

// ActionHandler interface - all handlers implement this
type ActionHandler interface {
	// Validate checks if the fix step can be executed safely
	Validate(ctx context.Context, step types.FixStep) error

	// Execute performs the actual fix action
	Execute(ctx context.Context, step types.FixStep, execCtx *ExecutionContext) (*StepResult, error)

	// Rollback reverses the action if execution fails
	Rollback(ctx context.Context, step types.FixStep, execCtx *ExecutionContext) error

	// CanHandle returns true if this handler supports the action
	CanHandle(action string) bool
}

// Common action types (from AI-generated FixSteps)
const (
	ActionUpdateFile     = "update_file"
	ActionCreateFile     = "create_file"
	ActionDeleteFile     = "delete_file"
	ActionRunCommand     = "run_command"
	ActionRestartService = "restart_service"
	ActionCreatePR       = "create_pr"
	ActionUpdateConfig   = "update_config"
	ActionSetEnvVar      = "set_env_var"
	ActionRunMigration   = "run_migration"
	ActionScaleService   = "scale_service"
)

// ExecutionContext tracks execution state across steps
type ExecutionContext struct {
	EventID          string
	FixPlanType      types.AutoFixType
	StartedAt        time.Time
	CompletedSteps   []StepResult
	RollbackData     []RollbackData
	WorkingDirectory string
	GitBranch        string // For code changes
	PRNumber         int    // If PR created
	Metadata         map[string]interface{}
}

// StepResult captures result of a single fix step
type StepResult struct {
	StepIndex        int
	Action           string
	Success          bool
	Output           string
	Error            error
	ExecutionTime    time.Duration
	Validated        bool
	ValidationOutput string
}

// RollbackData stores info needed to rollback a step
type RollbackData struct {
	StepIndex    int
	Action       string
	OriginalData interface{} // File content, env value, etc.
	Timestamp    time.Time
}

// ExecutionResult is the final result of fix execution
type ExecutionResult struct {
	Success         bool
	CompletedSteps  int
	TotalSteps      int
	StepResults     []StepResult
	RollbackRequired bool
	RollbackSuccess bool
	Duration        time.Duration
	Error           error
}

// HandlerRegistry manages action handlers
type HandlerRegistry struct {
	handlers map[string]ActionHandler
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]ActionHandler),
	}
}

// Register adds a handler to the registry
func (r *HandlerRegistry) Register(handler ActionHandler) {
	// Each handler declares which actions it handles
	// For now, we'll register by handler type and let the handler decide
	r.handlers[fmt.Sprintf("%T", handler)] = handler
}

// GetHandler finds the appropriate handler for an action
func (r *HandlerRegistry) GetHandler(action string) ActionHandler {
	for _, handler := range r.handlers {
		if handler.CanHandle(action) {
			return handler
		}
	}
	return nil
}

// RegisterDefaultHandlers registers all default handlers
func (r *HandlerRegistry) RegisterDefaultHandlers(
	fileHandler ActionHandler,
	configHandler ActionHandler,
	commandHandler ActionHandler,
	prHandler ActionHandler,
) {
	if fileHandler != nil {
		r.Register(fileHandler)
	}
	if configHandler != nil {
		r.Register(configHandler)
	}
	if commandHandler != nil {
		r.Register(commandHandler)
	}
	if prHandler != nil {
		r.Register(prHandler)
	}
}
