package types

import (
	"encoding/json"
	"time"
)

// LiberationGuardianEvent represents an observability event processed by Liberation Guardian
type LiberationGuardianEvent struct {
	ID            string                 `json:"id"`
	Source        string                 `json:"source"` // sentry, prometheus, github, etc.
	Type          string                 `json:"type"`   // error, alert, deployment, etc.
	Severity      Severity               `json:"severity"`
	Timestamp     time.Time              `json:"timestamp"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	RawPayload    json.RawMessage        `json:"raw_payload"`
	Metadata      map[string]interface{} `json:"metadata"`
	Fingerprint   string                 `json:"fingerprint"` // For deduplication
	Environment   string                 `json:"environment"`
	Service       string                 `json:"service"`
	Tags          []string               `json:"tags"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

// Severity levels for Liberation Guardian events
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// EventSource represents different observability sources
type EventSource string

const (
	SourceSentry     EventSource = "sentry"
	SourcePrometheus EventSource = "prometheus"
	SourceGrafana    EventSource = "grafana"
	SourceGitHub     EventSource = "github"
	SourceGitLab     EventSource = "gitlab"
	SourceCustom     EventSource = "custom"
)

// TriageResult represents the AI triage decision
type TriageResult struct {
	Decision           TriageDecision `json:"decision"`
	Confidence         float64        `json:"confidence"`
	Reasoning          string         `json:"reasoning"`
	SuggestedActions   []string       `json:"suggested_actions"`
	SimilarPatterns    []string       `json:"similar_patterns"`
	RequiresEscalation bool           `json:"requires_escalation"`
	AutoFixAttempt     *AutoFixPlan   `json:"auto_fix_attempt,omitempty"`
}

// TriageDecision represents possible AI triage decisions
type TriageDecision string

const (
	DecisionAutoAcknowledge TriageDecision = "auto_acknowledge"
	DecisionAutoFix         TriageDecision = "auto_fix"
	DecisionEscalateHuman   TriageDecision = "escalate_human"
	DecisionAnalyzeDeeper   TriageDecision = "analyze_deeper"
	DecisionIgnore          TriageDecision = "ignore"
)

// AutoFixPlan represents an automated fix attempt
type AutoFixPlan struct {
	Type             AutoFixType `json:"type"`
	Description      string      `json:"description"`
	Steps            []FixStep   `json:"steps"`
	EstimatedTime    int         `json:"estimated_time_minutes"`
	RequiresApproval bool        `json:"requires_approval"`
	RollbackPlan     []FixStep   `json:"rollback_plan"`
}

// AutoFixType represents different types of automated fixes
type AutoFixType string

const (
	FixTypeCodeChange       AutoFixType = "code_change"
	FixTypeConfigUpdate     AutoFixType = "config_update"
	FixTypeInfrastructure   AutoFixType = "infrastructure"
	FixTypeDependencyUpdate AutoFixType = "dependency_update"
	FixTypeEnvironmentVar   AutoFixType = "environment_variable"
)

// FixStep represents a single step in an automated fix
type FixStep struct {
	Action     string            `json:"action"`
	Target     string            `json:"target"`
	Parameters map[string]string `json:"parameters"`
	Validation string            `json:"validation"`
	OnFailure  string            `json:"on_failure"`
}

// AIAgent represents different AI agents in the system
type AIAgent string

const (
	AgentTriage   AIAgent = "triage"
	AgentAnalysis AIAgent = "analysis"
	AgentCoding   AIAgent = "coding"
	AgentInfraSec AIAgent = "infrastructure_security"
	AgentPerf     AIAgent = "performance"
)

// AIRequest represents a request to an AI agent
type AIRequest struct {
	Agent        AIAgent                  `json:"agent"`
	Context      *LiberationGuardianEvent `json:"context"`
	Prompt       string                   `json:"prompt"`
	SystemPrompt string                   `json:"system_prompt"`
	MaxTokens    int                      `json:"max_tokens"`
	Temperature  float64                  `json:"temperature"`
	Metadata     map[string]interface{}   `json:"metadata"`
}

// AIResponse represents a response from an AI agent
type AIResponse struct {
	Agent          AIAgent `json:"agent"`
	Content        string  `json:"content"`
	TokensUsed     int     `json:"tokens_used"`
	Cost           float64 `json:"cost"`
	Confidence     float64 `json:"confidence"`
	ProcessingTime int64   `json:"processing_time_ms"`
	Model          string  `json:"model,omitempty"`
	Provider       string  `json:"provider,omitempty"`
	Error          string  `json:"error,omitempty"`
}

// KnowledgePattern represents a learned pattern in the knowledge base
type KnowledgePattern struct {
	ID              string                 `json:"id"`
	PatternType     string                 `json:"pattern_type"`
	Signature       string                 `json:"signature"` // Hash of key characteristics
	Occurrences     int                    `json:"occurrences"`
	SuccessfulFixes int                    `json:"successful_fixes"`
	FailedFixes     int                    `json:"failed_fixes"`
	Confidence      float64                `json:"confidence"`
	LastSeen        time.Time              `json:"last_seen"`
	Resolution      *AutoFixPlan           `json:"resolution,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// Notification represents a notification to be sent
type Notification struct {
	ID         string                 `json:"id"`
	Type       NotificationType       `json:"type"`
	Severity   Severity               `json:"severity"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Channels   []NotificationChannel  `json:"channels"`
	Recipients []string               `json:"recipients"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

// NotificationType represents different notification types
type NotificationType string

const (
	NotificationAlert      NotificationType = "alert"
	NotificationResolution NotificationType = "resolution"
	NotificationEscalation NotificationType = "escalation"
	NotificationStatus     NotificationType = "status"
)

// NotificationChannel represents different notification channels
type NotificationChannel string

const (
	ChannelSlack     NotificationChannel = "slack"
	ChannelEmail     NotificationChannel = "email"
	ChannelWebhook   NotificationChannel = "webhook"
	ChannelPagerDuty NotificationChannel = "pagerduty"
)
