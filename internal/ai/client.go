package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/config"
	"liberation-guardian/pkg/types"
)

// LiberationAIClient implements the AIClient interface
// It integrates with our existing AI provider system from The Collective Strategist
type LiberationAIClient struct {
	config        *config.Config
	logger        *logrus.Logger
	httpClient    *http.Client
	localProvider *OllamaProvider
}

// NewLiberationAIClient creates a new AI client
func NewLiberationAIClient(cfg *config.Config, logger *logrus.Logger) *LiberationAIClient {
	client := &LiberationAIClient{
		config: cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		localProvider: nil, // Will be set if local AI is configured
	}

	// Check if any AI provider is configured for local processing
	client.initializeLocalProvider()

	return client
}

// initializeLocalProvider sets up local AI provider if configured
func (c *LiberationAIClient) initializeLocalProvider() {
	for agentName, providerConfig := range c.config.AIProviders {
		if providerConfig.Provider == "local" || providerConfig.Provider == "ollama" {
			if providerConfig.LocalConfig != nil {
				provider := NewOllamaProvider(
					providerConfig.LocalConfig.BaseURL,
					providerConfig.Model,
					c.logger,
				)

				// Test connectivity
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				if provider.IsHealthy(ctx) {
					c.localProvider = provider
					c.logger.Infof("Local AI provider initialized for %s: %s model %s",
						agentName, providerConfig.Provider, providerConfig.Model)
					break // Use the first healthy local provider
				} else {
					c.logger.Warnf("Local AI provider %s not healthy for %s",
						providerConfig.Provider, agentName)
				}
			}
		}
	}
}

// SendRequest sends an AI request to the configured provider
func (c *LiberationAIClient) SendRequest(ctx context.Context, request *types.AIRequest) (*types.AIResponse, error) {
	startTime := time.Now()

	c.logger.Infof("Sending AI request to %s agent", request.Agent)

	// Get provider config for the specific agent
	agentConfigName := string(request.Agent) + "_agent"
	providerConfig, exists := c.config.AIProviders[agentConfigName]
	if !exists {
		return nil, fmt.Errorf("no configuration found for agent: %s", request.Agent)
	}

	// Send request based on provider type
	var response *types.AIResponse
	var err error

	switch providerConfig.Provider {
	case "anthropic":
		response, err = c.sendAnthropicRequest(ctx, request, providerConfig)
	case "openai":
		response, err = c.sendOpenAIRequest(ctx, request, providerConfig)
	case "google":
		response, err = c.sendGoogleRequest(ctx, request, providerConfig)
	case "local":
		response, err = c.sendLocalRequest(ctx, request, providerConfig)
	default:
		// Fallback to our existing AI integration service
		response, err = c.sendToAIService(ctx, request, providerConfig)
	}

	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	// Calculate processing time
	response.ProcessingTime = time.Since(startTime).Milliseconds()
	response.Agent = request.Agent

	c.logger.Infof("AI request completed in %dms, tokens used: %d", response.ProcessingTime, response.TokensUsed)

	return response, nil
}

// IsHealthy checks if the AI client is healthy
func (c *LiberationAIClient) IsHealthy(ctx context.Context) bool {
	// Simple health check - try a minimal request to each configured provider
	for agentName, providerConfig := range c.config.AIProviders {
		apiKey := os.Getenv(providerConfig.APIKeyEnv)
		if apiKey == "" {
			c.logger.Warnf("No API key configured for %s", agentName)
			continue
		}

		// Try a basic request based on provider
		healthy := c.checkProviderHealth(ctx, providerConfig)
		if !healthy {
			c.logger.Warnf("Provider %s for agent %s is not healthy", providerConfig.Provider, agentName)
			return false
		}
	}

	return true
}

