package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/config"
	"liberation-guardian/pkg/types"
)

// OllamaProvider implements local AI using Ollama
type OllamaProvider struct {
	baseURL    string
	model      string
	httpClient *http.Client
	logger     *logrus.Logger
}

// OllamaRequest represents a request to Ollama API
type OllamaRequest struct {
	Model   string        `json:"model"`
	Prompt  string        `json:"prompt"`
	Stream  bool          `json:"stream"`
	Options OllamaOptions `json:"options,omitempty"`
}

// OllamaOptions for controlling generation
type OllamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// OllamaResponse represents a response from Ollama API
type OllamaResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Context  []int  `json:"context,omitempty"`
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(baseURL, model string, logger *logrus.Logger) *OllamaProvider {
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		logger:  logger,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Local models can be slow
		},
	}
}

// SendRequest sends a request to local Ollama model
func (o *OllamaProvider) SendRequest(ctx context.Context, request *types.AIRequest) (*types.AIResponse, error) {
	startTime := time.Now()

	o.logger.Infof("Sending request to local model %s via Ollama", o.model)

	// Build full prompt with system context
	fullPrompt := o.buildFullPrompt(request)

	// Create Ollama request
	ollamaReq := OllamaRequest{
		Model:  o.model,
		Prompt: fullPrompt,
		Stream: false,
		Options: OllamaOptions{
			Temperature: float64(request.Temperature),
			NumPredict:  request.MaxTokens,
		},
	}

	// Marshal request
	jsonData, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/generate", o.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Calculate processing time and token usage estimate
	processingTime := time.Since(startTime).Milliseconds()
	tokensUsed := o.estimateTokens(fullPrompt + ollamaResp.Response)

	o.logger.Infof("Local model request completed in %dms, estimated tokens: %d", processingTime, tokensUsed)

	return &types.AIResponse{
		Content:        ollamaResp.Response,
		TokensUsed:     tokensUsed,
		ProcessingTime: processingTime,
		Agent:          request.Agent,
		Model:          o.model,
		Provider:       "ollama",
		Cost:           0.0, // Local models are free!
	}, nil
}

// IsHealthy checks if Ollama is accessible and model is loaded
func (o *OllamaProvider) IsHealthy(ctx context.Context) bool {
	// Check if Ollama is running
	url := fmt.Sprintf("%s/api/tags", o.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		o.logger.Warnf("Failed to create health check request: %v", err)
		return false
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		o.logger.Warnf("Ollama health check failed: %v", err)
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		o.logger.Warnf("Ollama health check returned status %d", resp.StatusCode)
		return false
	}

	// Check if our specific model is available
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		o.logger.Warnf("Failed to read health check response: %v", err)
		return false
	}

	// Parse models list to verify our model is loaded
	var modelsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.Unmarshal(body, &modelsResp); err != nil {
		o.logger.Warnf("Failed to parse models response: %v", err)
		return false
	}

	// Check if our model is in the list
	for _, model := range modelsResp.Models {
		if model.Name == o.model {
			o.logger.Debugf("Model %s is available and healthy", o.model)
			return true
		}
	}

	o.logger.Warnf("Model %s not found in available models", o.model)
	return false
}

// buildFullPrompt combines system prompt and user prompt for local models
func (o *OllamaProvider) buildFullPrompt(request *types.AIRequest) string {
	if request.SystemPrompt == "" {
		return request.Prompt
	}

	// Format for better local model understanding
	return fmt.Sprintf(`System Instructions:
%s

User Request:
%s

Please provide a helpful response following the system instructions.`,
		request.SystemPrompt, request.Prompt)
}

// estimateTokens provides a rough token count estimate for local models
func (o *OllamaProvider) estimateTokens(text string) int {
	// Rough estimate: ~4 characters per token for English text
	// This is imprecise but good enough for tracking/logging
	return len(text) / 4
}

// LocalModelConfig represents configuration for local AI models
type LocalModelConfig struct {
	Provider string `yaml:"provider"` // "ollama"
	BaseURL  string `yaml:"base_url"` // "http://ollama:11434"
	Model    string `yaml:"model"`    // "qwen2.5:7b"

	// Performance tuning
	Temperature float32 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
	ContextSize int     `yaml:"context_size"`

	// Health check settings
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	StartupTimeout      time.Duration `yaml:"startup_timeout"`
}

// CreateLocalAIClient creates an AI client for local models
func CreateLocalAIClient(cfg LocalModelConfig, logger *logrus.Logger) (AIClient, error) {
	switch cfg.Provider {
	case "ollama":
		provider := NewOllamaProvider(cfg.BaseURL, cfg.Model, logger)

		// Verify model is accessible
		ctx, cancel := context.WithTimeout(context.Background(), cfg.StartupTimeout)
		defer cancel()

		if !provider.IsHealthy(ctx) {
			return nil, fmt.Errorf("local model %s not accessible at %s", cfg.Model, cfg.BaseURL)
		}

		logger.Infof("Local AI provider initialized: %s model %s", cfg.Provider, cfg.Model)
		return &LiberationAIClient{
			config:        &config.Config{}, // Will be set by caller
			logger:        logger,
			localProvider: provider,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported local AI provider: %s", cfg.Provider)
	}
}

// PullModel attempts to pull/download a model via Ollama
func (o *OllamaProvider) PullModel(ctx context.Context, modelName string) error {
	o.logger.Infof("Pulling model %s via Ollama...", modelName)

	pullReq := map[string]interface{}{
		"name": modelName,
	}

	jsonData, err := json.Marshal(pullReq)
	if err != nil {
		return fmt.Errorf("failed to marshal pull request: %w", err)
	}

	url := fmt.Sprintf("%s/api/pull", o.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send pull request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("model pull failed (status %d, failed to read response: %v)", resp.StatusCode, err)
		}
		return fmt.Errorf("model pull failed (status %d): %s", resp.StatusCode, string(body))
	}

	o.logger.Infof("Model %s pulled successfully", modelName)
	return nil
}
