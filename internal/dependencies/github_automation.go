package dependencies

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

// GitHubAutomation handles automated GitHub PR operations for dependencies
type GitHubAutomation struct {
	config      *config.Config
	logger      *logrus.Logger
	httpClient  *http.Client
	analyzer    *DependencyAnalyzer
	githubToken string
}

// NewGitHubAutomation creates a new GitHub automation handler
func NewGitHubAutomation(cfg *config.Config, logger *logrus.Logger, analyzer *DependencyAnalyzer) *GitHubAutomation {
	return &GitHubAutomation{
		config:      cfg,
		logger:      logger,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		analyzer:    analyzer,
		githubToken: os.Getenv("GITHUB_TOKEN"),
	}
}

// HandleDependabotPR processes a Dependabot PR and takes automated action
func (ga *GitHubAutomation) HandleDependabotPR(ctx context.Context, webhook *types.GitHubDependabotWebhook) (*types.PRAutomationResult, error) {
	ga.logger.Infof("Processing Dependabot PR #%d: %s", webhook.Number, webhook.PullRequest.Title)

	// Step 1: Parse dependency information from PR
	update, err := ga.parseDependencyUpdate(webhook)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dependency update: %w", err)
	}

	// Step 2: Analyze the dependency update
	analysis, err := ga.analyzer.AnalyzeDependencyUpdate(ctx, update)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze dependency update: %w", err)
	}

	// Step 3: Determine action based on analysis
	action := ga.determineAction(analysis, update)

	// Step 4: Execute the action
	result, err := ga.executeAction(ctx, webhook, action, analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to execute action: %w", err)
	}

	// Step 5: Log the automation result
	ga.logAutomationResult(result)

	return result, nil
}

// parseDependencyUpdate extracts dependency information from GitHub webhook
func (ga *GitHubAutomation) parseDependencyUpdate(webhook *types.GitHubDependabotWebhook) (*types.DependencyUpdate, error) {
	title := webhook.PullRequest.Title
	body := webhook.PullRequest.Body

	// Parse package name and versions from title
	// Example: "Bump lodash from 4.17.20 to 4.17.21"
	update := &types.DependencyUpdate{
		ID:         fmt.Sprintf("pr-%d", webhook.PullRequest.ID),
		Repository: webhook.Repository.FullName,
		PRNumber:   webhook.PullRequest.Number,
		PRUrl:      webhook.PullRequest.URL,
		CreatedAt:  time.Now(), // Would parse from webhook.PullRequest.CreatedAt
		Metadata: map[string]interface{}{
			"pr_author": webhook.PullRequest.User.Login,
			"pr_branch": webhook.PullRequest.Head.Ref,
		},
	}

	// Parse dependency information from title
	if err := ga.parseTitleForDependencyInfo(title, update); err != nil {
		return nil, err
	}

	// Parse additional information from body
	ga.parseBodyForDependencyInfo(body, update)

	// Determine ecosystem from repository or package name
	update.Ecosystem = ga.determineEcosystem(webhook.Repository.Name, update.PackageName)

	// Determine update type from version change
	update.UpdateType = ga.determineUpdateType(update.CurrentVersion, update.NewVersion)

	return update, nil
}

// parseTitleForDependencyInfo extracts package and version info from PR title
func (ga *GitHubAutomation) parseTitleForDependencyInfo(title string, update *types.DependencyUpdate) error {
	// Common Dependabot title patterns:
	// "Bump package from 1.0.0 to 1.0.1"
	// "Update package requirement from ~1.0.0 to ~1.1.0"
	// "Bump package from 1.0.0 to 1.1.0 in /subdirectory"

	patterns := []string{
		`Bump (.+) from (.+) to (.+)`,
		`Update (.+) requirement from (.+) to (.+)`,
		`Bump (.+) from (.+) to (.+) in`,
	}

	for _, pattern := range patterns {
		if matches := ga.extractMatches(title, pattern); len(matches) >= 4 {
			update.PackageName = strings.TrimSpace(matches[1])
			update.CurrentVersion = strings.TrimSpace(matches[2])
			update.NewVersion = strings.TrimSpace(matches[3])
			return nil
		}
	}

	return fmt.Errorf("could not parse dependency information from title: %s", title)
}