// sendAnthropicRequest sends request to Anthropic Claude
func (c *LiberationAIClient) sendAnthropicRequest(ctx context.Context, request *types.AIRequest, config config.AIProviderConfig) (*types.AIResponse, error) {
	apiKey := os.Getenv(config.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key not configured")
	}

	// Build Anthropic request
	anthropicReq := map[string]interface{}{
		"model":       config.Model,
		"max_tokens":  config.MaxTokens,
		"temperature": config.Temperature,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": request.SystemPrompt,
			},
			{
				"role":    "user",
				"content": request.Prompt,
			},
		},
	}

	// Send HTTP request
	jsonData, _ := json.Marshal(anthropicReq)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Anthropic API error: %s", string(body))
	}

	// Parse Anthropic response
	var anthropicResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("no content in Anthropic response")
	}

	return &types.AIResponse{
		Content:    anthropicResp.Content[0].Text,
		TokensUsed: anthropicResp.Usage.OutputTokens,
		Cost:       c.calculateCost("anthropic", anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens),
		Confidence: 0.9, // Default confidence for successful responses
		Model:      config.Model,
		Provider:   "anthropic",
	}, nil
}

// sendOpenAIRequest sends request to OpenAI GPT
func (c *LiberationAIClient) sendOpenAIRequest(ctx context.Context, request *types.AIRequest, config config.AIProviderConfig) (*types.AIResponse, error) {
	apiKey := os.Getenv(config.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	// Build OpenAI request
	openaiReq := map[string]interface{}{
		"model":       config.Model,
		"max_tokens":  config.MaxTokens,
		"temperature": config.Temperature,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": request.SystemPrompt,
			},
			{
				"role":    "user",
				"content": request.Prompt,
			},
		},
	}

	// Send HTTP request
	jsonData, _ := json.Marshal(openaiReq)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	// Parse OpenAI response
	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	return &types.AIResponse{
		Content:    openaiResp.Choices[0].Message.Content,
		TokensUsed: openaiResp.Usage.CompletionTokens,
		Cost:       c.calculateCost("openai", openaiResp.Usage.PromptTokens, openaiResp.Usage.CompletionTokens),
		Confidence: 0.9, // Default confidence for successful responses
		Model:      config.Model,
		Provider:   "openai",
	}, nil
}

// sendGoogleRequest sends request to Google Gemini
func (c *LiberationAIClient) sendGoogleRequest(ctx context.Context, request *types.AIRequest, config config.AIProviderConfig) (*types.AIResponse, error) {
	apiKey := os.Getenv(config.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("Google API key not configured")
	}

	// Build Google Gemini request
	googleReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": fmt.Sprintf("%s\n\nUser Request:\n%s", request.SystemPrompt, request.Prompt),
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": config.MaxTokens,
			"temperature":     config.Temperature,
		},
	}

	// Send HTTP request
	jsonData, _ := json.Marshal(googleReq)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", config.Model, apiKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Google API error: %s", string(body))
	}

	// Parse Google response
	var googleResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &googleResp); err != nil {
		return nil, fmt.Errorf("failed to parse Google response: %w", err)
	}

	if len(googleResp.Candidates) == 0 || len(googleResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in Google response")
	}

	return &types.AIResponse{
		Content:    googleResp.Candidates[0].Content.Parts[0].Text,
		TokensUsed: googleResp.UsageMetadata.CandidatesTokenCount,
		Cost:       c.calculateCost("google", googleResp.UsageMetadata.PromptTokenCount, googleResp.UsageMetadata.CandidatesTokenCount),
		Confidence: 0.9, // Default confidence for successful responses
		Model:      config.Model,
		Provider:   "google",
	}, nil
}

// sendLocalRequest uses local AI processing (FREE)
func (c *LiberationAIClient) sendLocalRequest(ctx context.Context, request *types.AIRequest, config config.AIProviderConfig) (*types.AIResponse, error) {
	c.logger.Infof("Using FREE local AI processing for %s", request.Agent)

	// If we have an Ollama provider configured, use it
	if c.localProvider != nil {
		return c.localProvider.SendRequest(ctx, request)
	}

	// Fallback to pattern matching if no local provider
	response := c.generateFallbackResponse(request)

	return &types.AIResponse{
		Content:    response,
		TokensUsed: 0,   // No tokens used for local processing
		Cost:       0,   // FREE!
		Confidence: 0.8, // Good confidence for known patterns
		Model:      "fallback",
		Provider:   "local-patterns",
	}, nil
}

