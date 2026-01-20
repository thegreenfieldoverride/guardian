package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/codebase"
	"liberation-guardian/internal/config"
	"liberation-guardian/pkg/types"
)

// TriageEngine handles AI-powered event triage
type TriageEngine struct {
	config           *config.Config
	logger           *logrus.Logger
	aiClient         AIClient
	knowledgeBase    KnowledgeBase
	patternMatcher   *PatternMatcher
	codebaseAnalyzer *codebase.CodebaseAnalyzer
}

// AIClient interface for making AI requests
type AIClient interface {
	SendRequest(ctx context.Context, request *types.AIRequest) (*types.AIResponse, error)
	IsHealthy(ctx context.Context) bool
}

// KnowledgeBase interface for accessing learned patterns
type KnowledgeBase interface {
	FindSimilarPatterns(ctx context.Context, event *types.LiberationGuardianEvent) ([]*types.KnowledgePattern, error)
	RecordResolution(ctx context.Context, eventID string, resolution *types.AutoFixPlan, success bool) error
	UpdatePatternConfidence(ctx context.Context, patternID string, feedback float64) error
}

// NewTriageEngine creates a new AI triage engine
func NewTriageEngine(cfg *config.Config, logger *logrus.Logger, aiClient AIClient, kb KnowledgeBase, codeAnalyzer *codebase.CodebaseAnalyzer) *TriageEngine {
	return &TriageEngine{
		config:           cfg,
		logger:           logger,
		aiClient:         aiClient,
		knowledgeBase:    kb,
		patternMatcher:   NewPatternMatcher(cfg.DecisionRules),
		codebaseAnalyzer: codeAnalyzer,
	}
}

// TriageEvent performs AI triage on an incoming event
func (te *TriageEngine) TriageEvent(ctx context.Context, event *types.LiberationGuardianEvent) (*types.TriageResult, error) {
	te.logger.Infof("Starting triage for event %s from %s", event.ID, event.Source)

	// Step 1: Check for immediate patterns that require escalation
	if te.shouldEscalateImmediately(event) {
		return &types.TriageResult{
			Decision:           types.DecisionEscalateHuman,
			Confidence:         1.0,
			Reasoning:          "Event matches critical escalation pattern",
			RequiresEscalation: true,
		}, nil
	}

	// Step 2: Check knowledge base for similar patterns
	similarPatterns, err := te.knowledgeBase.FindSimilarPatterns(ctx, event)
	if err != nil {
		te.logger.Warnf("Failed to query knowledge base: %v", err)
		similarPatterns = []*types.KnowledgePattern{}
	}

	// Step 3: Check rule-based patterns for auto-acknowledge
	if te.shouldAutoAcknowledge(event) {
		return &types.TriageResult{
			Decision:        types.DecisionAutoAcknowledge,
			Confidence:      0.9,
			Reasoning:       "Event matches auto-acknowledge pattern",
			SimilarPatterns: te.extractPatternIDs(similarPatterns),
		}, nil
	}

	// Step 4: AI-powered triage decision
	aiResult, err := te.performAITriage(ctx, event, similarPatterns)
	if err != nil {
		te.logger.Errorf("AI triage failed for event %s: %v", event.ID, err)
		// Fallback to rule-based decision
		return te.fallbackTriage(event), nil
	}

	return aiResult, nil
}

// shouldEscalateImmediately checks if event requires immediate escalation
func (te *TriageEngine) shouldEscalateImmediately(event *types.LiberationGuardianEvent) bool {
	// Critical severity always escalates
	if event.Severity == types.SeverityCritical {
		return true
	}

	// Check escalation patterns
	for _, pattern := range te.config.DecisionRules.Escalate.Patterns {
		matched, err := regexp.MatchString(pattern, event.Title)
		if err != nil {
			te.logger.Warnf("Invalid escalation pattern '%s': %v", pattern, err)
			continue
		}
		if matched {
			return true
		}
		matched, err = regexp.MatchString(pattern, event.Description)
		if err != nil {
			te.logger.Warnf("Invalid escalation pattern '%s': %v", pattern, err)
			continue
		}
		if matched {
			return true
		}
	}

	return false
}