// parseBodyForDependencyInfo extracts additional info from PR body
func (ga *GitHubAutomation) parseBodyForDependencyInfo(body string, update *types.DependencyUpdate) {
	// Look for security information
	if strings.Contains(strings.ToLower(body), "security") {
		update.UpdateType = types.UpdateTypeSecurity
	}

	// Look for CVE information
	cvePattern := `CVE-\d{4}-\d+`
	if cves := ga.extractAllMatches(body, cvePattern); len(cves) > 0 {
		update.CVEFixed = cves
		update.Severity = types.DependencySeverityHigh // Assume high for any CVE
	}

	// Extract changelog or release notes
	if changelogStart := strings.Index(body, "Release notes"); changelogStart != -1 {
		changelog := body[changelogStart:]
		if len(changelog) > 1000 {
			changelog = changelog[:1000] + "..."
		}
		update.Changelog = changelog
	}
}

// determineEcosystem determines the package ecosystem
func (ga *GitHubAutomation) determineEcosystem(repoName, packageName string) types.DependencyEcosystem {
	// Check for ecosystem indicators in repository
	if strings.Contains(repoName, "node") || strings.Contains(repoName, "js") || strings.Contains(repoName, "react") {
		return types.EcosystemNPM
	}
	if strings.Contains(repoName, "python") || strings.Contains(repoName, "django") {
		return types.EcosystemPython
	}
	if strings.Contains(repoName, "go") || strings.Contains(repoName, "golang") {
		return types.EcosystemGo
	}
	if strings.Contains(repoName, "rust") || strings.Contains(repoName, "cargo") {
		return types.EcosystemRust
	}

	// Check package name patterns
	if strings.Contains(packageName, "/") && !strings.Contains(packageName, "@") {
		return types.EcosystemGo
	}
	if strings.HasPrefix(packageName, "@") {
		return types.EcosystemNPM
	}

	// Default to npm (most common)
	return types.EcosystemNPM
}

// determineUpdateType determines the semantic version update type
func (ga *GitHubAutomation) determineUpdateType(current, new string) types.DependencyUpdateType {
	// Simple semantic version parsing
	currentParts := strings.Split(strings.TrimPrefix(current, "v"), ".")
	newParts := strings.Split(strings.TrimPrefix(new, "v"), ".")

	if len(currentParts) >= 1 && len(newParts) >= 1 && currentParts[0] != newParts[0] {
		return types.UpdateTypeMajor
	}
	if len(currentParts) >= 2 && len(newParts) >= 2 && currentParts[1] != newParts[1] {
		return types.UpdateTypeMinor
	}
	return types.UpdateTypePatch
}

// determineAction determines what action to take based on analysis
func (ga *GitHubAutomation) determineAction(analysis *types.DependencyAnalysis, update *types.DependencyUpdate) types.PRAction {
	switch analysis.Recommendation {
	case types.RecommendApprove:
		// High confidence updates can be auto-merged
		if analysis.Confidence >= 0.9 && !analysis.BreakingChanges {
			return types.ActionMerge
		}
		return types.ActionApprove

	case types.RecommendReview:
		return types.ActionComment

	case types.RecommendReject:
		return types.ActionReject

	case types.RecommendDelay:
		return types.ActionComment

	default:
		return types.ActionComment
	}
}