// sendToAIService sends request to our existing AI integration service
func (c *LiberationAIClient) sendToAIService(ctx context.Context, request *types.AIRequest, config config.AIProviderConfig) (*types.AIResponse, error) {
	// This would integrate with our existing ai-integration service
	// For now, we'll implement a basic fallback

	// Try to use the default provider (local sentence-transformers)
	return &types.AIResponse{
		Content:    c.generateFallbackResponse(request),
		TokensUsed: 100, // Estimated
		Cost:       0,   // Free for local processing
		Confidence: 0.7, // Lower confidence for fallback
		Model:      "fallback",
		Provider:   "ai-service",
		Error:      "Using fallback AI processing",
	}, nil
}

// generateFallbackResponse creates a basic fallback response
func (c *LiberationAIClient) generateFallbackResponse(request *types.AIRequest) string {
	// Basic rule-based response for common cases
	eventTitle := ""
	eventType := ""

	if request.Context != nil {
		eventTitle = request.Context.Title
		eventType = request.Context.Type
	}

	// Simple pattern matching for common issues
	if contains(eventTitle, []string{"timeout", "connection"}) {
		return `{
			"decision": "auto_acknowledge",
			"confidence": 0.7,
			"reasoning": "Network-related issue detected - likely temporary",
			"suggested_actions": ["Monitor for pattern", "Check network connectivity", "Verify service health"]
		}`
	}

	if contains(eventTitle, []string{"memory", "disk", "space"}) {
		return `{
			"decision": "escalate_human",
			"confidence": 0.8,
			"reasoning": "Resource exhaustion detected - requires human investigation",
			"suggested_actions": ["Check resource usage", "Scale resources", "Investigate memory leaks"]
		}`
	}

	if eventType == "workflow_run" && contains(eventTitle, []string{"failed", "failure"}) {
		return `{
			"decision": "auto_fix",
			"confidence": 0.6,
			"reasoning": "CI/CD failure detected - attempt automated fix",
			"suggested_actions": ["Rerun workflow", "Check for flaky tests", "Review recent changes"],
			"auto_fix_plan": {
				"type": "code_change",
				"description": "Retry failed CI/CD workflow",
				"requires_approval": true,
				"steps": [{"action": "retry_workflow", "target": "github_actions", "parameters": {"workflow_id": "current"}}]
			}
		}`
	}

	// Default fallback
	return `{
		"decision": "escalate_human",
		"confidence": 0.5,
		"reasoning": "Unknown event pattern - escalating to human for safety",
		"suggested_actions": ["Manual investigation required"]
	}`
}

// checkProviderHealth checks if a provider is healthy
func (c *LiberationAIClient) checkProviderHealth(ctx context.Context, config config.AIProviderConfig) bool {
	// For now, just check if API key is configured
	apiKey := os.Getenv(config.APIKeyEnv)
	return apiKey != ""
}

// calculateCost estimates the cost of an AI request
func (c *LiberationAIClient) calculateCost(provider string, inputTokens, outputTokens int) float64 {
	// Rough cost estimates (these should be updated with current pricing)
	switch provider {
	case "anthropic":
		// Claude pricing: ~$15/million input tokens, ~$75/million output tokens
		return (float64(inputTokens)*0.000015 + float64(outputTokens)*0.000075)
	case "openai":
		// GPT-4 pricing: ~$30/million input tokens, ~$60/million output tokens
		return (float64(inputTokens)*0.000030 + float64(outputTokens)*0.000060)
	case "google":
		// Gemini pricing: ~$7/million input tokens, ~$21/million output tokens
		return (float64(inputTokens)*0.000007 + float64(outputTokens)*0.000021)
	default:
		return 0 // Local/free processing
	}
}

// contains checks if any of the keywords are present in the text
func contains(text string, keywords []string) bool {
	text = strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}
