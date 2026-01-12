package types

import (
	"time"
)

// DependencyUpdate represents a dependency update from Dependabot or similar tools
type DependencyUpdate struct {
	ID                string                 `json:"id"`
	Source            string                 `json:"source"` // "dependabot" or "snyk"
	Repository        string                 `json:"repository"`
	PackageName       string                 `json:"package_name"`
	CurrentVersion    string                 `json:"current_version"`
	NewVersion        string                 `json:"new_version"`
	UpdateType        DependencyUpdateType   `json:"update_type"`
	Ecosystem         DependencyEcosystem    `json:"ecosystem"`
	Severity          DependencySeverity     `json:"severity"`
	CVEFixed          []string               `json:"cve_fixed,omitempty"`
	Changelog         string                 `json:"changelog,omitempty"`
	ChangelogURL      string                 `json:"changelog_url,omitempty"`
	ReleaseNotesURL   string                 `json:"release_notes_url,omitempty"`
	PRNumber          int                    `json:"pr_number,omitempty"`
	PRUrl             string                 `json:"pr_url,omitempty"`
	DiffStats         *DiffStats             `json:"diff_stats,omitempty"`
	VulnerabilityInfo map[string]interface{} `json:"vulnerability_info,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// DependencyUpdateType represents the type of dependency update
type DependencyUpdateType string

const (
	UpdateTypePatch    DependencyUpdateType = "patch"    // 1.2.3 → 1.2.4
	UpdateTypeMinor    DependencyUpdateType = "minor"    // 1.2.3 → 1.3.0
	UpdateTypeMajor    DependencyUpdateType = "major"    // 1.2.3 → 2.0.0
	UpdateTypeSecurity DependencyUpdateType = "security" // Security-focused update
)

// DependencyEcosystem represents different package ecosystems
type DependencyEcosystem string

const (
	EcosystemNPM      DependencyEcosystem = "npm"
	EcosystemPython   DependencyEcosystem = "pip"
	EcosystemGo       DependencyEcosystem = "go_modules"
	EcosystemRust     DependencyEcosystem = "cargo"
	EcosystemJava     DependencyEcosystem = "maven"
	EcosystemRuby     DependencyEcosystem = "bundler"
	EcosystemNuGet    DependencyEcosystem = "nuget"
	EcosystemComposer DependencyEcosystem = "composer"
)

// DependencySeverity represents the severity of security issues
type DependencySeverity string

const (
	DependencySeverityInfo     DependencySeverity = "info"
	DependencySeverityLow      DependencySeverity = "low"
	DependencySeverityModerate DependencySeverity = "moderate"
	DependencySeverityHigh     DependencySeverity = "high"
	DependencySeverityCritical DependencySeverity = "critical"
)

// Severity is an alias for backwards compatibility with event severity
type Severity = string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// DiffStats represents the diff statistics for a PR
type DiffStats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Changes   int `json:"changes"`
}

// DependencyAnalysis represents AI analysis of a dependency update
type DependencyAnalysis struct {
	UpdateID          string                   `json:"update_id"`
	SecurityImpact    DependencySeverity       `json:"security_impact"`
	BreakingChanges   bool                     `json:"breaking_changes"`
	Confidence        float64                  `json:"confidence"`
	RiskFactors       []string                 `json:"risk_factors"`
	Recommendation    DependencyRecommendation `json:"recommendation"`
	Reasoning         string                   `json:"reasoning"`
	AutoFixSuggestion *AutoFixPlan             `json:"auto_fix_suggestion,omitempty"`
	TestCompatibility float64                  `json:"test_compatibility"`
	CommunityAdoption CommunityMetrics         `json:"community_adoption"`
	ProcessingTime    int64                    `json:"processing_time_ms"`
	AIProvider        string                   `json:"ai_provider"`
	Cost              float64                  `json:"cost"`
	FastPathEligible  bool                     `json:"fast_path_eligible"` // Was eligible for fast-path
	FastPathUsed      bool                     `json:"fast_path_used"`     // Did use fast-path
}

// DependencyRecommendation represents AI recommendation for handling update
type DependencyRecommendation string

const (
	RecommendApprove  DependencyRecommendation = "approve"
	RecommendReview   DependencyRecommendation = "review"
	RecommendReject   DependencyRecommendation = "reject"
	RecommendDelay    DependencyRecommendation = "delay"
	RecommendRollback DependencyRecommendation = "rollback"
)

// CommunityMetrics represents community adoption metrics for a package
type CommunityMetrics struct {
	WeeklyDownloads    int     `json:"weekly_downloads"`
	GithubStars        int     `json:"github_stars"`
	OpenIssues         int     `json:"open_issues"`
	LastUpdateDays     int     `json:"last_update_days"`
	MaintainerActivity float64 `json:"maintainer_activity"`
	TestCoverage       float64 `json:"test_coverage"`
}

// TrustLevel represents the automation trust level for dependency updates
type TrustLevel int

const (
	TrustParanoid     TrustLevel = 0 // Human approval for ALL updates
	TrustConservative TrustLevel = 1 // Patch + security only
	TrustBalanced     TrustLevel = 2 // Patch + minor security (RECOMMENDED)
	TrustProgressive  TrustLevel = 3 // All security + confident minor
	TrustAutonomous   TrustLevel = 4 // Full automation with analysis
)

// DependencyConfig represents dependency automation configuration
type DependencyConfig struct {
	TrustLevel          TrustLevel            `yaml:"trust_level"`
	SecurityAutoApprove bool                  `yaml:"security_auto_approve"`
	PatchAutoApprove    bool                  `yaml:"patch_auto_approve"`
	MinorAutoApprove    bool                  `yaml:"minor_auto_approve"`
	MajorAutoApprove    bool                  `yaml:"major_auto_approve"`
	RequiredTests       bool                  `yaml:"required_tests"`
	MinTestCoverage     float64               `yaml:"min_test_coverage"`
	MinConfidence       float64               `yaml:"min_confidence"`
	ExcludedPackages    []string              `yaml:"excluded_packages"`
	IncludedPackages    []string              `yaml:"included_packages"`
	Ecosystems          []DependencyEcosystem `yaml:"ecosystems"`
	CustomRules         []DependencyRule      `yaml:"custom_rules"`
	SupportedBots       []string              `yaml:"supported_bots"`        // "dependabot", "snyk"
	SimplePRFastPath    SimplePRFastPath      `yaml:"simple_pr_fast_path"`   // Fast-path configuration
	Snyk                SnykConfig            `yaml:"snyk"`                  // Snyk-specific config
}

// SimplePRFastPath configures the fast-path for simple dependency PRs
type SimplePRFastPath struct {
	Enabled             bool `yaml:"enabled"`
	PatchOnly           bool `yaml:"patch_only"`
	PopularPackagesOnly bool `yaml:"popular_packages_only"`
	MinWeeklyDownloads  int  `yaml:"min_weekly_downloads"`
	MaxDiffLines        int  `yaml:"max_diff_lines"`
	BlockSecurityFixes  bool `yaml:"block_security_fixes"` // Security fixes need AI analysis
}

// SnykConfig represents Snyk-specific configuration
type SnykConfig struct {
	Enabled            bool `yaml:"enabled"`
	AutoApprovePatches bool `yaml:"auto_approve_patches"`
	TrustSnykPriority  bool `yaml:"trust_snyk_priority"` // Trust Snyk's severity assessment
}

// DependencyRule represents a custom rule for dependency automation
type DependencyRule struct {
	Name        string                   `yaml:"name"`
	Pattern     string                   `yaml:"pattern"`     // Package name pattern
	UpdateType  DependencyUpdateType     `yaml:"update_type"` // Which update types this applies to
	Action      DependencyRecommendation `yaml:"action"`      // What action to take
	Conditions  map[string]interface{}   `yaml:"conditions"`  // Additional conditions
	Description string                   `yaml:"description"`
}

// PRAutomationResult represents the result of automated PR handling
type PRAutomationResult struct {
	PRID         string              `json:"pr_id"`
	Action       PRAction            `json:"action"`
	Reasoning    string              `json:"reasoning"`
	Confidence   float64             `json:"confidence"`
	ExecutedAt   time.Time           `json:"executed_at"`
	ExecutedBy   string              `json:"executed_by"` // "liberation-guardian"
	TrustLevel   TrustLevel          `json:"trust_level"`
	Analysis     *DependencyAnalysis `json:"analysis"`
	TestResults  *TestResults        `json:"test_results,omitempty"`
	RollbackPlan *RollbackPlan       `json:"rollback_plan,omitempty"`
}

// PRAction represents actions that can be taken on a PR
type PRAction string

const (
	ActionApprove  PRAction = "approve"
	ActionMerge    PRAction = "merge"
	ActionComment  PRAction = "comment"
	ActionReject   PRAction = "reject"
	ActionEscalate PRAction = "escalate"
	ActionMonitor  PRAction = "monitor"
)

// TestResults represents automated test execution results
type TestResults struct {
	Passed            bool     `json:"passed"`
	TestSuite         string   `json:"test_suite"`
	Coverage          float64  `json:"coverage"`
	Duration          int64    `json:"duration_ms"`
	FailedTests       []string `json:"failed_tests,omitempty"`
	NewFailures       []string `json:"new_failures,omitempty"`
	PerformanceImpact float64  `json:"performance_impact"`
	Logs              string   `json:"logs,omitempty"`
}

// RollbackPlan represents a plan for rolling back a dependency update
type RollbackPlan struct {
	Triggers         []string       `json:"triggers"` // What conditions trigger rollback
	Steps            []RollbackStep `json:"steps"`    // Steps to perform rollback
	EstimatedTime    int            `json:"estimated_time_minutes"`
	AutoRollback     bool           `json:"auto_rollback"` // Whether to auto-rollback
	MonitoringWindow int            `json:"monitoring_window_hours"`
}

// RollbackStep represents a single step in a rollback plan
type RollbackStep struct {
	Action      string            `json:"action"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters"`
	Timeout     int               `json:"timeout_seconds"`
}

// DependencySecurityVulnerability represents a security vulnerability
type DependencySecurityVulnerability struct {
	CVE              string             `json:"cve"`
	CVSS             float64            `json:"cvss"`
	Severity         DependencySeverity `json:"severity"`
	Description      string             `json:"description"`
	AffectedVersions []string           `json:"affected_versions"`
	PatchedVersions  []string           `json:"patched_versions"`
	ExploitAvailable bool               `json:"exploit_available"`
	ExploitPublic    bool               `json:"exploit_public"`
}

// GitHubDependabotWebhook represents a webhook payload from GitHub Dependabot
type GitHubDependabotWebhook struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		ID     int    `json:"id"`
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		User   struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		URL       string `json:"html_url"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"pull_request"`
	Repository struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Owner    struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
}
