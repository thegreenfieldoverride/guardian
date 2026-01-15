package autofix

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"liberation-guardian/pkg/types"
)

// ConfigHandler handles configuration file updates (YAML/JSON/ENV)
type ConfigHandler struct {
	logger    *logrus.Logger
	validator *SafetyValidator
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(logger *logrus.Logger, validator *SafetyValidator) *ConfigHandler {
	return &ConfigHandler{
		logger:    logger,
		validator: validator,
	}
}

// CanHandle returns true if this handler can handle the given action
func (h *ConfigHandler) CanHandle(action string) bool {
	return action == ActionUpdateConfig
}

// Validate validates the fix step
func (h *ConfigHandler) Validate(ctx context.Context, step types.FixStep) error {
	// Validate file path
	if step.Target == "" {
		return fmt.Errorf("config file target is required")
	}

	// Validate config file path
	if err := h.validator.ValidateFilePath(step.Target); err != nil {
		return fmt.Errorf("config file path validation failed: %w", err)
	}

	// Ensure key and value are provided
	if step.Parameters["key"] == "" {
		return fmt.Errorf("key parameter is required")
	}
	if step.Parameters["value"] == "" {
		return fmt.Errorf("value parameter is required")
	}

	return nil
}

// Execute executes the config update
func (h *ConfigHandler) Execute(ctx context.Context, step types.FixStep, execCtx *ExecutionContext) (*StepResult, error) {
	configFile := step.Target
	key := step.Parameters["key"]
	value := step.Parameters["value"]
	format := step.Parameters["format"] // yaml, json, env

	fullPath := h.getFullPath(execCtx.WorkingDirectory, configFile)

	h.logger.Debugf("Updating config file: %s (format: %s, key: %s)", fullPath, format, key)

	// Auto-detect format if not provided
	if format == "" {
		format = h.detectFormat(configFile)
	}

	// Read original content
	// #nosec G304 - File path is validated against safety rules in handler.Validate()
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Store for rollback
	execCtx.RollbackData = append(execCtx.RollbackData, RollbackData{
		Action:       ActionUpdateConfig,
		OriginalData: string(content),
		Timestamp:    execCtx.StartedAt,
	})

	// Update based on format
	var updatedContent []byte
	switch format {
	case "yaml", "yml":
		updatedContent, err = h.updateYAML(content, key, value)
	case "json":
		updatedContent, err = h.updateJSON(content, key, value)
	case "env":
		updatedContent, err = h.updateEnvFile(content, key, value)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	// Write updated config
	if err := os.WriteFile(fullPath, updatedContent, 0600); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	return &StepResult{
		Success: true,
		Output:  fmt.Sprintf("Updated %s: %s = %s", configFile, key, value),
	}, nil
}

// Rollback reverses the config update
func (h *ConfigHandler) Rollback(ctx context.Context, step types.FixStep, execCtx *ExecutionContext) error {
	h.logger.Infof("Rolling back config update: %s", step.Target)

	// Find the rollback data
	for _, rollback := range execCtx.RollbackData {
		if rollback.Action == ActionUpdateConfig {
			fullPath := h.getFullPath(execCtx.WorkingDirectory, step.Target)
			originalContent, ok := rollback.OriginalData.(string)
			if !ok {
				return fmt.Errorf("invalid rollback data type")
			}

			return os.WriteFile(fullPath, []byte(originalContent), 0600)
		}
	}

	return nil
}

// updateYAML updates a YAML configuration file
func (h *ConfigHandler) updateYAML(content []byte, key, value string) ([]byte, error) {
	var data map[string]interface{}

	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Support nested keys with dot notation: "database.host"
	keys := strings.Split(key, ".")
	current := data
	for i, k := range keys {
		if i == len(keys)-1 {
			// Last key - set the value
			current[k] = h.parseValue(value)
		} else {
			// Intermediate key - navigate or create nested map
			if _, ok := current[k]; !ok {
				current[k] = make(map[string]interface{})
			}
			nested, ok := current[k].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("key %s is not a map", k)
			}
			current = nested
		}
	}

	return yaml.Marshal(data)
}

// updateJSON updates a JSON configuration file
func (h *ConfigHandler) updateJSON(content []byte, key, value string) ([]byte, error) {
	var data map[string]interface{}

	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Support nested keys with dot notation
	keys := strings.Split(key, ".")
	current := data
	for i, k := range keys {
		if i == len(keys)-1 {
			current[k] = h.parseValue(value)
		} else {
			if _, ok := current[k]; !ok {
				current[k] = make(map[string]interface{})
			}
			nested, ok := current[k].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("key %s is not a map", k)
			}
			current = nested
		}
	}

	return json.MarshalIndent(data, "", "  ")
}

// updateEnvFile updates a .env file
func (h *ConfigHandler) updateEnvFile(content []byte, key, value string) ([]byte, error) {
	lines := strings.Split(string(content), "\n")
	updated := false
	result := make([]string, 0)

	// Update existing key or add at end
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			result = append(result, line)
			continue
		}

		// Check if this line contains the key
		if strings.HasPrefix(trimmed, key+"=") {
			result = append(result, fmt.Sprintf("%s=%s", key, value))
			updated = true
		} else {
			result = append(result, line)
		}
	}

	// If key not found, add it at the end
	if !updated {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}

	return []byte(strings.Join(result, "\n")), nil
}

// parseValue attempts to parse the value as the appropriate type
func (h *ConfigHandler) parseValue(value string) interface{} {
	// Try to parse as int
	var intVal int
	if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
		return intVal
	}

	// Try to parse as float
	var floatVal float64
	if _, err := fmt.Sscanf(value, "%f", &floatVal); err == nil {
		return floatVal
	}

	// Try to parse as bool
	if value == "true" || value == "false" {
		return value == "true"
	}

	// Return as string
	return value
}

// detectFormat detects the config file format from extension
func (h *ConfigHandler) detectFormat(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".env":
		return "env"
	default:
		// Check filename
		if strings.HasSuffix(filename, ".env") || strings.Contains(filename, ".env.") {
			return "env"
		}
		return "yaml" // Default to YAML
	}
}

// getFullPath returns the full path to the config file
func (h *ConfigHandler) getFullPath(workingDir, target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	if workingDir != "" {
		return filepath.Join(workingDir, target)
	}
	return target
}
