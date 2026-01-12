package dependencies

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// BotType represents the type of dependency bot
type BotType string

const (
	BotTypeDependabot BotType = "dependabot"
	BotTypeSnyk       BotType = "snyk"
	BotTypeUnknown    BotType = "unknown"
)

// BotDetector detects and classifies dependency bots
type BotDetector struct {
	logger *logrus.Logger
}

// NewBotDetector creates a new bot detector
func NewBotDetector(logger *logrus.Logger) *BotDetector {
	return &BotDetector{
		logger: logger,
	}
}

// DetectBotType detects the bot type from PR metadata
func (bd *BotDetector) DetectBotType(username, userType, prTitle, prBody string) BotType {
	// Check username patterns
	if bd.isDependabotUsername(username) {
		return BotTypeDependabot
	}

	if bd.isSnykUsername(username) {
		return BotTypeSnyk
	}

	// Check user type (must be Bot)
	if userType != "Bot" {
		return BotTypeUnknown
	}

	// Check PR title/body patterns as fallback
	if bd.isSnykPattern(prTitle, prBody) {
		return BotTypeSnyk
	}

	if bd.isDependabotPattern(prTitle, prBody) {
		return BotTypeDependabot
	}

	return BotTypeUnknown
}

// IsDependencyBot returns true if the username/type indicates a dependency bot
func (bd *BotDetector) IsDependencyBot(username, userType string) bool {
	return bd.isDependabotUsername(username) ||
		bd.isSnykUsername(username) ||
		(userType == "Bot" && (strings.Contains(strings.ToLower(username), "dependabot") || strings.Contains(strings.ToLower(username), "snyk")))
}

// isDependabotUsername checks if username matches Dependabot patterns
func (bd *BotDetector) isDependabotUsername(username string) bool {
	dependabotPatterns := []string{
		"dependabot[bot]",
		"dependabot-preview[bot]",
		"dependabot",
	}

	usernameLower := strings.ToLower(username)
	for _, pattern := range dependabotPatterns {
		if strings.Contains(usernameLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// isSnykUsername checks if username matches Snyk patterns
func (bd *BotDetector) isSnykUsername(username string) bool {
	snykPatterns := []string{
		"snyk-bot",
		"snyk[bot]",
		"snyk-io",
	}

	usernameLower := strings.ToLower(username)
	for _, pattern := range snykPatterns {
		if strings.Contains(usernameLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// isDependabotPattern checks if PR title/body matches Dependabot patterns
func (bd *BotDetector) isDependabotPattern(title, body string) bool {
	titleLower := strings.ToLower(title)
	bodyLower := strings.ToLower(body)

	dependabotIndicators := []string{
		"bump ",
		"update ",
		"dependabot",
	}

	for _, indicator := range dependabotIndicators {
		if strings.Contains(titleLower, indicator) || strings.Contains(bodyLower, indicator) {
			return true
		}
	}

	return false
}

// isSnykPattern checks if PR title/body matches Snyk patterns
func (bd *BotDetector) isSnykPattern(title, body string) bool {
	titleLower := strings.ToLower(title)
	bodyLower := strings.ToLower(body)

	snykIndicators := []string{
		"snyk has created",
		"snyk:",
		"fix: upgrade",
		"fix: security",
		"[snyk]",
		"snyk.io",
	}

	for _, indicator := range snykIndicators {
		if strings.Contains(titleLower, indicator) || strings.Contains(bodyLower, indicator) {
			return true
		}
	}

	return false
}

// GetBotDisplayName returns a human-readable bot name
func (bd *BotDetector) GetBotDisplayName(botType BotType) string {
	switch botType {
	case BotTypeDependabot:
		return "Dependabot"
	case BotTypeSnyk:
		return "Snyk"
	default:
		return "Unknown Bot"
	}
}
