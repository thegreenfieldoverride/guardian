package dependencies

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/ai"
	"liberation-guardian/internal/config"
	"liberation-guardian/pkg/types"
)

// DependencyEventProcessor handles dependency-related events and automates PR decisions
type DependencyEventProcessor struct {
	config           *config.Config
	logger           *logrus.Logger
	analyzer         *DependencyAnalyzer
	githubAutomation *GitHubAutomation
}

// NewDependencyEventProcessor creates a new dependency event processor
func NewDependencyEventProcessor(cfg *config.Config, logger *logrus.Logger, aiClient ai.AIClient) *DependencyEventProcessor {
	analyzer := NewDependencyAnalyzer(cfg, logger, aiClient)
	githubAutomation := NewGitHubAutomation(cfg, logger, analyzer)

	return &DependencyEventProcessor{
		config:           cfg,
		logger:           logger,
		analyzer:         analyzer,
		githubAutomation: githubAutomation,
	}
}

// ProcessDependencyEvent processes a dependency-related event
func (dep *DependencyEventProcessor) ProcessDependencyEvent(ctx context.Context, event *types.LiberationGuardianEvent) error {
	dep.logger.Infof("Processing dependency event: %s", event.ID)

	// Check if this is a Dependabot PR event
	if !dep.isDependabotEvent(event) {
		dep.logger.Debugf("Event %s is not a Dependabot event, skipping", event.ID)
		return nil
	}

	// Parse webhook payload
	webhook, err := dep.parseWebhookPayload(event.RawPayload)
	if err != nil {
		return fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Process the Dependabot PR
	result, err := dep.githubAutomation.HandleDependabotPR(ctx, webhook)
	if err != nil {
		dep.logger.Errorf("Failed to handle Dependabot PR: %v", err)
		return fmt.Errorf("failed to handle Dependabot PR: %w", err)
	}

	// Log the automation result
	dep.logDependencyAutomation(event, result)

	// Store the result for audit trail
	dep.storeDependencyResult(ctx, event, result)

	return nil
}

// isDependabotEvent checks if the event is from Dependabot
func (dep *DependencyEventProcessor) isDependabotEvent(event *types.LiberationGuardianEvent) bool {
	// Check event metadata for Dependabot indicators
	if isDependabot, exists := event.Metadata["is_dependabot"].(bool); exists && isDependabot {
		return true
	}

	// Check tags for Dependabot
	for _, tag := range event.Tags {
		if tag == "dependabot" {
			return true
		}
	}

	// Check if the event type indicates dependency update
	return event.Type == "dependency_update"
}

// parseWebhookPayload parses the raw webhook payload into a Dependabot webhook struct
func (dep *DependencyEventProcessor) parseWebhookPayload(rawPayload json.RawMessage) (*types.GitHubDependabotWebhook, error) {
	var webhook types.GitHubDependabotWebhook
	if err := json.Unmarshal(rawPayload, &webhook); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook payload: %w", err)
	}
	return &webhook, nil
}

// logDependencyAutomation logs the automation decision for audit purposes
func (dep *DependencyEventProcessor) logDependencyAutomation(event *types.LiberationGuardianEvent, result *types.PRAutomationResult) {
	dep.logger.WithFields(map[string]interface{}{
		"event_id":    event.ID,
		"pr_id":       result.PRID,
		"action":      result.Action,
		"confidence":  result.Confidence,
		"trust_level": result.TrustLevel,
		"reasoning":   result.Reasoning,
		"cost":        result.Analysis.Cost,
		"ai_provider": result.Analysis.AIProvider,
	}).Info("Dependency automation completed")
}

// storeDependencyResult stores the automation result for audit and learning
func (dep *DependencyEventProcessor) storeDependencyResult(ctx context.Context, event *types.LiberationGuardianEvent, result *types.PRAutomationResult) {
	// In a full implementation, this would store to a database for:
	// 1. Audit trail
	// 2. Learning from human feedback
	// 3. Improving AI decision accuracy
	// 4. Cost tracking and optimization

	dep.logger.Debugf("Storing automation result for PR %s (action: %s, confidence: %.2f)",
		result.PRID, result.Action, result.Confidence)
}