// executeAction executes the determined action on the GitHub PR
func (ga *GitHubAutomation) executeAction(ctx context.Context, webhook *types.GitHubDependabotWebhook, action types.PRAction, analysis *types.DependencyAnalysis) (*types.PRAutomationResult, error) {
	result := &types.PRAutomationResult{
		PRID:       fmt.Sprintf("pr-%d", webhook.PullRequest.ID),
		Action:     action,
		Reasoning:  analysis.Reasoning,
		Confidence: analysis.Confidence,
		ExecutedAt: time.Now(),
		ExecutedBy: "liberation-guardian",
		TrustLevel: ga.analyzer.depConfig.TrustLevel,
		Analysis:   analysis,
	}

	switch action {
	case types.ActionApprove:
		err := ga.approvePR(ctx, webhook)
		if err != nil {
			result.Reasoning += fmt.Sprintf(" (Approval failed: %v)", err)
		}

	case types.ActionMerge:
		err := ga.mergePR(ctx, webhook)
		if err != nil {
			result.Reasoning += fmt.Sprintf(" (Merge failed: %v)", err)
			// Fall back to approval
			result.Action = types.ActionApprove
			ga.approvePR(ctx, webhook)
		}

	case types.ActionComment:
		err := ga.commentOnPR(ctx, webhook, ga.generateAnalysisComment(analysis))
		if err != nil {
			result.Reasoning += fmt.Sprintf(" (Comment failed: %v)", err)
		}

	case types.ActionReject:
		err := ga.commentOnPR(ctx, webhook, ga.generateRejectionComment(analysis))
		if err != nil {
			result.Reasoning += fmt.Sprintf(" (Rejection comment failed: %v)", err)
		}

	case types.ActionEscalate:
		err := ga.escalatePR(ctx, webhook, analysis)
		if err != nil {
			result.Reasoning += fmt.Sprintf(" (Escalation failed: %v)", err)
		}
	}

	return result, nil
}

// approvePR approves the GitHub PR
func (ga *GitHubAutomation) approvePR(ctx context.Context, webhook *types.GitHubDependabotWebhook) error {
	if ga.githubToken == "" {
		return fmt.Errorf("GitHub token not configured")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d/reviews",
		webhook.Repository.FullName, webhook.PullRequest.Number)

	reviewBody := map[string]interface{}{
		"event": "APPROVE",
		"body":  "🤖 Liberation Guardian: Dependency update approved after AI analysis",
	}

	return ga.makeGitHubAPICall(ctx, "POST", url, reviewBody)
}

// mergePR merges the GitHub PR
func (ga *GitHubAutomation) mergePR(ctx context.Context, webhook *types.GitHubDependabotWebhook) error {
	if ga.githubToken == "" {
		return fmt.Errorf("GitHub token not configured")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d/merge",
		webhook.Repository.FullName, webhook.PullRequest.Number)

	mergeBody := map[string]interface{}{
		"commit_title":   fmt.Sprintf("Auto-merge: %s", webhook.PullRequest.Title),
		"commit_message": "Automatically merged by Liberation Guardian after AI security analysis",
		"merge_method":   "squash",
	}

	return ga.makeGitHubAPICall(ctx, "PUT", url, mergeBody)
}

// commentOnPR adds a comment to the GitHub PR
func (ga *GitHubAutomation) commentOnPR(ctx context.Context, webhook *types.GitHubDependabotWebhook, comment string) error {
	if ga.githubToken == "" {
		return fmt.Errorf("GitHub token not configured")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments",
		webhook.Repository.FullName, webhook.PullRequest.Number)

	commentBody := map[string]interface{}{
		"body": comment,
	}

	return ga.makeGitHubAPICall(ctx, "POST", url, commentBody)
}

// escalatePR escalates the PR to human reviewers
func (ga *GitHubAutomation) escalatePR(ctx context.Context, webhook *types.GitHubDependabotWebhook, analysis *types.DependencyAnalysis) error {
	escalationComment := ga.generateEscalationComment(analysis)
	return ga.commentOnPR(ctx, webhook, escalationComment)
}

// makeGitHubAPICall makes an authenticated API call to GitHub
func (ga *GitHubAutomation) makeGitHubAPICall(ctx context.Context, method, url string, body interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+ga.githubToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "liberation-guardian/1.0")

	resp, err := ga.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make API call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// generateAnalysisComment creates a comment with AI analysis results
