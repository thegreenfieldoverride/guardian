package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/config"
	"liberation-guardian/pkg/types"
)

// CostManager handles AI cost tracking and escalation decisions
type CostManager struct {
	config        *config.Config
	logger        *logrus.Logger
	dailySpend    float64
	hourlySpend   float64
	lastReset     time.Time
	lastHourReset time.Time
	mutex         sync.RWMutex
	lastExpensive time.Time // Cooldown tracking
}

// NewCostManager creates a new cost manager
func NewCostManager(cfg *config.Config, logger *logrus.Logger) *CostManager {
	return &CostManager{
		config:        cfg,
		logger:        logger,
		lastReset:     time.Now(),
		lastHourReset: time.Now(),
	}
}

// EscalationDecision represents the AI escalation decision
type EscalationDecision struct {
	Agent            types.AIAgent
	Reason           string
	EstimatedCost    float64
	WithinBudget     bool
	RequiresApproval bool
	FallbackStrategy string
}

// DetermineEscalation decides which AI agent to use based on cost and complexity
func (cm *CostManager) DetermineEscalation(ctx context.Context, event *types.LiberationGuardianEvent, previousAttempts []types.AIAgent) (*EscalationDecision, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Reset budgets if needed
	cm.resetBudgetsIfNeeded()

	// Start with cheapest tier
	if !cm.hasAttempted(previousAttempts, types.AgentTriage) {
		return cm.evaluateTier1(event)
	}

	// Escalate to tier 2 if tier 1 failed or low confidence
	if !cm.hasAttempted(previousAttempts, types.AgentAnalysis) {
		return cm.evaluateTier2(event, previousAttempts)
	}

	// Last resort: tier 3 (expensive)
	return cm.evaluateTier3(event, previousAttempts)
}

// evaluateTier1 - Cheap triage agent (Haiku)
func (cm *CostManager) evaluateTier1(event *types.LiberationGuardianEvent) (*EscalationDecision, error) {
	estimatedCost := 0.005 // ~$0.005 for typical triage request

	decision := &EscalationDecision{
		Agent:         types.AgentTriage,
		Reason:        "Initial triage with cost-effective model",
		EstimatedCost: estimatedCost,
		WithinBudget:  cm.isWithinBudget(estimatedCost),
	}

	if !decision.WithinBudget {
		decision.FallbackStrategy = "rule_based_only"
		cm.logger.Warnf("Budget exceeded, falling back to rule-based triage for event %s", event.ID)
	}

	return decision, nil
}

// evaluateTier2 - Moderate analysis agent (Sonnet)
func (cm *CostManager) evaluateTier2(event *types.LiberationGuardianEvent, previousAttempts []types.AIAgent) (*EscalationDecision, error) {
	estimatedCost := 0.05 // ~$0.05 for analysis request

	// Check if escalation is justified
	escalationReasons := cm.checkTier2EscalationReasons(event)
	if len(escalationReasons) == 0 {
		return &EscalationDecision{
			Agent:            types.AgentTriage, // Stay on tier 1
			Reason:           "No justification for tier 2 escalation",
			EstimatedCost:    0.005,
			WithinBudget:     true,
			FallbackStrategy: "rule_based_if_low_confidence",
		}, nil
	}

	decision := &EscalationDecision{
		Agent:         types.AgentAnalysis,
		Reason:        fmt.Sprintf("Escalating to analysis: %v", escalationReasons),
		EstimatedCost: estimatedCost,
		WithinBudget:  cm.isWithinBudget(estimatedCost),
	}

	if !decision.WithinBudget {
		decision.FallbackStrategy = "human_escalation"
		cm.logger.Warnf("Budget exceeded, escalating to human for event %s", event.ID)
	}

	return decision, nil
}

