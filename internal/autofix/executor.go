package autofix

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/codebase"
	"liberation-guardian/internal/config"
	"liberation-guardian/internal/events"
	"liberation-guardian/pkg/types"
)

// AutoFixExecutor orchestrates the execution of auto-fix plans
type AutoFixExecutor struct {
	config           *config.Config
	logger           *logrus.Logger
	handlerRegistry  *HandlerRegistry
	validator        *SafetyValidator
	knowledgeBase    *events.RedisKnowledgeBase
	workspaceManager *WorkspaceManager
}

// NewAutoFixExecutor creates a new auto-fix executor
func NewAutoFixExecutor(cfg *config.Config, logger *logrus.Logger, knowledgeBase *events.RedisKnowledgeBase) *AutoFixExecutor {
	// Create codebase analyzer config for validation
	codebaseConfig := &codebase.AnalyzerConfig{
		AllowedPaths:      []string{"src/", "internal/", "pkg/", "config/", "lib/", "app/"},
		BlockedPaths:      []string{".env", ".secret", ".git/", "node_modules/", "vendor/", "credentials"},
		MaxFileSize:       100 * 1024, // 100KB
		MaxFiles:          20,
		IncludeGitHistory: false,
		TrustLevel:        "cautious",
	}

	// Create validator
	validator := NewSafetyValidator(cfg, logger, codebaseConfig)

	// Create workspace manager
	workspaceBaseDir := "/tmp/liberation-guardian-workspaces"
	workspaceManager := NewWorkspaceManager(logger, workspaceBaseDir)

	// Create handler registry
	handlerRegistry := NewHandlerRegistry()

	return &AutoFixExecutor{
		config:           cfg,
		logger:           logger,
		handlerRegistry:  handlerRegistry,
		validator:        validator,
		knowledgeBase:    knowledgeBase,
		workspaceManager: workspaceManager,
	}
}

// RegisterHandlers registers all action handlers
func (e *AutoFixExecutor) RegisterHandlers(
	fileHandler ActionHandler,
	configHandler ActionHandler,
	commandHandler ActionHandler,
	prHandler ActionHandler,
) {
	e.handlerRegistry.RegisterDefaultHandlers(fileHandler, configHandler, commandHandler, prHandler)
}

// ExecuteFixPlan executes a complete auto-fix plan
func (e *AutoFixExecutor) ExecuteFixPlan(ctx context.Context, event *types.LiberationGuardianEvent, plan *types.AutoFixPlan) (*ExecutionResult, error) {
	startTime := time.Now()
	e.logger.Infof("Executing fix plan for event %s (type: %s)", event.ID, plan.Type)

	// 1. PRE-EXECUTION SAFETY CHECKS
	if err := e.validator.ValidateFixPlan(plan, event); err != nil {
		e.logger.Errorf("Fix plan validation failed: %v", err)
		return &ExecutionResult{
			Success:    false,
			TotalSteps: len(plan.Steps),
			Error:      fmt.Errorf("validation failed: %w", err),
			Duration:   time.Since(startTime),
		}, err
	}

	// 2. CREATE EXECUTION CONTEXT
	execCtx := e.createExecutionContext(event, plan)

	// 3. SETUP ISOLATED WORKSPACE (for file operations)
	var workspace *Workspace
	if e.requiresWorkspace(plan.Type) {
		var err error
		workspace, err = e.workspaceManager.CreateWorkspace(ctx, execCtx)
		if err != nil {
			return &ExecutionResult{
				Success:    false,
				TotalSteps: len(plan.Steps),
				Error:      fmt.Errorf("workspace creation failed: %w", err),
				Duration:   time.Since(startTime),
			}, err
		}
		defer e.workspaceManager.Cleanup(workspace)
		execCtx.WorkingDirectory = workspace.Path
	}

	// 4. EXECUTE STEPS SEQUENTIALLY
	result := &ExecutionResult{
		TotalSteps:  len(plan.Steps),
		StepResults: make([]StepResult, 0),
	}

	for i, step := range plan.Steps {
		stepResult, err := e.executeStep(ctx, step, i, execCtx)
		result.StepResults = append(result.StepResults, *stepResult)
		execCtx.CompletedSteps = append(execCtx.CompletedSteps, *stepResult)

		if err != nil {
			e.logger.Errorf("Step %d failed: %v", i, err)
			result.Error = err

			// Check OnFailure policy
			if step.OnFailure == "rollback" {
				e.logger.Warnf("Initiating rollback due to step %d failure", i)
				rollbackErr := e.rollbackAllSteps(ctx, execCtx)
				result.RollbackRequired = true
				result.RollbackSuccess = (rollbackErr == nil)
				break
			} else if step.OnFailure == "continue" {
				e.logger.Infof("Step %d failed but continuing per policy", i)
				continue
			} else {
				// Default: stop execution
				e.logger.Warnf("Stopping execution due to step %d failure", i)
				break
			}
		}

		result.CompletedSteps++
	}

	// 5. POST-EXECUTION VALIDATION
	if result.CompletedSteps == result.TotalSteps && result.Error == nil {
		validated, validationMsg := e.validator.ValidateFixSuccess(ctx, plan, execCtx)
		result.Success = validated
		if !validated {
			e.logger.Warnf("Post-execution validation failed: %s", validationMsg)
			result.Error = fmt.Errorf("validation failed: %s", validationMsg)

			// Auto-rollback if configured
			if err := e.rollbackAllSteps(ctx, execCtx); err != nil {
				e.logger.Errorf("Rollback after validation failure failed: %v", err)
			} else {
				result.RollbackRequired = true
				result.RollbackSuccess = true
			}
		}
	}

	result.Duration = time.Since(startTime)

	// 6. RECORD TO KNOWLEDGE BASE
	if e.knowledgeBase != nil {
		if err := e.knowledgeBase.RecordResolution(ctx, event.ID, plan, result.Success); err != nil {
			e.logger.Warnf("Failed to record resolution to knowledge base: %v", err)
		}
	}

	e.logger.Infof("Fix execution completed for event %s: success=%v, steps=%d/%d, duration=%v",
		event.ID, result.Success, result.CompletedSteps, result.TotalSteps, result.Duration)

	return result, result.Error
}

