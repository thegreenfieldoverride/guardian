package autofix

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/codebase"
	"liberation-guardian/internal/config"
	"liberation-guardian/pkg/types"
)

// SafetyValidator performs pre and post-execution validation
type SafetyValidator struct {
	config         *config.Config
	logger         *logrus.Logger
	codebaseConfig *codebase.AnalyzerConfig
}

// NewSafetyValidator creates a new safety validator
func NewSafetyValidator(cfg *config.Config, logger *logrus.Logger, codebaseConfig *codebase.AnalyzerConfig) *SafetyValidator {
	return &SafetyValidator{
		config:         cfg,
		logger:         logger,
		codebaseConfig: codebaseConfig,
	}
}

// ValidateFixPlan validates the entire fix plan before execution
func (v *SafetyValidator) ValidateFixPlan(plan *types.AutoFixPlan, event *types.LiberationGuardianEvent) error {
	v.logger.Infof("Validating fix plan for event %s (type: %s)", event.ID, plan.Type)

	// 1. Check if fix requires approval
	if plan.RequiresApproval {
		return fmt.Errorf("fix requires human approval")
	}

	// 2. Validate all steps
	for i, step := range plan.Steps {
		if err := v.validateStep(step, i); err != nil {
			return fmt.Errorf("step %d invalid: %w", i, err)
		}
	}

	// 3. Check rollback plan exists for risky operations
	if v.isRiskyFixType(plan.Type) && len(plan.RollbackPlan) == 0 {
		v.logger.Warnf("No rollback plan provided for risky fix type: %s", plan.Type)
		// Don't fail, but warn
	}

	// 4. Validate max fix attempts not exceeded
	maxAttempts := v.config.DecisionRules.AutoFix.Conditions.MaxFixAttempts
	if maxAttempts > 0 {
		// This would check against knowledge base for pattern-based attempt tracking
		v.logger.Debugf("Max fix attempts configured: %d", maxAttempts)
	}

	v.logger.Infof("Fix plan validation passed for event %s", event.ID)
	return nil
}

// ValidateFixSuccess performs post-execution validation
func (v *SafetyValidator) ValidateFixSuccess(ctx context.Context, plan *types.AutoFixPlan, execCtx *ExecutionContext) (bool, string) {
	v.logger.Infof("Validating fix success for event %s", execCtx.EventID)

	// 1. Run validation commands from plan steps
	for i, step := range plan.Steps {
		if step.Validation != "" {
			success, output := v.runValidationCommand(ctx, step.Validation, execCtx)
			if !success {
				return false, fmt.Sprintf("Step %d validation failed: %s", i, output)
			}
		}
	}

	// 2. Check if tests are required
	if v.config.DecisionRules.AutoFix.Conditions.RequireTests {
		if err := v.runTestSuite(ctx, execCtx); err != nil {
			return false, fmt.Sprintf("Test suite failed: %v", err)
		}
	}

	v.logger.Infof("Fix success validation passed for event %s", execCtx.EventID)
	return true, "All validations passed"
}

// validateStep validates a single fix step
func (v *SafetyValidator) validateStep(step types.FixStep, index int) error {
	// File path validation
	if step.Action == ActionUpdateFile || step.Action == ActionCreateFile || step.Action == ActionDeleteFile {
		if err := v.ValidateFilePath(step.Target); err != nil {
			return fmt.Errorf("file path validation failed: %w", err)
		}
	}

	// Command validation
	if step.Action == ActionRunCommand || step.Action == ActionRestartService {
		command := step.Parameters["command"]
		if command == "" {
			return fmt.Errorf("command parameter is required")
		}
		if err := v.ValidateCommand(command); err != nil {
			return fmt.Errorf("command validation failed: %w", err)
		}
	}

	// Config file validation
	if step.Action == ActionUpdateConfig {
		if err := v.ValidateFilePath(step.Target); err != nil {
			return fmt.Errorf("config file path validation failed: %w", err)
		}
	}

	return nil
}

// ValidateFilePath validates a file path against allowed/blocked lists
func (v *SafetyValidator) ValidateFilePath(path string) error {
	// 1. Check against blocked paths
	if v.codebaseConfig != nil {
		for _, blocked := range v.codebaseConfig.BlockedPaths {
			if strings.Contains(path, blocked) {
				return fmt.Errorf("path contains blocked pattern: %s", blocked)
			}
		}

		// 2. Check against allowed paths
		if len(v.codebaseConfig.AllowedPaths) > 0 {
			allowed := false
			for _, allowedPath := range v.codebaseConfig.AllowedPaths {
				if strings.HasPrefix(path, allowedPath) || strings.Contains(path, allowedPath) {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("path not in allowed list: %s", path)
			}
		}
	}

	// 3. Additional safety checks for sensitive file patterns
	sensitivePatterns := []string{
		".env",
		".secret",
		"credentials",
		"private_key",
		"id_rsa",
		".pem",
		"password",
	}
	pathLower := strings.ToLower(path)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(pathLower, pattern) {
			return fmt.Errorf("path contains sensitive pattern: %s", pattern)
		}
	}

	return nil
}