// evaluateTier3 - Expensive expert agent (Opus)
func (cm *CostManager) evaluateTier3(event *types.LiberationGuardianEvent, previousAttempts []types.AIAgent) (*EscalationDecision, error) {
	estimatedCost := 0.50 // ~$0.50 for expert analysis

	// Check cooldown period
	if time.Since(cm.lastExpensive) < 5*time.Minute {
		return &EscalationDecision{
			Agent:            types.AgentAnalysis, // Stay on tier 2
			Reason:           "Expert agent on cooldown (cost control)",
			EstimatedCost:    0.05,
			WithinBudget:     true,
			FallbackStrategy: "human_escalation_if_failed",
		}, nil
	}

	// Check if tier 3 escalation is justified
	escalationReasons := cm.checkTier3EscalationReasons(event)
	if len(escalationReasons) == 0 {
		return &EscalationDecision{
			Agent:            types.AgentAnalysis,
			Reason:           "No justification for expensive expert analysis",
			EstimatedCost:    0.05,
			WithinBudget:     true,
			FallbackStrategy: "human_escalation",
		}, nil
	}

	decision := &EscalationDecision{
		Agent:            types.AgentInfraSec, // Using this as expert agent
		Reason:           fmt.Sprintf("Critical escalation to expert: %v", escalationReasons),
		EstimatedCost:    estimatedCost,
		WithinBudget:     cm.isWithinBudget(estimatedCost),
		RequiresApproval: true, // Expensive calls need approval
	}

	if !decision.WithinBudget {
		decision.FallbackStrategy = "immediate_human_escalation"
		cm.logger.Errorf("Budget exceeded for critical event %s, immediate human intervention required", event.ID)
	}

	return decision, nil
}

// checkTier2EscalationReasons determines if tier 2 escalation is justified
func (cm *CostManager) checkTier2EscalationReasons(event *types.LiberationGuardianEvent) []string {
	reasons := []string{}

	// Critical or high severity
	if event.Severity == types.SeverityCritical || event.Severity == types.SeverityHigh {
		reasons = append(reasons, "high_severity")
	}

	// Production environment
	if event.Environment == "production" {
		reasons = append(reasons, "production_environment")
	}

	// Unknown patterns (no similar events in knowledge base)
	// This would be checked against the knowledge base in practice
	if len(event.Tags) == 0 || containsString(event.Tags, "unknown") {
		reasons = append(reasons, "unknown_pattern")
	}

	// Security-related events
	if containsString(event.Tags, "security") || containsAny(event.Title, []string{"security", "breach", "unauthorized", "vulnerability"}) {
		reasons = append(reasons, "security_related")
	}

	return reasons
}

// checkTier3EscalationReasons determines if expensive expert analysis is justified
func (cm *CostManager) checkTier3EscalationReasons(event *types.LiberationGuardianEvent) []string {
	reasons := []string{}

	// Critical security incidents
	if event.Severity == types.SeverityCritical && containsString(event.Tags, "security") {
		reasons = append(reasons, "critical_security_incident")
	}

	// Data loss or corruption
	if containsAny(event.Title, []string{"data loss", "corruption", "database failure"}) {
		reasons = append(reasons, "data_integrity_threat")
	}

	// Business-critical outages
	if containsAny(event.Description, []string{"revenue", "payment", "checkout", "critical service"}) {
		reasons = append(reasons, "business_critical_outage")
	}

	// Compliance violations
	if containsStringSlice(event.Tags, []string{"compliance", "audit", "regulation"}) {
		reasons = append(reasons, "compliance_violation")
	}

	return reasons
}

// RecordCost records the actual cost of an AI request
func (cm *CostManager) RecordCost(cost float64, agent types.AIAgent) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.dailySpend += cost
	cm.hourlySpend += cost

	if agent == types.AgentInfraSec { // Expert agent
		cm.lastExpensive = time.Now()
	}

	cm.logger.Infof("AI cost recorded: $%.4f for %s (daily: $%.2f, hourly: $%.2f)",
		cost, agent, cm.dailySpend, cm.hourlySpend)
}

// Helper methods
func (cm *CostManager) isWithinBudget(estimatedCost float64) bool {
	return (cm.dailySpend+estimatedCost <= 50.0) && (cm.hourlySpend+estimatedCost <= 10.0)
}

func (cm *CostManager) hasAttempted(attempts []types.AIAgent, agent types.AIAgent) bool {
	for _, a := range attempts {
		if a == agent {
			return true
		}
	}
	return false
}

func (cm *CostManager) resetBudgetsIfNeeded() {
	now := time.Now()

	// Reset daily budget at midnight
	if now.Day() != cm.lastReset.Day() {
		cm.dailySpend = 0
		cm.lastReset = now
		cm.logger.Info("Daily AI budget reset")
	}

	// Reset hourly budget
	if now.Hour() != cm.lastHourReset.Hour() {
		cm.hourlySpend = 0
		cm.lastHourReset = now
		cm.logger.Debugf("Hourly AI budget reset")
	}
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsStringSlice(slice []string, items []string) bool {
	for _, item := range items {
		if containsString(slice, item) {
			return true
		}
	}
	return false
}

func containsAny(text string, keywords []string) bool {
	// Simple substring check - could be enhanced with regex
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(text), strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}
