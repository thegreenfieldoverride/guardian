package dependencies

import (
	"strings"

	"github.com/sirupsen/logrus"

	"liberation-guardian/pkg/types"
)

// SimplePRDetector determines if a dependency PR is simple enough to skip AI analysis
type SimplePRDetector struct {
	logger          *logrus.Logger
	popularPackages map[string]bool
	config          *SimplePRFastPathConfig
}

// SimplePRFastPathConfig configures the fast-path logic
type SimplePRFastPathConfig struct {
	Enabled             bool
	PatchOnly           bool
	PopularPackagesOnly bool
	MinWeeklyDownloads  int
	MaxDiffLines        int
	BlockSecurityFixes  bool
}

// NewSimplePRDetector creates a new simple PR detector
func NewSimplePRDetector(logger *logrus.Logger, config *SimplePRFastPathConfig) *SimplePRDetector {
	if config == nil {
		config = &SimplePRFastPathConfig{
			Enabled:             true,
			PatchOnly:           true,
			PopularPackagesOnly: true,
			MinWeeklyDownloads:  100000,
			MaxDiffLines:        50,
			BlockSecurityFixes:  true,
		}
	}

	return &SimplePRDetector{
		logger:          logger,
		config:          config,
		popularPackages: buildPopularPackagesMap(),
	}
}

// IsSimplePR determines if a PR is simple enough to skip AI analysis
func (spd *SimplePRDetector) IsSimplePR(update *types.DependencyUpdate) bool {
	// Fast-path must be enabled
	if !spd.config.Enabled {
		return false
	}

	spd.logger.Debugf("Evaluating fast-path for %s %s → %s", update.PackageName, update.CurrentVersion, update.NewVersion)

	// 1. Must be patch update
	if spd.config.PatchOnly && update.UpdateType != types.UpdateTypePatch {
		spd.logger.Debugf("Not a patch update: %s", update.UpdateType)
		return false
	}

	// 2. Must be popular package (if required)
	if spd.config.PopularPackagesOnly && !spd.isPopularPackage(update.PackageName, update.Ecosystem) {
		spd.logger.Debugf("Not a popular package: %s", update.PackageName)
		return false
	}

	// 3. Security fixes should go through AI (if configured)
	if spd.config.BlockSecurityFixes && len(update.CVEFixed) > 0 {
		spd.logger.Debugf("Security fix detected, blocking fast-path")
		return false
	}

	// 4. Check for breaking changes in changelog/notes
	if spd.hasBreakingChanges(update) {
		spd.logger.Debugf("Breaking changes detected")
		return false
	}

	// 5. Check diff size (if available)
	if update.DiffStats != nil && spd.config.MaxDiffLines > 0 {
		totalLines := update.DiffStats.Additions + update.DiffStats.Deletions
		if totalLines > spd.config.MaxDiffLines {
			spd.logger.Debugf("Diff too large: %d lines (max: %d)", totalLines, spd.config.MaxDiffLines)
			return false
		}
	}

	spd.logger.Infof("Fast-path eligible: %s %s → %s", update.PackageName, update.CurrentVersion, update.NewVersion)
	return true
}

// isPopularPackage checks if a package is popular enough for fast-path
func (spd *SimplePRDetector) isPopularPackage(packageName string, ecosystem types.DependencyEcosystem) bool {
	// Normalize package name
	normalized := strings.ToLower(packageName)

	// Check against popular packages map
	key := string(ecosystem) + ":" + normalized
	if popular, exists := spd.popularPackages[key]; exists && popular {
		return true
	}

	// Also check without ecosystem prefix (for packages popular across ecosystems)
	if popular, exists := spd.popularPackages[normalized]; exists && popular {
		return true
	}

	return false
}

// hasBreakingChanges checks if the update has breaking changes
func (spd *SimplePRDetector) hasBreakingChanges(update *types.DependencyUpdate) bool {
	// Check changelog/release notes for breaking change indicators
	changelogLower := strings.ToLower(update.ChangelogURL)
	releaseNotesLower := strings.ToLower(update.ReleaseNotesURL)

	breakingIndicators := []string{
		"breaking",
		"breaking change",
		"incompatible",
		"removed",
		"deprecated",
		"migration required",
	}

	for _, indicator := range breakingIndicators {
		if strings.Contains(changelogLower, indicator) || strings.Contains(releaseNotesLower, indicator) {
			return true
		}
	}

	return false
}

// buildPopularPackagesMap builds a map of popular packages by ecosystem
func buildPopularPackagesMap() map[string]bool {
	packages := make(map[string]bool)

	// npm popular packages
	npmPackages := []string{
		"lodash", "react", "react-dom", "axios", "express", "moment", "vue",
		"jquery", "webpack", "typescript", "eslint", "prettier", "jest",
		"babel", "next", "redux", "graphql", "apollo", "chalk", "commander",
		"debug", "dotenv", "uuid", "classnames", "prop-types", "styled-components",
	}
	for _, pkg := range npmPackages {
		packages["npm:"+pkg] = true
	}

	// Python popular packages
	pythonPackages := []string{
		"requests", "numpy", "pandas", "django", "flask", "pytest",
		"sqlalchemy", "boto3", "pillow", "matplotlib", "scipy", "setuptools",
		"wheel", "pip", "black", "flake8", "mypy", "pydantic", "fastapi",
		"celery", "redis", "psycopg2", "cryptography", "jinja2",
	}
	for _, pkg := range pythonPackages {
		packages["pypi:"+pkg] = true
	}

	// Go popular packages
	goPackages := []string{
		"github.com/gin-gonic/gin", "github.com/gorilla/mux", "github.com/sirupsen/logrus",
		"github.com/stretchr/testify", "github.com/spf13/cobra", "github.com/spf13/viper",
		"github.com/lib/pq", "github.com/go-sql-driver/mysql", "github.com/redis/go-redis",
		"google.golang.org/grpc", "golang.org/x/crypto", "gopkg.in/yaml.v3",
	}
	for _, pkg := range goPackages {
		packages["go:"+pkg] = true
	}

	// Ruby popular packages
	rubyPackages := []string{
		"rails", "rack", "rake", "bundler", "rspec", "puma", "sidekiq",
		"devise", "pg", "redis", "nokogiri", "activerecord", "actionpack",
	}
	for _, pkg := range rubyPackages {
		packages["rubygems:"+pkg] = true
	}

	return packages
}