// shouldAutoAcknowledge checks if event can be auto-acknowledged
func (te *TriageEngine) shouldAutoAcknowledge(event *types.LiberationGuardianEvent) bool {
	for _, pattern := range te.config.DecisionRules.AutoAcknowledge.Patterns {
		matched, err := regexp.MatchString(pattern, event.Title)
		if err != nil {
			te.logger.Warnf("Invalid auto-acknowledge pattern '%s': %v", pattern, err)
			continue
		}
		if matched {
			return true
		}
		matched, err = regexp.MatchString(pattern, event.Description)
		if err != nil {
			te.logger.Warnf("Invalid auto-acknowledge pattern '%s': %v", pattern, err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

// performAITriage uses AI to make triage decisions
func (te *TriageEngine) performAITriage(ctx context.Context, event *types.LiberationGuardianEvent, patterns []*types.KnowledgePattern) (*types.TriageResult, error) {
	// Build context for AI
	context := te.buildAIContext(event, patterns)

	// NEW: Add codebase analysis if available
	var codeContext *codebase.CodeContext
	if te.codebaseAnalyzer != nil {
		var err error
		codeContext, err = te.codebaseAnalyzer.AnalyzeForEvent(ctx, event)
		if err != nil {
			te.logger.Warnf("Codebase analysis failed: %v", err)
			// Continue without codebase context
		} else {
			te.logger.Infof("Codebase analysis complete: %d files analyzed, %d patterns detected",
				codeContext.FilesAnalyzed, len(codeContext.ErrorPatterns))
		}
	}

	// Create AI request
	request := &types.AIRequest{
		Agent:        types.AgentTriage,
		Context:      event,
		SystemPrompt: te.buildTriageSystemPrompt(),
		Prompt:       te.buildEnhancedTriagePrompt(event, context, codeContext),
		MaxTokens:    te.getMaxTokensForAgent(types.AgentTriage),
		Temperature:  te.getTemperatureForAgent(types.AgentTriage),
	}

	// Send to AI
	response, err := te.aiClient.SendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	// Parse AI response
	result, err := te.parseTriageResponse(response.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Validate confidence threshold
	if result.Confidence < te.config.DecisionRules.AutoFix.Conditions.ConfidenceThreshold {
		result.Decision = types.DecisionEscalateHuman
		result.RequiresEscalation = true
		result.Reasoning = fmt.Sprintf("Low confidence (%.2f) - escalating to human", result.Confidence)
	}

	result.SimilarPatterns = te.extractPatternIDs(patterns)

	return result, nil
}

// buildTriageSystemPrompt creates the system prompt for AI triage
func (te *TriageEngine) buildTriageSystemPrompt() string {
	return `You are Liberation Guardian, an AI-powered operations assistant that helps developers manage observability events autonomously.

Your role is to analyze incoming events (errors, alerts, deployment failures, etc.) and make intelligent triage decisions. You should:

1. CLASSIFY the event severity and type
2. DETERMINE if this requires immediate human attention or can be handled automatically
3. SUGGEST specific actions to resolve the issue
4. PROVIDE reasoning for your decision

Decision types:
- auto_acknowledge: Event is known/temporary, acknowledge and monitor
- auto_fix: Event has a known fix that can be automated
- escalate_human: Event requires human intervention
- analyze_deeper: Need more information before deciding
- ignore: Event is noise/false positive

Always respond in JSON format with these fields:
{
  "decision": "one of the decision types above",
  "confidence": 0.0-1.0,
  "reasoning": "explain your decision",
  "suggested_actions": ["action1", "action2"],
  "auto_fix_plan": {
    "type": "code_change|config_update|infrastructure|dependency_update|environment_variable",
    "description": "what will be done",
    "steps": [{"action": "step", "target": "where", "parameters": {}}],
    "requires_approval": boolean
  }
}

Be conservative - when in doubt, escalate to human.`
}

// buildTriagePrompt creates the specific prompt for this event
func (te *TriageEngine) buildTriagePrompt(event *types.LiberationGuardianEvent, context string) string {
	return fmt.Sprintf(`Analyze this observability event and provide a triage decision:

EVENT DETAILS:
Source: %s
Type: %s
Severity: %s
Title: %s
Description: %s
Service: %s
Environment: %s
Tags: %s

RAW PAYLOAD PREVIEW:
%s

SIMILAR PATTERNS FROM KNOWLEDGE BASE:
%s

SYSTEM CONFIGURATION:
- Auto-acknowledge confidence threshold: %.2f
- Auto-fix confidence threshold: %.2f
- Max fix attempts: %d
- Require tests for auto-fix: %t

Please analyze this event and provide your triage decision in JSON format.`,
		event.Source,
		event.Type,
		event.Severity,
		event.Title,
		event.Description,
		event.Service,
		event.Environment,
		strings.Join(event.Tags, ", "),
		te.truncatePayload(string(event.RawPayload), 500),
		context,
		te.config.DecisionRules.AutoAcknowledge.Conditions.ConfidenceThreshold,
		te.config.DecisionRules.AutoFix.Conditions.ConfidenceThreshold,
		te.config.DecisionRules.AutoFix.Conditions.MaxFixAttempts,
		te.config.DecisionRules.AutoFix.Conditions.RequireTests,
	)
}

// buildEnhancedTriagePrompt creates enhanced prompt with codebase context
func (te *TriageEngine) buildEnhancedTriagePrompt(event *types.LiberationGuardianEvent, context string, codeContext *codebase.CodeContext) string {
	basePrompt := te.buildTriagePrompt(event, context)

	if codeContext == nil {
		return basePrompt
	}

	// Add codebase analysis to the prompt
	codeAnalysis := "\n\nCODEBASE ANALYSIS:\n"
	codeAnalysis += fmt.Sprintf("Files analyzed: %d\n", codeContext.FilesAnalyzed)
	codeAnalysis += fmt.Sprintf("Analysis depth: %s\n", codeContext.AnalysisDepth)

	if len(codeContext.StackTraceFiles) > 0 {
		codeAnalysis += "\nSTACK TRACE FILES:\n"
		for _, file := range codeContext.StackTraceFiles {
			codeAnalysis += fmt.Sprintf("- %s (%s, %d lines, complexity: %d)\n",
				file.Path, file.Language, file.LineCount, file.Complexity)
			if file.CodeSnippet != "" {
				codeAnalysis += fmt.Sprintf("  Code context: %s\n", file.CodeSnippet)
			}
		}
	}

	if len(codeContext.RelevantFiles) > 0 {
		codeAnalysis += "\nRELEVANT FILES:\n"
		for _, file := range codeContext.RelevantFiles {
			codeAnalysis += fmt.Sprintf("- %s (%s, %d lines, last modified: %s)\n",
				file.Path, file.Language, file.LineCount, file.LastModified)
			if file.IsCritical {
				codeAnalysis += "  [CRITICAL FILE]\n"
			}
		}
	}

	if len(codeContext.RecentChanges) > 0 {
		codeAnalysis += "\nRECENT COMMITS:\n"
		for _, commit := range codeContext.RecentChanges {
			codeAnalysis += fmt.Sprintf("- %s by %s: %s (+%d -%d lines)\n",
				commit.Hash, commit.Author, commit.Message, commit.LinesAdded, commit.LinesRemoved)
		}
	}

	if len(codeContext.ErrorPatterns) > 0 {
		codeAnalysis += "\nDETECTED ERROR PATTERNS:\n"
		for _, pattern := range codeContext.ErrorPatterns {
			codeAnalysis += fmt.Sprintf("- %s in %s (confidence: %.2f): %s\n",
				pattern.Type, pattern.Location, pattern.Confidence, pattern.Description)
		}
	}

	if len(codeContext.Dependencies) > 0 {
		codeAnalysis += "\nDEPENDENCY ANALYSIS:\n"
		for _, dep := range codeContext.Dependencies {
			codeAnalysis += fmt.Sprintf("- %s (%s)\n", dep.Name, dep.Type)
			if dep.HasVulnerability {
				codeAnalysis += fmt.Sprintf("  [SECURITY RISK: %s]\n", dep.SecurityRisk)
			}
		}
	}

	return basePrompt + codeAnalysis
}

// buildAIContext creates context string from similar patterns
func (te *TriageEngine) buildAIContext(event *types.LiberationGuardianEvent, patterns []*types.KnowledgePattern) string {
	if len(patterns) == 0 {
		return "No similar patterns found in knowledge base."
	}

	var contextParts []string
	for i, pattern := range patterns {
		if i >= 3 { // Limit to top 3 patterns
			break
		}

		contextParts = append(contextParts, fmt.Sprintf(
			"Pattern %d: %s (Confidence: %.2f, Occurrences: %d, Success Rate: %.1f%%)",
			i+1,
			pattern.PatternType,
			pattern.Confidence,
			pattern.Occurrences,
			te.calculateSuccessRate(pattern),
		))
	}

	return strings.Join(contextParts, "\n")
}

// parseTriageResponse parses the AI's JSON response
func (te *TriageEngine) parseTriageResponse(content string) (*types.TriageResult, error) {
	// Try to extract JSON from the response
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}") + 1

	if jsonStart == -1 || jsonEnd == 0 {
		return nil, fmt.Errorf("no JSON found in AI response")
	}

	jsonContent := content[jsonStart:jsonEnd]

	var parsed struct {
		Decision         string   `json:"decision"`
		Confidence       float64  `json:"confidence"`
		Reasoning        string   `json:"reasoning"`
		SuggestedActions []string `json:"suggested_actions"`
		AutoFixPlan      *struct {
			Type             string `json:"type"`
			Description      string `json:"description"`
			RequiresApproval bool   `json:"requires_approval"`
			Steps            []struct {
				Action     string            `json:"action"`
				Target     string            `json:"target"`
				Parameters map[string]string `json:"parameters"`
			} `json:"steps"`
		} `json:"auto_fix_plan"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := &types.TriageResult{
		Decision:         types.TriageDecision(parsed.Decision),
		Confidence:       parsed.Confidence,
		Reasoning:        parsed.Reasoning,
		SuggestedActions: parsed.SuggestedActions,
	}

	// Convert auto-fix plan if present
	if parsed.AutoFixPlan != nil {
		result.AutoFixAttempt = &types.AutoFixPlan{
			Type:             types.AutoFixType(parsed.AutoFixPlan.Type),
			Description:      parsed.AutoFixPlan.Description,
			RequiresApproval: parsed.AutoFixPlan.RequiresApproval,
		}

		for _, step := range parsed.AutoFixPlan.Steps {
			result.AutoFixAttempt.Steps = append(result.AutoFixAttempt.Steps, types.FixStep{
				Action:     step.Action,
				Target:     step.Target,
				Parameters: step.Parameters,
			})
		}
	}

	return result, nil
}

// fallbackTriage provides rule-based fallback when AI fails
func (te *TriageEngine) fallbackTriage(event *types.LiberationGuardianEvent) *types.TriageResult {
	return &types.TriageResult{
		Decision:           types.DecisionEscalateHuman,
		Confidence:         0.5,
		Reasoning:          "AI triage failed, escalating to human as safety measure",
		RequiresEscalation: true,
	}
}

// Helper methods
func (te *TriageEngine) getMaxTokensForAgent(agent types.AIAgent) int {
	if config, exists := te.config.AIProviders[string(agent)]; exists {
		return config.MaxTokens
	}
	return 4000 // Default
}

func (te *TriageEngine) getTemperatureForAgent(agent types.AIAgent) float64 {
	if config, exists := te.config.AIProviders[string(agent)]; exists {
		return config.Temperature
	}
	return 0.1 // Default conservative temperature
}

func (te *TriageEngine) extractPatternIDs(patterns []*types.KnowledgePattern) []string {
	ids := make([]string, len(patterns))
	for i, pattern := range patterns {
		ids[i] = pattern.ID
	}
	return ids
}

func (te *TriageEngine) calculateSuccessRate(pattern *types.KnowledgePattern) float64 {
	total := pattern.SuccessfulFixes + pattern.FailedFixes
	if total == 0 {
		return 0
	}
	return (float64(pattern.SuccessfulFixes) / float64(total)) * 100
}

func (te *TriageEngine) truncatePayload(payload string, maxLength int) string {
	if len(payload) <= maxLength {
		return payload
	}
	return payload[:maxLength] + "..."
}

// PatternMatcher handles rule-based pattern matching
type PatternMatcher struct {
	rules config.DecisionRulesConfig
}

func NewPatternMatcher(rules config.DecisionRulesConfig) *PatternMatcher {
	return &PatternMatcher{rules: rules}
}

// MatchesPattern checks if event matches any configured patterns
func (pm *PatternMatcher) MatchesPattern(event *types.LiberationGuardianEvent, patterns []string) bool {
	text := fmt.Sprintf("%s %s", event.Title, event.Description)

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, text)
		if err != nil {
			continue // Skip invalid regex
		}
		if matched {
			return true
		}
	}

	return false
}
