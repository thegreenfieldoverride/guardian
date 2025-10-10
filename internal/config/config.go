package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the Liberation Guardian configuration
type Config struct {
	Core          CoreConfig                  `yaml:"core"`
	Redis         RedisConfig                 `yaml:"redis"`
	AIProviders   map[string]AIProviderConfig `yaml:"ai_providers"`
	Integrations  IntegrationsConfig          `yaml:"integrations"`
	DecisionRules DecisionRulesConfig         `yaml:"decision_rules"`
	Learning      LearningConfig              `yaml:"learning"`
}

// CoreConfig represents core application settings
type CoreConfig struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	LogLevel    string `yaml:"log_level"`
	Port        int    `yaml:"port"`
}

// RedisConfig represents Redis connection settings
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// AIProviderConfig represents AI provider settings
type AIProviderConfig struct {
	Provider    string  `yaml:"provider"`
	Model       string  `yaml:"model"`
	APIKeyEnv   string  `yaml:"api_key_env"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`

	// Local AI specific settings
	LocalConfig *LocalAIConfig `yaml:"local_config,omitempty"`
}

// LocalAIConfig represents configuration for local AI providers
type LocalAIConfig struct {
	BaseURL             string `yaml:"base_url"`              // e.g., "http://ollama:11434"
	HealthCheckInterval string `yaml:"health_check_interval"` // e.g., "30s"
	StartupTimeout      string `yaml:"startup_timeout"`       // e.g., "5m"
	ContextSize         int    `yaml:"context_size"`          // Model context window
}

// IntegrationsConfig represents external service integrations
type IntegrationsConfig struct {
	Observability ObservabilityConfig `yaml:"observability"`
	SourceControl SourceControlConfig `yaml:"source_control"`
	Notifications NotificationsConfig `yaml:"notifications"`
}