// GetDependencyStats returns statistics about dependency automation
func (dep *DependencyEventProcessor) GetDependencyStats(ctx context.Context) (*DependencyStats, error) {
	// This would query stored results to provide insights
	return &DependencyStats{
		TotalPRsProcessed:       100, // Example values
		AutoApproved:            75,
		AutoMerged:              50,
		HumanReviewRequired:     20,
		Rejected:                5,
		AverageConfidence:       0.85,
		AverageCostPerPR:        0.003,  // $0.003 per PR
		TotalCostSavings:        2500.0, // $2500 saved vs manual review
		SecurityUpdatesFixed:    30,
		BreakingChangesDetected: 8,
	}, nil
}

// DependencyStats represents automation statistics
type DependencyStats struct {
	TotalPRsProcessed       int     `json:"total_prs_processed"`
	AutoApproved            int     `json:"auto_approved"`
	AutoMerged              int     `json:"auto_merged"`
	HumanReviewRequired     int     `json:"human_review_required"`
	Rejected                int     `json:"rejected"`
	AverageConfidence       float64 `json:"average_confidence"`
	AverageCostPerPR        float64 `json:"average_cost_per_pr"`
	TotalCostSavings        float64 `json:"total_cost_savings"`
	SecurityUpdatesFixed    int     `json:"security_updates_fixed"`
	BreakingChangesDetected int     `json:"breaking_changes_detected"`
}

// ValidateDependencyConfig validates the dependency automation configuration
func (dep *DependencyEventProcessor) ValidateDependencyConfig() error {
	config := dep.analyzer.depConfig

	// Validate trust level
	if config.TrustLevel < types.TrustParanoid || config.TrustLevel > types.TrustAutonomous {
		return fmt.Errorf("invalid trust level: %d", config.TrustLevel)
	}

	// Validate confidence thresholds
	if config.MinConfidence < 0.0 || config.MinConfidence > 1.0 {
		return fmt.Errorf("invalid confidence threshold: %.2f", config.MinConfidence)
	}

	// Validate test coverage requirements
	if config.MinTestCoverage < 0.0 || config.MinTestCoverage > 1.0 {
		return fmt.Errorf("invalid test coverage requirement: %.2f", config.MinTestCoverage)
	}

	// Validate custom rules
	for i, rule := range config.CustomRules {
		if rule.Name == "" {
			return fmt.Errorf("custom rule %d has empty name", i)
		}
		if rule.Action == "" {
			return fmt.Errorf("custom rule '%s' has empty action", rule.Name)
		}
	}

	dep.logger.Info("Dependency configuration validated successfully")
	return nil
}

// UpdateTrustLevel dynamically updates the trust level
func (dep *DependencyEventProcessor) UpdateTrustLevel(newLevel types.TrustLevel) error {
	if newLevel < types.TrustParanoid || newLevel > types.TrustAutonomous {
		return fmt.Errorf("invalid trust level: %d", newLevel)
	}

	oldLevel := dep.analyzer.depConfig.TrustLevel
	dep.analyzer.depConfig.TrustLevel = newLevel

	dep.logger.Infof("Trust level updated from %d to %d", oldLevel, newLevel)
	return nil
}

// GetTrustLevelDescription returns a human-readable description of the current trust level
func (dep *DependencyEventProcessor) GetTrustLevelDescription() string {
	switch dep.analyzer.depConfig.TrustLevel {
	case types.TrustParanoid:
		return "PARANOID: Human approval required for ALL dependency updates"
	case types.TrustConservative:
		return "CONSERVATIVE: Auto-approve patch and security updates only"
	case types.TrustBalanced:
		return "BALANCED: Auto-approve patch + minor security updates (RECOMMENDED)"
	case types.TrustProgressive:
		return "PROGRESSIVE: Auto-approve most updates with high confidence"
	case types.TrustAutonomous:
		return "AUTONOMOUS: Full automation with AI safety analysis"
	default:
		return "UNKNOWN: Invalid trust level configuration"
	}
}
