package dependencies

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/ai"
	"liberation-guardian/internal/config"
	"liberation-guardian/pkg/types"
)

// DependencyAnalyzer provides AI-powered dependency update analysis
type DependencyAnalyzer struct {
	config    *config.Config
	logger    *logrus.Logger
	aiClient  ai.AIClient
	depConfig *types.DependencyConfig
}

// NewDependencyAnalyzer creates a new dependency analyzer
func NewDependencyAnalyzer(cfg *config.Config, logger *logrus.Logger, aiClient ai.AIClient) *DependencyAnalyzer {
	// Load dependency configuration with defaults
	depConfig := loadDependencyConfig(cfg)

	return &DependencyAnalyzer{
		config:    cfg,
		logger:    logger,
		aiClient:  aiClient,
		depConfig: depConfig,
	}
}

// AnalyzeDependencyUpdate performs comprehensive AI analysis of a dependency update
func (da *DependencyAnalyzer) AnalyzeDependencyUpdate(ctx context.Context, update *types.DependencyUpdate) (*types.DependencyAnalysis, error) {
	startTime := time.Now()
	da.logger.Infof("Analyzing dependency update: %s %s → %s", update.PackageName, update.CurrentVersion, update.NewVersion)

	// Step 1: Basic risk assessment
	riskFactors := da.identifyRiskFactors(update)

	// Step 2: Community metrics analysis
	communityMetrics := da.analyzeCommunityMetrics(ctx, update)

	// Step 2.5: Check if fast-path can be used (skip expensive AI analysis)
	var aiAnalysis *aiAnalysisResult
	var err error
	fastPathEligible := false
	fastPathUsed := false

	if da.shouldUseFastPath(update) {
		fastPathEligible = true
		da.logger.Infof("Using fast-path for %s (skipping AI analysis)", update.PackageName)
		aiAnalysis = da.fastPathAnalysis(update, riskFactors)
		fastPathUsed = true
	} else {
		// Step 3: AI-powered analysis (expensive)
		aiAnalysis, err = da.performAIAnalysis(ctx, update, riskFactors, communityMetrics)
		if err != nil {
			da.logger.Errorf("AI analysis failed for %s: %v", update.PackageName, err)
			// Fall back to rule-based analysis
			aiAnalysis = da.fallbackAnalysis(update, riskFactors)
		}
	}

	// Step 4: Apply trust level and custom rules
	recommendation := da.applyTrustLevelRules(aiAnalysis, update)

	// Step 5: Generate auto-fix suggestions if applicable
	autoFix := da.generateAutoFixSuggestion(ctx, update, aiAnalysis)

	analysis := &types.DependencyAnalysis{
		UpdateID:          update.ID,
		SecurityImpact:    aiAnalysis.SecurityImpact,
		BreakingChanges:   aiAnalysis.BreakingChanges,
		Confidence:        aiAnalysis.Confidence,
		RiskFactors:       riskFactors,
		Recommendation:    recommendation,
		Reasoning:         aiAnalysis.Reasoning,
		AutoFixSuggestion: autoFix,
		TestCompatibility: aiAnalysis.TestCompatibility,
		CommunityAdoption: communityMetrics,
		ProcessingTime:    time.Since(startTime).Milliseconds(),
		AIProvider:        aiAnalysis.AIProvider,
		Cost:              aiAnalysis.Cost,
		FastPathEligible:  fastPathEligible,
		FastPathUsed:      fastPathUsed,
	}

	da.logger.Infof("Analysis complete for %s: %s (confidence: %.2f, fast-path: %v)",
		update.PackageName, recommendation, analysis.Confidence, fastPathUsed)

	return analysis, nil
}