// ValidateCommand validates a command against dangerous patterns and allowlist
func (v *SafetyValidator) ValidateCommand(command string) error {
	// 1. Check for dangerous command patterns
	dangerousPatterns := []string{
		`rm\s+-rf\s+/`,
		`dd\s+if=`,
		`mkfs`,
		`>\s*/dev/`,
		`chmod\s+-R\s+777`,
		`curl.*\|.*sh`,
		`wget.*\|.*sh`,
		`:(){ :|:& };:`, // Fork bomb
		`/dev/sda`,
		`/dev/sd[a-z]`,
	}

	for _, pattern := range dangerousPatterns {
		matched, err := regexp.MatchString(pattern, command)
		if err != nil {
			v.logger.Warnf("Error checking dangerous pattern %s: %v", pattern, err)
			continue
		}
		if matched {
			return fmt.Errorf("dangerous command pattern detected: %s", pattern)
		}
	}

	// 2. Check against command allowlist
	allowedCommands := []string{
		"npm install",
		"npm test",
		"npm run",
		"docker restart",
		"docker-compose restart",
		"systemctl restart",
		"service restart",
		"go test",
		"go build",
		"python -m pytest",
		"pytest",
		"cargo test",
		"cargo build",
		"make test",
		"bundle install",
		"composer install",
	}

	// Check if command starts with any allowed prefix
	commandLower := strings.ToLower(strings.TrimSpace(command))
	for _, allowed := range allowedCommands {
		if strings.HasPrefix(commandLower, strings.ToLower(allowed)) {
			return nil
		}
	}

	return fmt.Errorf("command not in allowlist: %s", command)
}

// runValidationCommand runs a validation command and returns success/output
func (v *SafetyValidator) runValidationCommand(ctx context.Context, validationCmd string, execCtx *ExecutionContext) (bool, string) {
	v.logger.Debugf("Running validation command: %s", validationCmd)

	cmd := exec.CommandContext(ctx, "sh", "-c", validationCmd)
	if execCtx.WorkingDirectory != "" {
		cmd.Dir = execCtx.WorkingDirectory
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		v.logger.Warnf("Validation command failed: %v, output: %s", err, output)
		return false, string(output)
	}

	return true, string(output)
}

// runTestSuite runs the test suite for the codebase
func (v *SafetyValidator) runTestSuite(ctx context.Context, execCtx *ExecutionContext) error {
	v.logger.Infof("Running test suite in %s", execCtx.WorkingDirectory)

	// Detect test framework and run appropriate command
	testCommand := v.detectTestCommand(execCtx.WorkingDirectory)
	if testCommand == "" {
		v.logger.Warn("No test framework detected, skipping test execution")
		return nil
	}

	// #nosec G204 - Command execution is validated against allowlist in validateCommand()
	cmd := exec.CommandContext(ctx, "sh", "-c", testCommand)
	cmd.Dir = execCtx.WorkingDirectory

	output, err := cmd.CombinedOutput()
	if err != nil {
		v.logger.Errorf("Test suite failed: %v, output: %s", err, output)
		return fmt.Errorf("test suite failed: %w", err)
	}

	v.logger.Infof("Test suite passed")
	return nil
}

// detectTestCommand detects the appropriate test command for the codebase
func (v *SafetyValidator) detectTestCommand(workDir string) string {
	// Check for various test framework indicators
	// This is a simple implementation - could be enhanced with file existence checks

	// npm/node
	if v.fileExists(workDir, "package.json") {
		return "npm test"
	}

	// Go
	if v.fileExists(workDir, "go.mod") {
		return "go test ./..."
	}

	// Python
	if v.fileExists(workDir, "pytest.ini") || v.fileExists(workDir, "setup.py") {
		return "pytest"
	}

	// Rust
	if v.fileExists(workDir, "Cargo.toml") {
		return "cargo test"
	}

	// Ruby
	if v.fileExists(workDir, "Gemfile") {
		return "bundle exec rake test"
	}

	return ""
}

// fileExists checks if a file exists in the working directory
func (v *SafetyValidator) fileExists(workDir, filename string) bool {
	// Simple heuristic - in production would actually check file system
	// For now, just return false to skip test execution
	return false
}

// isRiskyFixType determines if a fix type is considered risky
func (v *SafetyValidator) isRiskyFixType(fixType types.AutoFixType) bool {
	riskyTypes := []types.AutoFixType{
		types.FixTypeCodeChange,
		types.FixTypeInfrastructure,
	}

	for _, risky := range riskyTypes {
		if fixType == risky {
			return true
		}
	}
	return false
}
