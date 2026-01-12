package dependencies

import (
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"liberation-guardian/pkg/types"
)

// SnykParser parses Snyk PR information
type SnykParser struct {
	logger *logrus.Logger
}

// NewSnykParser creates a new Snyk parser
func NewSnykParser(logger *logrus.Logger) *SnykParser {
	return &SnykParser{
		logger: logger,
	}
}

// ParseSnykPR parses a Snyk pull request to extract dependency information
func (sp *SnykParser) ParseSnykPR(prTitle, prBody string) (*types.DependencyUpdate, error) {
	sp.logger.Debugf("Parsing Snyk PR: %s", prTitle)

	update := &types.DependencyUpdate{
		ID:         generateUpdateID(prTitle),
		Source:     "snyk",
		UpdateType: types.UpdateTypePatch, // Default, will be determined
	}

	// Parse package name and versions from title
	// Snyk titles are typically: "fix: upgrade {package} from {old} to {new}"
	// or "Snyk: {package} from {old} to {new}"
	sp.parseVersionsFromTitle(prTitle, update)

	// Parse vulnerability information from body
	sp.parseVulnerabilitiesFromBody(prBody, update)

	// Determine update type based on version change
	update.UpdateType = determineUpdateType(update.CurrentVersion, update.NewVersion)

	// Determine severity from CVE information
	update.Severity = sp.determineSeverity(prBody)

	sp.logger.Debugf("Parsed Snyk update: %s %s → %s", update.PackageName, update.CurrentVersion, update.NewVersion)

	return update, nil
}

// parseVersionsFromTitle extracts package name and versions from PR title
func (sp *SnykParser) parseVersionsFromTitle(title string, update *types.DependencyUpdate) {
	// Pattern 1: "fix: upgrade package from 1.0.0 to 1.0.1"
	upgradePattern := regexp.MustCompile(`(?i)upgrade\s+([^\s]+)\s+from\s+([^\s]+)\s+to\s+([^\s]+)`)
	matches := upgradePattern.FindStringSubmatch(title)
	if len(matches) == 4 {
		update.PackageName = matches[1]
		update.CurrentVersion = matches[2]
		update.NewVersion = matches[3]
		return
	}

	// Pattern 2: "Snyk: package from 1.0.0 to 1.0.1"
	snykPattern := regexp.MustCompile(`(?i)snyk:\s+([^\s]+)\s+from\s+([^\s]+)\s+to\s+([^\s]+)`)
	matches = snykPattern.FindStringSubmatch(title)
	if len(matches) == 4 {
		update.PackageName = matches[1]
		update.CurrentVersion = matches[2]
		update.NewVersion = matches[3]
		return
	}

	// Pattern 3: "[Snyk] Security upgrade package from 1.0.0 to 1.0.1"
	securityPattern := regexp.MustCompile(`(?i)\[snyk\].*?([^\s]+)\s+from\s+([^\s]+)\s+to\s+([^\s]+)`)
	matches = securityPattern.FindStringSubmatch(title)
	if len(matches) == 4 {
		update.PackageName = matches[1]
		update.CurrentVersion = matches[2]
		update.NewVersion = matches[3]
		return
	}

	// Fallback: try to extract just package name
	packagePattern := regexp.MustCompile(`(?i)(?:upgrade|update)\s+([^\s]+)`)
	matches = packagePattern.FindStringSubmatch(title)
	if len(matches) >= 2 {
		update.PackageName = matches[1]
	}
}

// parseVulnerabilitiesFromBody extracts CVE and vulnerability information from PR body
func (sp *SnykParser) parseVulnerabilitiesFromBody(body string, update *types.DependencyUpdate) {
	// Look for CVE identifiers
	cvePattern := regexp.MustCompile(`CVE-\d{4}-\d{4,7}`)
	cveMatches := cvePattern.FindAllString(body, -1)
	if len(cveMatches) > 0 {
		update.CVEFixed = cveMatches
	}

	// Look for SNYK IDs
	snykIDPattern := regexp.MustCompile(`SNYK-[A-Z]+-[A-Z]+-\d+`)
	snykMatches := snykIDPattern.FindAllString(body, -1)
	if len(snykMatches) > 0 {
		if update.VulnerabilityInfo == nil {
			update.VulnerabilityInfo = make(map[string]interface{})
		}
		update.VulnerabilityInfo["snyk_ids"] = snykMatches
	}

	// Extract severity information
	severityPattern := regexp.MustCompile(`(?i)severity:\s*([^\n]+)`)
	severityMatches := severityPattern.FindStringSubmatch(body)
	if len(severityMatches) >= 2 {
		if update.VulnerabilityInfo == nil {
			update.VulnerabilityInfo = make(map[string]interface{})
		}
		update.VulnerabilityInfo["severity"] = strings.TrimSpace(severityMatches[1])
	}

	// Extract CVSS score if present
	cvssPattern := regexp.MustCompile(`(?i)CVSS\s*(?:score)?:?\s*([0-9.]+)`)
	cvssMatches := cvssPattern.FindStringSubmatch(body)
	if len(cvssMatches) >= 2 {
		if update.VulnerabilityInfo == nil {
			update.VulnerabilityInfo = make(map[string]interface{})
		}
		update.VulnerabilityInfo["cvss_score"] = cvssMatches[1]
	}
}

// determineSeverity determines the severity based on PR body content
func (sp *SnykParser) determineSeverity(body string) types.Severity {
	bodyLower := strings.ToLower(body)

	// Check for severity keywords
	if strings.Contains(bodyLower, "critical") {
		return types.SeverityCritical
	}
	if strings.Contains(bodyLower, "high severity") {
		return types.SeverityHigh
	}
	if strings.Contains(bodyLower, "medium severity") {
		return types.SeverityMedium
	}
	if strings.Contains(bodyLower, "low severity") {
		return types.SeverityLow
	}

	// Check for CVE presence (security fix)
	if strings.Contains(bodyLower, "cve-") || strings.Contains(bodyLower, "security") {
		return types.SeverityHigh // Default to high for security fixes
	}

	return types.SeverityMedium // Default
}

// IsSnykSecurityFix determines if this is a security-related fix
func (sp *SnykParser) IsSnykSecurityFix(prTitle, prBody string) bool {
	titleLower := strings.ToLower(prTitle)
	bodyLower := strings.ToLower(prBody)

	securityIndicators := []string{
		"security",
		"vulnerability",
		"cve-",
		"snyk-",
		"exploit",
	}

	for _, indicator := range securityIndicators {
		if strings.Contains(titleLower, indicator) || strings.Contains(bodyLower, indicator) {
			return true
		}
	}

	return false
}