// identifyRiskFactors identifies risk factors based on update characteristics
func (da *DependencyAnalyzer) identifyRiskFactors(update *types.DependencyUpdate) []string {
	var risks []string

	// Version jump analysis
	if update.UpdateType == types.UpdateTypeMajor {
		risks = append(risks, "major_version_update")
	}

	// Security update analysis
	if len(update.CVEFixed) > 0 {
		risks = append(risks, "security_vulnerabilities_fixed")
		for _, cve := range update.CVEFixed {
			if strings.Contains(cve, "CRITICAL") {
				risks = append(risks, "critical_security_fix")
				break
			}
		}
	}

	// Package name analysis
	if da.isPopularPackage(update.PackageName, update.Ecosystem) {
		risks = append(risks, "popular_package") // Actually reduces risk
	} else {
		risks = append(risks, "low_adoption_package")
	}

	// Ecosystem-specific risks
	switch update.Ecosystem {
	case types.EcosystemNPM:
		if da.isNPMRiskyUpdate(update) {
			risks = append(risks, "npm_dependency_confusion_risk")
		}
	case types.EcosystemPython:
		if da.isPythonRiskyUpdate(update) {
			risks = append(risks, "python_package_risk")
		}
	}

	// Time-based risks
	if da.isRecentPackage(update.PackageName, update.NewVersion) {
		risks = append(risks, "very_recent_release")
	}

	return risks
}

// analyzeCommunityMetrics gathers community adoption metrics
func (da *DependencyAnalyzer) analyzeCommunityMetrics(ctx context.Context, update *types.DependencyUpdate) types.CommunityMetrics {
	// This would integrate with package registry APIs
	// For now, return estimated metrics based on ecosystem and package name
	metrics := types.CommunityMetrics{
		WeeklyDownloads:    da.estimateDownloads(update.PackageName, update.Ecosystem),
		GithubStars:        da.estimateStars(update.PackageName),
		OpenIssues:         da.estimateIssues(update.PackageName),
		LastUpdateDays:     da.estimateLastUpdate(update.NewVersion),
		MaintainerActivity: 0.8,  // Default assumption
		TestCoverage:       0.75, // Default assumption
	}

	// Adjust metrics based on package popularity
	if da.isPopularPackage(update.PackageName, update.Ecosystem) {
		metrics.WeeklyDownloads *= 10
		metrics.GithubStars *= 5
		metrics.MaintainerActivity = 0.9
		metrics.TestCoverage = 0.85
	}

	return metrics
}

// performAIAnalysis uses AI to analyze the dependency update
func (da *DependencyAnalyzer) performAIAnalysis(ctx context.Context, update *types.DependencyUpdate, riskFactors []string, metrics types.CommunityMetrics) (*aiAnalysisResult, error) {
	prompt := da.buildAIPrompt(update, riskFactors, metrics)

	aiRequest := &types.AIRequest{
		Agent:        types.AgentAnalysis,
		Prompt:       prompt,
		SystemPrompt: da.getSecurityAnalysisSystemPrompt(),
		MaxTokens:    2000,
		Temperature:  0.1, // Low temperature for consistent analysis
		Metadata: map[string]interface{}{
			"update_type": update.UpdateType,
			"ecosystem":   update.Ecosystem,
			"severity":    update.Severity,
		},
	}

	response, err := da.aiClient.SendRequest(ctx, aiRequest)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	// Parse AI response
	var analysis aiAnalysisResult
	if err := json.Unmarshal([]byte(response.Content), &analysis); err != nil {
		da.logger.Warnf("Failed to parse AI response, using fallback: %v", err)
		return da.parseUnstructuredAIResponse(response.Content, update), nil
	}

	analysis.AIProvider = response.Provider
	analysis.Cost = response.Cost

	return &analysis, nil
}

