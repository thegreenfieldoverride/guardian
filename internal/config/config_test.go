package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "guardian.yml")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}

func TestLoadConfigAppliesDefaults(t *testing.T) {
	// A config that sets none of the defaulted fields.
	cfg, err := LoadConfig(writeConfig(t, "core:\n  environment: test\n"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Core.Port != 8080 {
		t.Errorf("Core.Port = %d, want default 8080", cfg.Core.Port)
	}
	if cfg.Core.LogLevel != "info" {
		t.Errorf("Core.LogLevel = %q, want default info", cfg.Core.LogLevel)
	}
	if cfg.Redis.Host != "localhost" {
		t.Errorf("Redis.Host = %q, want default localhost", cfg.Redis.Host)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis.Port = %d, want default 6379", cfg.Redis.Port)
	}
	if cfg.Core.Environment != "test" {
		t.Errorf("Core.Environment = %q, want the value from the file", cfg.Core.Environment)
	}
}

func TestLoadConfigDoesNotOverrideExplicitValues(t *testing.T) {
	cfg, err := LoadConfig(writeConfig(t, `
core:
  port: 9999
  log_level: debug
redis:
  host: redis.internal
  port: 6380
`))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Core.Port != 9999 {
		t.Errorf("Core.Port = %d, want 9999", cfg.Core.Port)
	}
	if cfg.Core.LogLevel != "debug" {
		t.Errorf("Core.LogLevel = %q, want debug", cfg.Core.LogLevel)
	}
	if cfg.Redis.Host != "redis.internal" {
		t.Errorf("Redis.Host = %q, want redis.internal", cfg.Redis.Host)
	}
	if cfg.Redis.Port != 6380 {
		t.Errorf("Redis.Port = %d, want 6380", cfg.Redis.Port)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	if _, err := LoadConfig(filepath.Join(t.TempDir(), "nope.yml")); err == nil {
		t.Fatal("expected an error for a missing config file, got nil")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	if _, err := LoadConfig(writeConfig(t, "core:\n  port: [not, a, number\n")); err == nil {
		t.Fatal("expected an error for malformed YAML, got nil")
	}
}

func TestGetWebhookSecretReadsEnv(t *testing.T) {
	cfg := &Config{}
	cfg.Integrations.Observability.Sentry.WebhookSecretEnv = "TEST_SENTRY_SECRET"
	cfg.Integrations.Observability.Grafana.WebhookSecretEnv = "TEST_GRAFANA_SECRET"
	cfg.Integrations.SourceControl.GitHub.WebhookSecretEnv = "TEST_GITHUB_SECRET"

	t.Setenv("TEST_SENTRY_SECRET", "sentry-value")
	t.Setenv("TEST_GRAFANA_SECRET", "grafana-value")
	t.Setenv("TEST_GITHUB_SECRET", "github-value")

	for integration, want := range map[string]string{
		"sentry":  "sentry-value",
		"grafana": "grafana-value",
		"github":  "github-value",
	} {
		if got := cfg.GetWebhookSecret(integration); got != want {
			t.Errorf("GetWebhookSecret(%q) = %q, want %q", integration, got, want)
		}
	}
}

// Prometheus has no case in GetWebhookSecret's switch, so it falls through to
// the default and returns "". In Receiver.validateWebhookSignature an empty
// secret short-circuits to "valid", which means Prometheus webhooks cannot be
// signature-checked even when a secret is configured for them — unlike every
// other enabled source. Pinning it so the gap is visible rather than implied.
func TestGetWebhookSecretHasNoPrometheusCase(t *testing.T) {
	cfg := &Config{}
	cfg.Integrations.Observability.Prometheus.Enabled = true

	if got := cfg.GetWebhookSecret("prometheus"); got != "" {
		t.Fatalf("GetWebhookSecret(\"prometheus\") = %q; this test is stale and the gap is closed", got)
	}
	t.Log("NOTE: prometheus falls through to the default branch, so its webhooks are never signature-checked")
}

func TestGetWebhookSecretUnknownIntegration(t *testing.T) {
	cfg := &Config{}
	if got := cfg.GetWebhookSecret("not-a-real-source"); got != "" {
		t.Errorf("GetWebhookSecret(unknown) = %q, want empty", got)
	}
}

// An unset env var yields "", which callers treat as "no secret configured"
// rather than "misconfigured". Worth pinning: a typo'd env name is
// indistinguishable from deliberately having no secret.
func TestGetWebhookSecretUnsetEnvReturnsEmpty(t *testing.T) {
	cfg := &Config{}
	cfg.Integrations.Observability.Sentry.WebhookSecretEnv = "DEFINITELY_NOT_SET_12345"

	if got := cfg.GetWebhookSecret("sentry"); got != "" {
		t.Errorf("GetWebhookSecret = %q, want empty for an unset env var", got)
	}
}

func TestGetAIProviderAPIKey(t *testing.T) {
	t.Setenv("TEST_ANTHROPIC_KEY", "sk-test-value")

	cfg := &Config{
		AIProviders: map[string]AIProviderConfig{
			"triage_agent": {Provider: "anthropic", APIKeyEnv: "TEST_ANTHROPIC_KEY"},
		},
	}

	if got := cfg.GetAIProviderAPIKey("triage_agent"); got != "sk-test-value" {
		t.Errorf("GetAIProviderAPIKey = %q, want sk-test-value", got)
	}
	if got := cfg.GetAIProviderAPIKey("no_such_agent"); got != "" {
		t.Errorf("GetAIProviderAPIKey(unknown) = %q, want empty", got)
	}
}

func TestGetSlackWebhookURL(t *testing.T) {
	t.Setenv("TEST_SLACK_URL", "https://hooks.slack.test/abc")

	cfg := &Config{}
	cfg.Integrations.Notifications.Slack.WebhookURLEnv = "TEST_SLACK_URL"

	if got := cfg.GetSlackWebhookURL(); got != "https://hooks.slack.test/abc" {
		t.Errorf("GetSlackWebhookURL = %q", got)
	}
}
