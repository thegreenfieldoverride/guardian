package webhook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"liberation-guardian/pkg/types"
)

// SentryProcessor handles Sentry webhooks
type SentryProcessor struct {
	logger *logrus.Logger
}

func NewSentryProcessor(logger *logrus.Logger) *SentryProcessor {
	return &SentryProcessor{logger: logger}
}

func (p *SentryProcessor) GetEventSource() types.EventSource {
	return types.SourceSentry
}

func (p *SentryProcessor) ProcessWebhook(payload []byte, headers http.Header) (*types.LiberationGuardianEvent, error) {
	var sentryPayload struct {
		Action string `json:"action"`
		Data   struct {
			Issue struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				Level     string `json:"level"`
				Logger    string `json:"logger"`
				Platform  string `json:"platform"`
				Message   string `json:"message"`
				Timestamp string `json:"firstSeen"`
				Count     int    `json:"count"`
				URL       string `json:"permalink"`
				Project   struct {
					Name string `json:"name"`
					Slug string `json:"slug"`
				} `json:"project"`
			} `json:"issue"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &sentryPayload); err != nil {
		return nil, fmt.Errorf("failed to parse Sentry payload: %w", err)
	}

	timestamp, err := time.Parse(time.RFC3339, sentryPayload.Data.Issue.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	severity := p.mapSentrySeverity(sentryPayload.Data.Issue.Level)

	event := &types.LiberationGuardianEvent{
		ID:          uuid.New().String(),
		Source:      string(types.SourceSentry),
		Type:        sentryPayload.Action,
		Severity:    severity,
		Timestamp:   timestamp,
		Title:       sentryPayload.Data.Issue.Title,
		Description: sentryPayload.Data.Issue.Message,
		RawPayload:  json.RawMessage(payload),
		Metadata: map[string]interface{}{
			"sentry_issue_id": sentryPayload.Data.Issue.ID,
			"project":         sentryPayload.Data.Issue.Project.Name,
			"platform":        sentryPayload.Data.Issue.Platform,
			"logger":          sentryPayload.Data.Issue.Logger,
			"count":           sentryPayload.Data.Issue.Count,
			"url":             sentryPayload.Data.Issue.URL,
		},
		Environment: sentryPayload.Data.Issue.Project.Slug,
		Service:     sentryPayload.Data.Issue.Project.Name,
		Tags:        []string{"sentry", "error", sentryPayload.Data.Issue.Platform},
		Fingerprint: p.generateSentryFingerprint(sentryPayload.Data.Issue.ID, sentryPayload.Data.Issue.Title),
	}

	return event, nil
}

func (p *SentryProcessor) ValidateSignature(payload []byte, signature, secret string) bool {
	return ValidateHMAC(payload, signature, secret)
}

func (p *SentryProcessor) mapSentrySeverity(level string) types.Severity {
	switch strings.ToLower(level) {
	case "fatal", "error":
		return types.SeverityCritical
	case "warning":
		return types.SeverityHigh
	case "info":
		return types.SeverityMedium
	default:
		return types.SeverityLow
	}
}

func (p *SentryProcessor) generateSentryFingerprint(issueID, title string) string {
	data := fmt.Sprintf("sentry:%s:%s", issueID, title)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

// PrometheusProcessor handles Prometheus webhook alerts
type PrometheusProcessor struct {
	logger *logrus.Logger
}

func NewPrometheusProcessor(logger *logrus.Logger) *PrometheusProcessor {
	return &PrometheusProcessor{logger: logger}
}

func (p *PrometheusProcessor) GetEventSource() types.EventSource {
	return types.SourcePrometheus
}

func (p *PrometheusProcessor) ProcessWebhook(payload []byte, headers http.Header) (*types.LiberationGuardianEvent, error) {
	var prometheusPayload struct {
		Receiver string `json:"receiver"`
		Status   string `json:"status"`
		Alerts   []struct {
			Status       string            `json:"status"`
			Labels       map[string]string `json:"labels"`
			Annotations  map[string]string `json:"annotations"`
			StartsAt     string            `json:"startsAt"`
			EndsAt       string            `json:"endsAt"`
			GeneratorURL string            `json:"generatorURL"`
		} `json:"alerts"`
		GroupLabels       map[string]string `json:"groupLabels"`
		CommonLabels      map[string]string `json:"commonLabels"`
		CommonAnnotations map[string]string `json:"commonAnnotations"`
		ExternalURL       string            `json:"externalURL"`
	}

	if err := json.Unmarshal(payload, &prometheusPayload); err != nil {
		return nil, fmt.Errorf("failed to parse Prometheus payload: %w", err)
	}

	if len(prometheusPayload.Alerts) == 0 {
		return nil, fmt.Errorf("no alerts in Prometheus payload")
	}

	// Process the first alert (could be extended to handle multiple)
	alert := prometheusPayload.Alerts[0]

	timestamp, err := time.Parse(time.RFC3339, alert.StartsAt)
	if err != nil {
		timestamp = time.Now()
	}

	severity := p.mapPrometheusSeverity(alert.Labels["severity"])
	alertName := alert.Labels["alertname"]
	if alertName == "" {
		alertName = "Unknown Alert"
	}

	event := &types.LiberationGuardianEvent{
		ID:          uuid.New().String(),
		Source:      string(types.SourcePrometheus),
		Type:        prometheusPayload.Status,
		Severity:    severity,
		Timestamp:   timestamp,
		Title:       alertName,
		Description: alert.Annotations["description"],
		RawPayload:  json.RawMessage(payload),
		Metadata: map[string]interface{}{
			"receiver":      prometheusPayload.Receiver,
			"generator_url": alert.GeneratorURL,
			"external_url":  prometheusPayload.ExternalURL,
			"labels":        alert.Labels,
			"annotations":   alert.Annotations,
		},
		Environment: alert.Labels["environment"],
		Service:     alert.Labels["service"],
		Tags:        []string{"prometheus", "alert", prometheusPayload.Status},
		Fingerprint: p.generatePrometheusFingerprint(alertName, alert.Labels["instance"]),
	}

	return event, nil
}

func (p *PrometheusProcessor) ValidateSignature(payload []byte, signature, secret string) bool {
	// Prometheus doesn't typically use signatures, but could be extended
	return true
}

func (p *PrometheusProcessor) mapPrometheusSeverity(severity string) types.Severity {
	switch strings.ToLower(severity) {
	case "critical":
		return types.SeverityCritical
	case "warning":
		return types.SeverityHigh
	case "info":
		return types.SeverityMedium
	default:
		return types.SeverityLow
	}
}

func (p *PrometheusProcessor) generatePrometheusFingerprint(alertName, instance string) string {
	data := fmt.Sprintf("prometheus:%s:%s", alertName, instance)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

// GrafanaProcessor handles Grafana webhook alerts
type GrafanaProcessor struct {
	logger *logrus.Logger
}

func NewGrafanaProcessor(logger *logrus.Logger) *GrafanaProcessor {
	return &GrafanaProcessor{logger: logger}
}

func (p *GrafanaProcessor) GetEventSource() types.EventSource {
	return types.SourceGrafana
}

func (p *GrafanaProcessor) ProcessWebhook(payload []byte, headers http.Header) (*types.LiberationGuardianEvent, error) {
	var grafanaPayload struct {
		DashboardID int `json:"dashboardId"`
		EvalMatches []struct {
			Value  float64           `json:"value"`
			Metric string            `json:"metric"`
			Tags   map[string]string `json:"tags"`
		} `json:"evalMatches"`
		ImageURL string            `json:"imageUrl"`
		Message  string            `json:"message"`
		OrgID    int               `json:"orgId"`
		PanelID  int               `json:"panelId"`
		RuleID   int               `json:"ruleId"`
		RuleName string            `json:"ruleName"`
		RuleURL  string            `json:"ruleUrl"`
		State    string            `json:"state"`
		Tags     map[string]string `json:"tags"`
		Title    string            `json:"title"`
	}

	if err := json.Unmarshal(payload, &grafanaPayload); err != nil {
		return nil, fmt.Errorf("failed to parse Grafana payload: %w", err)
	}

	severity := p.mapGrafanaSeverity(grafanaPayload.State)

	event := &types.LiberationGuardianEvent{
		ID:          uuid.New().String(),
		Source:      string(types.SourceGrafana),
		Type:        grafanaPayload.State,
		Severity:    severity,
		Timestamp:   time.Now(),
		Title:       grafanaPayload.Title,
		Description: grafanaPayload.Message,
		RawPayload:  json.RawMessage(payload),
		Metadata: map[string]interface{}{
			"dashboard_id": grafanaPayload.DashboardID,
			"panel_id":     grafanaPayload.PanelID,
			"rule_id":      grafanaPayload.RuleID,
			"rule_name":    grafanaPayload.RuleName,
			"rule_url":     grafanaPayload.RuleURL,
			"image_url":    grafanaPayload.ImageURL,
			"org_id":       grafanaPayload.OrgID,
			"tags":         grafanaPayload.Tags,
		},
		Environment: grafanaPayload.Tags["environment"],
		Service:     grafanaPayload.Tags["service"],
		Tags:        []string{"grafana", "alert", grafanaPayload.State},
		Fingerprint: p.generateGrafanaFingerprint(grafanaPayload.RuleName, grafanaPayload.DashboardID),
	}

	return event, nil
}

func (p *GrafanaProcessor) ValidateSignature(payload []byte, signature, secret string) bool {
	// Grafana webhook signature validation could be implemented here
	return true
}

func (p *GrafanaProcessor) mapGrafanaSeverity(state string) types.Severity {
	switch strings.ToLower(state) {
	case "alerting":
		return types.SeverityCritical
	case "pending":
		return types.SeverityHigh
	case "ok":
		return types.SeverityLow
	default:
		return types.SeverityMedium
	}
}

func (p *GrafanaProcessor) generateGrafanaFingerprint(ruleName string, dashboardID int) string {
	data := fmt.Sprintf("grafana:%s:%d", ruleName, dashboardID)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

// GitHubProcessor handles GitHub webhooks
type GitHubProcessor struct {
	logger *logrus.Logger
}

func NewGitHubProcessor(logger *logrus.Logger) *GitHubProcessor {
	return &GitHubProcessor{logger: logger}
}

func (p *GitHubProcessor) GetEventSource() types.EventSource {
	return types.SourceGitHub
}

func (p *GitHubProcessor) ProcessWebhook(payload []byte, headers http.Header) (*types.LiberationGuardianEvent, error) {
	eventType := headers.Get("X-GitHub-Event")

	var githubPayload map[string]interface{}
	if err := json.Unmarshal(payload, &githubPayload); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub payload: %w", err)
	}

	severity := p.mapGitHubSeverity(eventType)
	title := p.generateGitHubTitle(eventType, githubPayload)
	description := p.generateGitHubDescription(eventType, githubPayload)

	event := &types.LiberationGuardianEvent{
		ID:          uuid.New().String(),
		Source:      string(types.SourceGitHub),
		Type:        eventType,
		Severity:    severity,
		Timestamp:   time.Now(),
		Title:       title,
		Description: description,
		RawPayload:  json.RawMessage(payload),
		Metadata:    githubPayload,
		Environment: "production", // Could be extracted from repo/branch
		Service:     p.extractRepoName(githubPayload),
		Tags:        []string{"github", eventType},
		Fingerprint: p.generateGitHubFingerprint(eventType, githubPayload),
	}

	return event, nil
}

func (p *GitHubProcessor) ValidateSignature(payload []byte, signature, secret string) bool {
	return ValidateHMAC(payload, signature, secret)
}

func (p *GitHubProcessor) mapGitHubSeverity(eventType string) types.Severity {
	switch eventType {
	case "security_advisory", "vulnerability_alert":
		return types.SeverityCritical
	case "check_run", "workflow_run":
		return types.SeverityHigh
	case "push", "pull_request":
		return types.SeverityMedium
	default:
		return types.SeverityLow
	}
}

func (p *GitHubProcessor) generateGitHubTitle(eventType string, payload map[string]interface{}) string {
	switch eventType {
	case "push":
		if repo, ok := payload["repository"].(map[string]interface{}); ok {
			if name, ok := repo["name"].(string); ok {
				return fmt.Sprintf("Push to %s", name)
			}
		}
	case "pull_request":
		if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
			if title, ok := pr["title"].(string); ok {
				return fmt.Sprintf("PR: %s", title)
			}
		}
	case "workflow_run":
		if workflow, ok := payload["workflow_run"].(map[string]interface{}); ok {
			if name, ok := workflow["name"].(string); ok {
				if conclusion, ok := workflow["conclusion"].(string); ok {
					return fmt.Sprintf("Workflow %s: %s", name, conclusion)
				}
			}
		}
	}
	return fmt.Sprintf("GitHub %s event", eventType)
}

func (p *GitHubProcessor) generateGitHubDescription(eventType string, payload map[string]interface{}) string {
	switch eventType {
	case "workflow_run":
		if workflow, ok := payload["workflow_run"].(map[string]interface{}); ok {
			if conclusion, ok := workflow["conclusion"].(string); ok && conclusion == "failure" {
				return "GitHub Actions workflow failed"
			}
		}
	case "pull_request":
		if action, ok := payload["action"].(string); ok {
			return fmt.Sprintf("Pull request %s", action)
		}
	}
	return fmt.Sprintf("GitHub %s event received", eventType)
}

func (p *GitHubProcessor) extractRepoName(payload map[string]interface{}) string {
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		if name, ok := repo["name"].(string); ok {
			return name
		}
	}
	return "unknown"
}

func (p *GitHubProcessor) generateGitHubFingerprint(eventType string, payload map[string]interface{}) string {
	repo := p.extractRepoName(payload)
	data := fmt.Sprintf("github:%s:%s", eventType, repo)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

// DependabotProcessor handles GitHub Dependabot PR webhooks
type DependabotProcessor struct {
	logger *logrus.Logger
}

func NewDependabotProcessor(logger *logrus.Logger) *DependabotProcessor {
	return &DependabotProcessor{logger: logger}
}

func (p *DependabotProcessor) GetEventSource() types.EventSource {
	return types.SourceGitHub // Dependabot is part of GitHub
}

func (p *DependabotProcessor) ProcessWebhook(payload []byte, headers http.Header) (*types.LiberationGuardianEvent, error) {
	var dependabotPayload types.GitHubDependabotWebhook

	if err := json.Unmarshal(payload, &dependabotPayload); err != nil {
		return nil, fmt.Errorf("failed to parse Dependabot webhook: %w", err)
	}

	// Only process Dependabot PRs
	if !p.isDependabotPR(&dependabotPayload) {
		return nil, fmt.Errorf("not a Dependabot PR")
	}

	// Only process specific actions
	if !p.shouldProcessAction(dependabotPayload.Action) {
		p.logger.Debugf("Ignoring Dependabot action: %s", dependabotPayload.Action)
		return nil, nil
	}

	severity := p.determineSeverity(&dependabotPayload)

	event := &types.LiberationGuardianEvent{
		ID:          uuid.New().String(),
		Source:      string(types.SourceGitHub),
		Type:        "dependency_update",
		Severity:    severity,
		Timestamp:   time.Now(),
		Title:       dependabotPayload.PullRequest.Title,
		Description: p.buildDescription(&dependabotPayload),
		RawPayload:  payload,
		Metadata:    p.buildMetadata(&dependabotPayload),
		Fingerprint: p.generateDependabotFingerprint(&dependabotPayload),
		Environment: "production", // Assume production unless specified
		Service:     dependabotPayload.Repository.Name,
		Tags:        p.buildTags(&dependabotPayload),
	}

	p.logger.Infof("Processed Dependabot PR: %s (#%d)", event.Title, dependabotPayload.PullRequest.Number)
	return event, nil
}

func (p *DependabotProcessor) isDependabotPR(webhook *types.GitHubDependabotWebhook) bool {
	return webhook.PullRequest.User.Login == "dependabot[bot]" ||
		webhook.PullRequest.User.Type == "Bot" &&
			strings.Contains(strings.ToLower(webhook.PullRequest.Title), "bump")
}

func (p *DependabotProcessor) shouldProcessAction(action string) bool {
	processableActions := []string{
		"opened",      // New Dependabot PR
		"reopened",    // Reopened PR
		"synchronize", // Updated PR
	}

	for _, processable := range processableActions {
		if action == processable {
			return true
		}
	}
	return false
}

func (p *DependabotProcessor) determineSeverity(webhook *types.GitHubDependabotWebhook) types.Severity {
	title := strings.ToLower(webhook.PullRequest.Title)
	body := strings.ToLower(webhook.PullRequest.Body)

	// Security updates are high priority
	if strings.Contains(body, "security") || strings.Contains(body, "cve-") {
		return types.SeverityHigh
	}

	// Major version updates are medium priority
	if strings.Contains(title, "major") {
		return types.SeverityMedium
	}

	// Patch updates are low priority
	return types.SeverityLow
}

func (p *DependabotProcessor) buildDescription(webhook *types.GitHubDependabotWebhook) string {
	return fmt.Sprintf("Dependabot created PR #%d: %s\n\nRepository: %s\nBranch: %s → %s",
		webhook.PullRequest.Number,
		webhook.PullRequest.Title,
		webhook.Repository.FullName,
		webhook.PullRequest.Head.Ref,
		webhook.PullRequest.Base.Ref,
	)
}

func (p *DependabotProcessor) buildMetadata(webhook *types.GitHubDependabotWebhook) map[string]interface{} {
	return map[string]interface{}{
		"pr_number":     webhook.PullRequest.Number,
		"pr_id":         webhook.PullRequest.ID,
		"pr_url":        webhook.PullRequest.URL,
		"repository":    webhook.Repository.FullName,
		"repo_id":       webhook.Repository.ID,
		"action":        webhook.Action,
		"head_ref":      webhook.PullRequest.Head.Ref,
		"head_sha":      webhook.PullRequest.Head.SHA,
		"base_ref":      webhook.PullRequest.Base.Ref,
		"author":        webhook.PullRequest.User.Login,
		"author_type":   webhook.PullRequest.User.Type,
		"created_at":    webhook.PullRequest.CreatedAt,
		"updated_at":    webhook.PullRequest.UpdatedAt,
		"is_dependabot": true,
	}
}

func (p *DependabotProcessor) buildTags(webhook *types.GitHubDependabotWebhook) []string {
	tags := []string{
		"dependabot",
		"dependency-update",
		"github",
	}

	title := strings.ToLower(webhook.PullRequest.Title)
	body := strings.ToLower(webhook.PullRequest.Body)

	// Add tags based on content
	if strings.Contains(body, "security") || strings.Contains(body, "cve-") {
		tags = append(tags, "security-update")
	}

	if strings.Contains(title, "major") {
		tags = append(tags, "major-update")
	} else if strings.Contains(title, "minor") {
		tags = append(tags, "minor-update")
	} else {
		tags = append(tags, "patch-update")
	}

	// Add ecosystem tags
	if strings.Contains(title, "npm") || strings.Contains(webhook.Repository.Name, "node") {
		tags = append(tags, "npm")
	}
	if strings.Contains(title, "python") || strings.Contains(webhook.Repository.Name, "python") {
		tags = append(tags, "python")
	}
	if strings.Contains(title, "go") || strings.Contains(webhook.Repository.Name, "go") {
		tags = append(tags, "golang")
	}

	return tags
}

func (p *DependabotProcessor) generateDependabotFingerprint(webhook *types.GitHubDependabotWebhook) string {
	// Create unique fingerprint for this specific dependency update
	data := fmt.Sprintf("dependabot:%s:%s:%s",
		webhook.Repository.FullName,
		webhook.PullRequest.Title,
		webhook.PullRequest.Head.Ref,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}