func (ga *GitHubAutomation) generateAnalysisComment(analysis *types.DependencyAnalysis) string {
	return fmt.Sprintf(`## 🤖 Liberation Guardian Analysis

**AI Recommendation:** %s
**Confidence:** %.1f%%
**Security Impact:** %s
**Breaking Changes:** %t

**Analysis:**
%s

**Risk Factors:**
%s

---
*Analyzed by Liberation Guardian AI (Trust Level: %d) • Cost: $%.4f*`,
		analysis.Recommendation,
		analysis.Confidence*100,
		analysis.SecurityImpact,
		analysis.BreakingChanges,
		analysis.Reasoning,
		strings.Join(analysis.RiskFactors, ", "),
		ga.analyzer.depConfig.TrustLevel,
		analysis.Cost,
	)
}

// generateRejectionComment creates a comment explaining why the update was rejected
func (ga *GitHubAutomation) generateRejectionComment(analysis *types.DependencyAnalysis) string {
	return fmt.Sprintf(`## ⚠️ Liberation Guardian: Update Not Recommended

**Recommendation:** %s
**Confidence:** %.1f%%

**Concerns:**
%s

**Risk Factors:**
%s

**Suggested Actions:**
1. Review the security implications manually
2. Consider the breaking changes impact
3. Test thoroughly in a staging environment
4. Update trust level configuration if needed

---
*This analysis was performed by Liberation Guardian AI*`,
		analysis.Recommendation,
		analysis.Confidence*100,
		analysis.Reasoning,
		strings.Join(analysis.RiskFactors, ", "),
	)
}

// generateEscalationComment creates an escalation comment for human review
func (ga *GitHubAutomation) generateEscalationComment(analysis *types.DependencyAnalysis) string {
	return fmt.Sprintf(`## 🚨 Liberation Guardian: Human Review Required

This dependency update requires human review due to:

**Risk Factors:**
%s

**AI Analysis:**
%s

**Recommendation:** Manual review required before proceeding

**Next Steps:**
1. Security team should review the update
2. Test in staging environment
3. Consider rollback plan if proceeding
4. Update automation rules if this type of update should be handled differently

---
*Escalated by Liberation Guardian AI • Trust Level: %d*`,
		strings.Join(analysis.RiskFactors, "\n- "),
		analysis.Reasoning,
		ga.analyzer.depConfig.TrustLevel,
	)
}

// logAutomationResult logs the automation result for audit purposes
func (ga *GitHubAutomation) logAutomationResult(result *types.PRAutomationResult) {
	ga.logger.WithFields(map[string]interface{}{
		"pr_id":       result.PRID,
		"action":      result.Action,
		"confidence":  result.Confidence,
		"trust_level": result.TrustLevel,
		"cost":        result.Analysis.Cost,
	}).Info("PR automation completed")
}

// Helper methods

func (ga *GitHubAutomation) extractMatches(text, pattern string) []string {
	// Simple regex matching - in production would use proper regex
	// For now, just handle common Dependabot patterns
	if strings.Contains(text, "Bump") && strings.Contains(text, "from") && strings.Contains(text, "to") {
		parts := strings.Fields(text)
		if len(parts) >= 6 {
			return []string{text, parts[1], parts[3], parts[5]}
		}
	}
	return nil
}

func (ga *GitHubAutomation) extractAllMatches(text, pattern string) []string {
	// Simple CVE extraction
	var matches []string
	if strings.Contains(text, "CVE-") {
		// Would use proper regex in production
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			if strings.Contains(line, "CVE-") {
				// Extract CVE IDs
				if start := strings.Index(line, "CVE-"); start != -1 {
					end := start + 13 // CVE-YYYY-NNNN format
					if end <= len(line) {
						matches = append(matches, line[start:end])
					}
				}
			}
		}
	}
	return matches
}