// buildAIPrompt creates a comprehensive prompt for AI analysis
func (da *DependencyAnalyzer) buildAIPrompt(update *types.DependencyUpdate, riskFactors []string, metrics types.CommunityMetrics) string {
	return fmt.Sprintf(`Analyze this dependency update for security and compatibility:

Package: %s
Ecosystem: %s
Current Version: %s
New Version: %s
Update Type: %s
Security Fixes: %v
Risk Factors: %v

Community Metrics:
- Weekly Downloads: %d
- GitHub Stars: %d
- Open Issues: %d
- Test Coverage: %.2f
- Maintainer Activity: %.2f

Changelog Summary:
%s

Provide analysis in this JSON format:
{
  "security_impact": "info|low|moderate|high|critical",
  "breaking_changes": boolean,
  "confidence": 0.0-1.0,
  "reasoning": "detailed explanation",
  "test_compatibility": 0.0-1.0,
  "migration_complexity": "simple|moderate|complex"
}

Focus on:
1. Security implications of the update
2. Likelihood of breaking changes
3. Community adoption and stability
4. Risk vs benefit analysis`,
		update.PackageName,
		update.Ecosystem,
		update.CurrentVersion,
		update.NewVersion,
		update.UpdateType,
		update.CVEFixed,
		riskFactors,
		metrics.WeeklyDownloads,
		metrics.GithubStars,
		metrics.OpenIssues,
		metrics.TestCoverage,
		metrics.MaintainerActivity,
		da.truncateChangelog(update.Changelog, 500),
	)
}

// getSecurityAnalysisSystemPrompt returns the system prompt for security analysis
func (da *DependencyAnalyzer) getSecurityAnalysisSystemPrompt() string {
	return `You are a security-focused dependency analyst with expertise in:
- Software supply chain security
- Semantic versioning and compatibility analysis
- Package ecosystem best practices
- Risk assessment for automated dependency updates

Your analysis should be:
- Conservative for security updates (favor applying them)
- Careful with breaking changes (high confidence required)
- Practical for development teams (balance security vs velocity)
- Cost-aware (minimize expensive manual reviews)

Provide structured, actionable analysis that helps teams make informed decisions about dependency updates.`
}

// applyTrustLevelRules applies user-configured trust level rules
func (da *DependencyAnalyzer) applyTrustLevelRules(aiAnalysis *aiAnalysisResult, update *types.DependencyUpdate) types.DependencyRecommendation {
	// Check custom rules first
	if customRec := da.checkCustomRules(update); customRec != "" {
		return customRec
	}

	// Apply trust level logic
	switch da.depConfig.TrustLevel {
	case types.TrustParanoid:
		return types.RecommendReview // Always require human review

	case types.TrustConservative:
		if update.UpdateType == types.UpdateTypePatch || update.UpdateType == types.UpdateTypeSecurity {
			if aiAnalysis.Confidence > 0.8 && !aiAnalysis.BreakingChanges {
				return types.RecommendApprove
			}
		}
		return types.RecommendReview

	case types.TrustBalanced: // RECOMMENDED
		if update.UpdateType == types.UpdateTypeSecurity {
			if aiAnalysis.Confidence > 0.75 {
				return types.RecommendApprove
			}
		}
		if update.UpdateType == types.UpdateTypePatch ||
			(update.UpdateType == types.UpdateTypeMinor && aiAnalysis.Confidence > 0.85) {
			if !aiAnalysis.BreakingChanges {
				return types.RecommendApprove
			}
		}
		return types.RecommendReview

	case types.TrustProgressive:
		if aiAnalysis.Confidence > 0.7 && !aiAnalysis.BreakingChanges {
			return types.RecommendApprove
		}
		if update.UpdateType == types.UpdateTypeMajor && aiAnalysis.Confidence < 0.9 {
			return types.RecommendReview
		}
		return types.RecommendApprove

	case types.TrustAutonomous:
		if aiAnalysis.Confidence > 0.6 {
			return types.RecommendApprove
		}
		return types.RecommendReview

	default:
		return types.RecommendReview
	}
}

// checkCustomRules applies user-defined custom rules
func (da *DependencyAnalyzer) checkCustomRules(update *types.DependencyUpdate) types.DependencyRecommendation {
	for _, rule := range da.depConfig.CustomRules {
		if da.matchesRule(update, rule) {
			da.logger.Infof("Custom rule '%s' matched for %s", rule.Name, update.PackageName)
			return rule.Action
		}
	}
	return ""
}