// executeStep executes a single fix step
func (e *AutoFixExecutor) executeStep(ctx context.Context, step types.FixStep, index int, execCtx *ExecutionContext) (*StepResult, error) {
	e.logger.Infof("Executing step %d: %s on %s", index, step.Action, step.Target)

	stepResult := &StepResult{
		StepIndex: index,
		Action:    step.Action,
	}
	startTime := time.Now()

	// 1. Find appropriate handler
	handler := e.handlerRegistry.GetHandler(step.Action)
	if handler == nil {
		stepResult.Error = fmt.Errorf("no handler for action: %s", step.Action)
		return stepResult, stepResult.Error
	}

	// 2. Validate step
	if err := handler.Validate(ctx, step); err != nil {
		stepResult.Error = err
		return stepResult, fmt.Errorf("step validation failed: %w", err)
	}

	// 3. Execute
	executionResult, err := handler.Execute(ctx, step, execCtx)
	stepResult.Success = (err == nil)
	if executionResult != nil {
		stepResult.Output = executionResult.Output
	}
	stepResult.Error = err
	stepResult.ExecutionTime = time.Since(startTime)

	// 4. Run validation command if specified
	if step.Validation != "" && err == nil {
		validated, validationOutput := e.runValidation(ctx, step.Validation, execCtx)
		stepResult.Validated = validated
		stepResult.ValidationOutput = validationOutput

		if !validated {
			err = fmt.Errorf("validation failed: %s", validationOutput)
			stepResult.Success = false
			stepResult.Error = err
			return stepResult, err
		}
	}

	if err == nil {
		e.logger.Infof("Step %d completed successfully in %v", index, stepResult.ExecutionTime)
	}

	return stepResult, err
}

// runValidation runs a validation command for a step
func (e *AutoFixExecutor) runValidation(ctx context.Context, validationCmd string, execCtx *ExecutionContext) (bool, string) {
	// Delegate to validator
	return e.validator.runValidationCommand(ctx, validationCmd, execCtx)
}

// rollbackAllSteps rolls back all completed steps
func (e *AutoFixExecutor) rollbackAllSteps(ctx context.Context, execCtx *ExecutionContext) error {
	e.logger.Warnf("Rolling back %d completed steps", len(execCtx.CompletedSteps))

	// Rollback in reverse order
	for i := len(execCtx.CompletedSteps) - 1; i >= 0; i-- {
		stepResult := execCtx.CompletedSteps[i]

		// Skip failed steps (nothing to rollback)
		if !stepResult.Success {
			continue
		}

		handler := e.handlerRegistry.GetHandler(stepResult.Action)
		if handler == nil {
			e.logger.Warnf("No handler found for rollback of step %d (action: %s)", i, stepResult.Action)
			continue
		}

		// Create a FixStep from the result for rollback
		step := types.FixStep{
			Action: stepResult.Action,
		}

		if err := handler.Rollback(ctx, step, execCtx); err != nil {
			e.logger.Errorf("Rollback failed for step %d: %v", i, err)
			// Continue rolling back other steps
		} else {
			e.logger.Infof("Successfully rolled back step %d", i)
		}
	}

	return nil
}

// createExecutionContext creates an execution context for the fix plan
func (e *AutoFixExecutor) createExecutionContext(event *types.LiberationGuardianEvent, plan *types.AutoFixPlan) *ExecutionContext {
	return &ExecutionContext{
		EventID:        event.ID,
		FixPlanType:    plan.Type,
		StartedAt:      time.Now(),
		CompletedSteps: make([]StepResult, 0),
		RollbackData:   make([]RollbackData, 0),
		Metadata:       make(map[string]interface{}),
	}
}

// requiresWorkspace determines if the fix type requires a workspace
func (e *AutoFixExecutor) requiresWorkspace(fixType types.AutoFixType) bool {
	workspaceTypes := []types.AutoFixType{
		types.FixTypeCodeChange,
		types.FixTypeDependencyUpdate,
	}

	for _, t := range workspaceTypes {
		if fixType == t {
			return true
		}
	}
	return false
}