// ObservabilityConfig represents observability tool integrations
type ObservabilityConfig struct {
	Sentry     SentryConfig     `yaml:"sentry"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
	Grafana    GrafanaConfig    `yaml:"grafana"`
}

// SentryConfig represents Sentry integration settings
type SentryConfig struct {
	Enabled          bool   `yaml:"enabled"`
	WebhookSecretEnv string `yaml:"webhook_secret_env"`
	DSNEnv           string `yaml:"dsn_env"`
	AutoAcknowledge  bool   `yaml:"auto_acknowledge"`
}

// PrometheusConfig represents Prometheus integration settings
type PrometheusConfig struct {
	Enabled          bool   `yaml:"enabled"`
	ScrapeURL        string `yaml:"scrape_url"`
	AlertWebhookPort int    `yaml:"alert_webhook_port"`
}

// GrafanaConfig represents Grafana integration settings
type GrafanaConfig struct {
	Enabled          bool   `yaml:"enabled"`
	WebhookSecretEnv string `yaml:"webhook_secret_env"`
}

// SourceControlConfig represents source control integrations
type SourceControlConfig struct {
	GitHub GitHubConfig `yaml:"github"`
}

// GitHubConfig represents GitHub integration settings
type GitHubConfig struct {
	Enabled          bool   `yaml:"enabled"`
	TokenEnv         string `yaml:"token_env"`
	WebhookSecretEnv string `yaml:"webhook_secret_env"`
	AutoMergeEnabled bool   `yaml:"auto_merge_enabled"`
}

// NotificationsConfig represents notification channel settings
type NotificationsConfig struct {
	Slack SlackConfig `yaml:"slack"`
}

// SlackConfig represents Slack integration settings
type SlackConfig struct {
	Enabled       bool   `yaml:"enabled"`
	WebhookURLEnv string `yaml:"webhook_url_env"`
}

// DecisionRulesConfig represents AI decision-making rules
type DecisionRulesConfig struct {
	AutoAcknowledge AutoAcknowledgeConfig `yaml:"auto_acknowledge"`
	AutoFix         AutoFixConfig         `yaml:"auto_fix"`
	Escalate        EscalateConfig        `yaml:"escalate"`
}

// AutoAcknowledgeConfig represents auto-acknowledge rules
type AutoAcknowledgeConfig struct {
	Patterns   []string                  `yaml:"patterns"`
	Conditions AutoAcknowledgeConditions `yaml:"conditions"`
}

// AutoAcknowledgeConditions represents conditions for auto-acknowledge
type AutoAcknowledgeConditions struct {
	Frequency           string  `yaml:"frequency"`
	UserImpact          string  `yaml:"user_impact"`
	ConfidenceThreshold float64 `yaml:"confidence_threshold"`
}

// AutoFixConfig represents auto-fix rules
type AutoFixConfig struct {
	Patterns   []string          `yaml:"patterns"`
	Conditions AutoFixConditions `yaml:"conditions"`
}

// AutoFixConditions represents conditions for auto-fix
type AutoFixConditions struct {
	ConfidenceThreshold float64 `yaml:"confidence_threshold"`
	MaxFixAttempts      int     `yaml:"max_fix_attempts"`
	RequireTests        bool    `yaml:"require_tests"`
}

// EscalateConfig represents escalation rules
type EscalateConfig struct {
	Patterns   []string           `yaml:"patterns"`
	Conditions EscalateConditions `yaml:"conditions"`
}

// EscalateConditions represents conditions for escalation
type EscalateConditions struct {
	AlwaysEscalate       bool     `yaml:"always_escalate"`
	NotificationChannels []string `yaml:"notification_channels"`
}

// LearningConfig represents learning and knowledge base settings
type LearningConfig struct {
	KnowledgeBase KnowledgeBaseConfig `yaml:"knowledge_base"`
	FeedbackLoop  FeedbackLoopConfig  `yaml:"feedback_loop"`
}

// KnowledgeBaseConfig represents knowledge base settings
type KnowledgeBaseConfig struct {
	RetentionDays              int     `yaml:"retention_days"`
	PatternConfidenceThreshold float64 `yaml:"pattern_confidence_threshold"`
	MinOccurrencesForPattern   int     `yaml:"min_occurrences_for_pattern"`
}

// FeedbackLoopConfig represents feedback loop settings
type FeedbackLoopConfig struct {
	Enabled                bool    `yaml:"enabled"`
	HumanFeedbackWeight    float64 `yaml:"human_feedback_weight"`
	OutcomeTrackingEnabled bool    `yaml:"outcome_tracking_enabled"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate and set defaults
	if config.Core.Port == 0 {
		config.Core.Port = 8080
	}
	if config.Core.LogLevel == "" {
		config.Core.LogLevel = "info"
	}
	if config.Redis.Host == "" {
		config.Redis.Host = "localhost"
	}
	if config.Redis.Port == 0 {
		config.Redis.Port = 6379
	}

	return &config, nil
}

// GetAIProviderAPIKey retrieves API key from environment for a given provider
func (c *Config) GetAIProviderAPIKey(agentName string) string {
	if provider, exists := c.AIProviders[agentName]; exists {
		return os.Getenv(provider.APIKeyEnv)
	}
	return ""
}

// GetWebhookSecret retrieves webhook secret from environment
func (c *Config) GetWebhookSecret(integration string) string {
	switch integration {
	case "sentry":
		return os.Getenv(c.Integrations.Observability.Sentry.WebhookSecretEnv)
	case "grafana":
		return os.Getenv(c.Integrations.Observability.Grafana.WebhookSecretEnv)
	case "github":
		return os.Getenv(c.Integrations.SourceControl.GitHub.WebhookSecretEnv)
	default:
		return ""
	}
}

// GetNotificationCredentials retrieves notification service credentials
func (c *Config) GetSlackWebhookURL() string {
	return os.Getenv(c.Integrations.Notifications.Slack.WebhookURLEnv)
}