// matchesRule checks if an update matches a custom rule
func (da *DependencyAnalyzer) matchesRule(update *types.DependencyUpdate, rule types.DependencyRule) bool {
	// Check package name pattern
	if rule.Pattern != "" {
		matched, err := regexp.MatchString(rule.Pattern, update.PackageName)
		if err != nil || !matched {
			return false
		}
	}

	// Check update type
	if rule.UpdateType != "" && rule.UpdateType != update.UpdateType {
		return false
	}

	// Check additional conditions
	for key, value := range rule.Conditions {
		if !da.evaluateCondition(update, key, value) {
			return false
		}
	}

	return true
}

// evaluateCondition evaluates a single rule condition
func (da *DependencyAnalyzer) evaluateCondition(update *types.DependencyUpdate, key string, value interface{}) bool {
	switch key {
	case "ecosystem":
		return string(update.Ecosystem) == value.(string)
	case "min_severity":
		// Implementation would compare severity levels
		return true
	case "has_cve":
		return len(update.CVEFixed) > 0
	default:
		return true
	}
}

// generateAutoFixSuggestion generates automated fix suggestions
func (da *DependencyAnalyzer) generateAutoFixSuggestion(ctx context.Context, update *types.DependencyUpdate, analysis *aiAnalysisResult) *types.AutoFixPlan {
	if analysis.BreakingChanges || analysis.Confidence < 0.8 {
		return nil // No auto-fix for risky updates
	}

	steps := []types.FixStep{
		{
			Action: "update_dependency",
			Target: update.PackageName,
			Parameters: map[string]string{
				"from_version": update.CurrentVersion,
				"to_version":   update.NewVersion,
				"ecosystem":    string(update.Ecosystem),
			},
			Validation: "run_tests",
			OnFailure:  "rollback",
		},
	}

	if da.depConfig.RequiredTests {
		steps = append(steps, types.FixStep{
			Action: "run_tests",
			Target: "test_suite",
			Parameters: map[string]string{
				"min_coverage": strconv.FormatFloat(da.depConfig.MinTestCoverage, 'f', 2, 64),
			},
			Validation: "coverage_check",
			OnFailure:  "rollback",
		})
	}

	rollbackSteps := []types.FixStep{
		{
			Action: "revert_dependency",
			Target: update.PackageName,
			Parameters: map[string]string{
				"to_version": update.CurrentVersion,
			},
		},
	}

	return &types.AutoFixPlan{
		Type:             types.FixTypeDependencyUpdate,
		Description:      fmt.Sprintf("Update %s from %s to %s", update.PackageName, update.CurrentVersion, update.NewVersion),
		Steps:            steps,
		EstimatedTime:    5, // 5 minutes
		RequiresApproval: da.depConfig.TrustLevel < types.TrustProgressive,
		RollbackPlan:     rollbackSteps,
	}
}

// Helper methods for risk assessment

func (da *DependencyAnalyzer) isPopularPackage(name string, ecosystem types.DependencyEcosystem) bool {
	popularPackages := map[types.DependencyEcosystem][]string{
		types.EcosystemNPM:    {"lodash", "express", "react", "vue", "axios", "moment"},
		types.EcosystemPython: {"requests", "numpy", "pandas", "django", "flask", "pytest"},
		types.EcosystemGo:     {"gin-gonic/gin", "gorilla/mux", "sirupsen/logrus"},
		types.EcosystemRust:   {"serde", "tokio", "clap", "reqwest"},
	}

	packages, exists := popularPackages[ecosystem]
	if !exists {
		return false
	}

	for _, pkg := range packages {
		if strings.Contains(name, pkg) {
			return true
		}
	}
	return false
}

func (da *DependencyAnalyzer) isNPMRiskyUpdate(update *types.DependencyUpdate) bool {
	// Check for potential dependency confusion attacks
	return strings.Contains(update.PackageName, "-") &&
		len(strings.Split(update.PackageName, "-")) > 3
}

