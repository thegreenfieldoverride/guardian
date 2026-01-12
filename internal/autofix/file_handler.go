package autofix

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"liberation-guardian/pkg/types"
)

// FileHandler handles file operations (update, create, delete)
type FileHandler struct {
	logger    *logrus.Logger
	validator *SafetyValidator
}

// NewFileHandler creates a new file handler
func NewFileHandler(logger *logrus.Logger, validator *SafetyValidator) *FileHandler {
	return &FileHandler{
		logger:    logger,
		validator: validator,
	}
}

// CanHandle returns true if this handler can handle the given action
func (h *FileHandler) CanHandle(action string) bool {
	return action == ActionUpdateFile ||
		action == ActionCreateFile ||
		action == ActionDeleteFile
}

// Validate validates the fix step
func (h *FileHandler) Validate(ctx context.Context, step types.FixStep) error {
	// Validate file path
	if step.Target == "" {
		return fmt.Errorf("file target is required")
	}

	return h.validator.ValidateFilePath(step.Target)
}

// Execute executes the file operation
func (h *FileHandler) Execute(ctx context.Context, step types.FixStep, execCtx *ExecutionContext) (*StepResult, error) {
	switch step.Action {
	case ActionUpdateFile:
		return h.updateFile(ctx, step, execCtx)
	case ActionCreateFile:
		return h.createFile(ctx, step, execCtx)
	case ActionDeleteFile:
		return h.deleteFile(ctx, step, execCtx)
	}

	return nil, fmt.Errorf("unsupported action: %s", step.Action)
}

// Rollback reverses the file operation
func (h *FileHandler) Rollback(ctx context.Context, step types.FixStep, execCtx *ExecutionContext) error {
	h.logger.Infof("Rolling back file operation: %s", step.Action)

	// Find the rollback data for this step
	for _, rollback := range execCtx.RollbackData {
		if rollback.Action == step.Action {
			// Restore original file content
			filePath := filepath.Join(execCtx.WorkingDirectory, step.Target)

			if rollback.OriginalData == nil {
				// File didn't exist before, delete it
				return os.Remove(filePath)
			}

			// Restore original content
			originalContent, ok := rollback.OriginalData.(string)
			if !ok {
				return fmt.Errorf("invalid rollback data type")
			}

			return os.WriteFile(filePath, []byte(originalContent), 0644)
		}
	}

	return nil
}

// updateFile updates a file's content
func (h *FileHandler) updateFile(ctx context.Context, step types.FixStep, execCtx *ExecutionContext) (*StepResult, error) {
	filePath := h.getFullPath(execCtx.WorkingDirectory, step.Target)

	h.logger.Debugf("Updating file: %s", filePath)

	// Read original content (for rollback)
	originalContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Store for rollback
	execCtx.RollbackData = append(execCtx.RollbackData, RollbackData{
		Action:       ActionUpdateFile,
		OriginalData: string(originalContent),
		Timestamp:    execCtx.StartedAt,
	})

	// Determine update strategy
	content := step.Parameters["content"]
	pattern := step.Parameters["pattern"]
	replacement := step.Parameters["replacement"]

	var newContent string

	if content != "" {
		// Full replacement
		newContent = content
	} else if pattern != "" && replacement != "" {
		// Pattern-based replacement
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		newContent = re.ReplaceAllString(string(originalContent), replacement)
	} else {
		return nil, fmt.Errorf("either 'content' or 'pattern'+'replacement' must be provided")
	}

	// Write updated content
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &StepResult{
		Success: true,
		Output:  fmt.Sprintf("Updated %s (%d bytes → %d bytes)", step.Target, len(originalContent), len(newContent)),
	}, nil
}

// createFile creates a new file
func (h *FileHandler) createFile(ctx context.Context, step types.FixStep, execCtx *ExecutionContext) (*StepResult, error) {
	filePath := h.getFullPath(execCtx.WorkingDirectory, step.Target)

	h.logger.Debugf("Creating file: %s", filePath)

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		return nil, fmt.Errorf("file already exists: %s", step.Target)
	}

	// Store for rollback (file didn't exist)
	execCtx.RollbackData = append(execCtx.RollbackData, RollbackData{
		Action:       ActionCreateFile,
		OriginalData: nil, // nil indicates file didn't exist
		Timestamp:    execCtx.StartedAt,
	})

	// Get content from parameters
	content := step.Parameters["content"]
	if content == "" {
		return nil, fmt.Errorf("content parameter is required for create_file")
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(filePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return &StepResult{
		Success: true,
		Output:  fmt.Sprintf("Created %s (%d bytes)", step.Target, len(content)),
	}, nil
}

// deleteFile deletes a file
func (h *FileHandler) deleteFile(ctx context.Context, step types.FixStep, execCtx *ExecutionContext) (*StepResult, error) {
	filePath := h.getFullPath(execCtx.WorkingDirectory, step.Target)

	h.logger.Debugf("Deleting file: %s", filePath)

	// Read original content (for rollback)
	originalContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Store for rollback
	execCtx.RollbackData = append(execCtx.RollbackData, RollbackData{
		Action:       ActionDeleteFile,
		OriginalData: string(originalContent),
		Timestamp:    execCtx.StartedAt,
	})

	// Delete file
	if err := os.Remove(filePath); err != nil {
		return nil, fmt.Errorf("failed to delete file: %w", err)
	}

	return &StepResult{
		Success: true,
		Output:  fmt.Sprintf("Deleted %s (%d bytes)", step.Target, len(originalContent)),
	}, nil
}

// getFullPath returns the full path to a file
func (h *FileHandler) getFullPath(workingDir, target string) string {
	// If target is already absolute, use it as-is
	if filepath.IsAbs(target) {
		return target
	}

	// If working directory is set, join with target
	if workingDir != "" {
		return filepath.Join(workingDir, target)
	}

	// Otherwise, use current directory
	return target
}
