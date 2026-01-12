package autofix

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"liberation-guardian/pkg/types"
)

// CommandHandler handles command execution (run_command, restart_service)
type CommandHandler struct {
	logger         *logrus.Logger
	validator      *SafetyValidator
	defaultTimeout time.Duration
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(logger *logrus.Logger, validator *SafetyValidator) *CommandHandler {
	return &CommandHandler{
		logger:         logger,
		validator:      validator,
		defaultTimeout: 5 * time.Minute, // Default 5 minute timeout
	}
}

// CanHandle returns true if this handler can handle the given action
func (h *CommandHandler) CanHandle(action string) bool {
	return action == ActionRunCommand ||
		action == ActionRestartService
}

// Validate validates the fix step
func (h *CommandHandler) Validate(ctx context.Context, step types.FixStep) error {
	command := step.Parameters["command"]
	if command == "" {
		return fmt.Errorf("command parameter is required")
	}

	// Validate command safety
	return h.validator.ValidateCommand(command)
}

// Execute executes the command
func (h *CommandHandler) Execute(ctx context.Context, step types.FixStep, execCtx *ExecutionContext) (*StepResult, error) {
	command := step.Parameters["command"]
	args := step.Parameters["args"]
	workdir := step.Parameters["workdir"]
	envVars := step.Parameters["env"]

	if workdir == "" && execCtx.WorkingDirectory != "" {
		workdir = execCtx.WorkingDirectory
	}

	h.logger.Infof("Executing command: %s %s", command, args)

	// Build full command
	fullCommand := command
	if args != "" {
		fullCommand = fmt.Sprintf("%s %s", command, args)
	}

	// Get timeout
	timeout := h.getTimeout(step)

	// Create context with timeout
	execContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(execContext, "sh", "-c", fullCommand)

	// Set working directory
	if workdir != "" {
		cmd.Dir = workdir
	}

	// Set environment variables
	cmd.Env = os.Environ() // Start with current environment
	if envVars != "" {
		for _, env := range strings.Split(envVars, ",") {
			env = strings.TrimSpace(env)
			if env != "" {
				cmd.Env = append(cmd.Env, env)
			}
		}
	}

	// Execute with combined output
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	executionTime := time.Since(startTime)

	if err != nil {
		h.logger.Errorf("Command failed after %v: %v, output: %s", executionTime, err, string(output))
		return &StepResult{
			Success: false,
			Output:  string(output),
			Error:   err,
		}, fmt.Errorf("command failed: %w", err)
	}

	h.logger.Infof("Command completed successfully in %v", executionTime)

	return &StepResult{
		Success: true,
		Output:  string(output),
	}, nil
}

// Rollback for commands is typically not possible, but we log it
func (h *CommandHandler) Rollback(ctx context.Context, step types.FixStep, execCtx *ExecutionContext) error {
	h.logger.Warnf("Command rollback requested for: %s (commands cannot be automatically rolled back)", step.Parameters["command"])

	// Check if a rollback command was specified
	rollbackCommand := step.Parameters["rollback_command"]
	if rollbackCommand != "" {
		h.logger.Infof("Executing rollback command: %s", rollbackCommand)

		cmd := exec.CommandContext(ctx, "sh", "-c", rollbackCommand)
		if execCtx.WorkingDirectory != "" {
			cmd.Dir = execCtx.WorkingDirectory
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			h.logger.Errorf("Rollback command failed: %v, output: %s", err, string(output))
			return fmt.Errorf("rollback command failed: %w", err)
		}

		h.logger.Infof("Rollback command completed successfully")
		return nil
	}

	// No rollback possible for most commands
	return nil
}

// getTimeout gets the timeout for command execution
func (h *CommandHandler) getTimeout(step types.FixStep) time.Duration {
	timeoutStr := step.Parameters["timeout"]
	if timeoutStr == "" {
		return h.defaultTimeout
	}

	// Parse timeout (e.g., "5m", "30s", "1h")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		h.logger.Warnf("Invalid timeout format %s, using default", timeoutStr)
		return h.defaultTimeout
	}

	// Cap at 10 minutes for safety
	maxTimeout := 10 * time.Minute
	if timeout > maxTimeout {
		h.logger.Warnf("Timeout %v exceeds maximum %v, capping", timeout, maxTimeout)
		return maxTimeout
	}

	return timeout
}