func (da *DependencyAnalyzer) isPythonRiskyUpdate(update *types.DependencyUpdate) bool {
	// Check for typosquatting risks
	commonPkgs := []string{"requests", "urllib3", "numpy", "pandas"}
	for _, common := range commonPkgs {
		if da.isTyposquattingRisk(update.PackageName, common) {
			return true
		}
	}
	return false
}

func (da *DependencyAnalyzer) isTyposquattingRisk(pkg, common string) bool {
	// Simple Levenshtein distance check
	return len(pkg) == len(common) && da.levenshteinDistance(pkg, common) == 1
}

func (da *DependencyAnalyzer) levenshteinDistance(s1, s2 string) int {
	// Simple implementation for basic typosquatting detection
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}
	if s1[0] == s2[0] {
		return da.levenshteinDistance(s1[1:], s2[1:])
	}
	return 1 + da.min(
		da.levenshteinDistance(s1[1:], s2),
		da.min(
			da.levenshteinDistance(s1, s2[1:]),
			da.levenshteinDistance(s1[1:], s2[1:]),
		),
	)
}

func (da *DependencyAnalyzer) min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (da *DependencyAnalyzer) isRecentPackage(name, version string) bool {
	// This would check package registry for release date
	// For now, return false (assume not too recent)
	return false
}

func (da *DependencyAnalyzer) estimateDownloads(name string, ecosystem types.DependencyEcosystem) int {
	if da.isPopularPackage(name, ecosystem) {
		return 1000000 // 1M+ weekly downloads
	}
	return 10000 // Default estimate
}

func (da *DependencyAnalyzer) estimateStars(name string) int {
	if strings.Contains(name, "react") || strings.Contains(name, "vue") {
		return 50000
	}
	return 1000
}

func (da *DependencyAnalyzer) estimateIssues(name string) int {
	return 25 // Default estimate
}

func (da *DependencyAnalyzer) estimateLastUpdate(version string) int {
	// Parse version for recency estimation
	return 7 // Default: 7 days ago
}

func (da *DependencyAnalyzer) truncateChangelog(changelog string, maxLen int) string {
	if len(changelog) <= maxLen {
		return changelog
	}
	return changelog[:maxLen] + "..."
}

// loadDependencyConfig loads dependency configuration with defaults
func loadDependencyConfig(cfg *config.Config) *types.DependencyConfig {
	// This would load from config file, for now use sensible defaults
	return &types.DependencyConfig{
		TrustLevel:          types.TrustBalanced, // Recommended default
		SecurityAutoApprove: true,
		PatchAutoApprove:    true,
		MinorAutoApprove:    false,
		MajorAutoApprove:    false,
		RequiredTests:       true,
		MinTestCoverage:     0.70,
		MinConfidence:       0.80,
		ExcludedPackages:    []string{},
		IncludedPackages:    []string{},
		Ecosystems: []types.DependencyEcosystem{
			types.EcosystemNPM,
			types.EcosystemPython,
			types.EcosystemGo,
			types.EcosystemRust,
		},
		CustomRules:   []types.DependencyRule{},
		SupportedBots: []string{"dependabot", "snyk"},
		SimplePRFastPath: types.SimplePRFastPath{
			Enabled:             true,
			PatchOnly:           true,
			PopularPackagesOnly: true,
			MinWeeklyDownloads:  100000,
			MaxDiffLines:        50,
			BlockSecurityFixes:  true,
		},
		Snyk: types.SnykConfig{
			Enabled:            true,
			AutoApprovePatches: true,
			TrustSnykPriority:  true,
		},
	}
}

// aiAnalysisResult represents the structured AI analysis result
type aiAnalysisResult struct {
	SecurityImpact      types.DependencySeverity `json:"security_impact"`
	BreakingChanges     bool                     `json:"breaking_changes"`
	Confidence          float64                  `json:"confidence"`
	Reasoning           string                   `json:"reasoning"`
	TestCompatibility   float64                  `json:"test_compatibility"`
	MigrationComplexity string                   `json:"migration_complexity"`
	AIProvider          string                   `json:"-"`
	Cost                float64                  `json:"-"`
}

// fallbackAnalysis provides rule-based analysis when AI fails
func (da *DependencyAnalyzer) fallbackAnalysis(update *types.DependencyUpdate, riskFactors []string) *aiAnalysisResult {
	confidence := 0.7 // Default confidence for rule-based analysis
	breakingChanges := update.UpdateType == types.UpdateTypeMajor

	securityImpact := types.DependencySeverityLow
	if len(update.CVEFixed) > 0 {
		securityImpact = types.DependencySeverityHigh
		confidence = 0.9 // High confidence for security fixes
	}

	reasoning := fmt.Sprintf("Rule-based analysis: %s update with %d risk factors identified",
		update.UpdateType, len(riskFactors))

	return &aiAnalysisResult{
		SecurityImpact:      securityImpact,
		BreakingChanges:     breakingChanges,
		Confidence:          confidence,
		Reasoning:           reasoning,
		TestCompatibility:   0.8, // Assume good compatibility
		MigrationComplexity: "simple",
		AIProvider:          "fallback",
		Cost:                0.0,
	}
}

// parseUnstructuredAIResponse parses AI response when JSON parsing fails
func (da *DependencyAnalyzer) parseUnstructuredAIResponse(content string, update *types.DependencyUpdate) *aiAnalysisResult {
	analysis := da.fallbackAnalysis(update, []string{})
	analysis.Reasoning = "AI provided unstructured response: " + content[:min(len(content), 200)]
	return analysis
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// shouldUseFastPath determines if fast-path should be used for this update
func (da *DependencyAnalyzer) shouldUseFastPath(update *types.DependencyUpdate) bool {
	// Fast-path must be enabled and respect trust level
	if !da.depConfig.SimplePRFastPath.Enabled {
		return false
	}

	// Trust level 0 (Paranoid) never uses fast-path
	if da.depConfig.TrustLevel == types.TrustParanoid {
		return false
	}

	// Create fast-path config from dependency config
	fastPathConfig := &SimplePRFastPathConfig{
		Enabled:             da.depConfig.SimplePRFastPath.Enabled,
		PatchOnly:           da.depConfig.SimplePRFastPath.PatchOnly,
		PopularPackagesOnly: da.depConfig.SimplePRFastPath.PopularPackagesOnly,
		MinWeeklyDownloads:  da.depConfig.SimplePRFastPath.MinWeeklyDownloads,
		MaxDiffLines:        da.depConfig.SimplePRFastPath.MaxDiffLines,
		BlockSecurityFixes:  da.depConfig.SimplePRFastPath.BlockSecurityFixes,
	}

	// Use SimplePRDetector to determine eligibility
	detector := NewSimplePRDetector(da.logger, fastPathConfig)
	return detector.IsSimplePR(update)
}

// fastPathAnalysis provides a quick rule-based analysis for simple PRs
func (da *DependencyAnalyzer) fastPathAnalysis(update *types.DependencyUpdate, riskFactors []string) *aiAnalysisResult {
	da.logger.Debugf("Performing fast-path analysis for %s", update.PackageName)

	// Fast-path: simple patches of popular packages are low risk
	confidence := 0.95 // High confidence for fast-path eligible updates
	breakingChanges := false
	securityImpact := types.DependencySeverityLow

	// If there are CVEs, still mark as security but trust Snyk/Dependabot assessment
	if len(update.CVEFixed) > 0 {
		securityImpact = types.DependencySeverityModerate
	}

	reasoning := fmt.Sprintf("Fast-path: %s patch update of popular package %s. "+
		"Skipped expensive AI analysis. Update type: %s, Risk factors: %d",
		update.Source, update.PackageName, update.UpdateType, len(riskFactors))

	return &aiAnalysisResult{
		SecurityImpact:      securityImpact,
		BreakingChanges:     breakingChanges,
		Confidence:          confidence,
		Reasoning:           reasoning,
		TestCompatibility:   0.95, // High test compatibility expected
		MigrationComplexity: "trivial",
		AIProvider:          "fast-path",
		Cost:                0.0, // No AI cost
	}
}
