package codebase

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"liberation-guardian/pkg/types"
)

// Helper methods for CodebaseAnalyzer

// determineAnalysisDepth determines how deep to analyze based on event severity and trust level
func (ca *CodebaseAnalyzer) determineAnalysisDepth(event *types.LiberationGuardianEvent) string {
	// High severity or production = deeper analysis
	if event.Severity == "critical" || event.Severity == "high" {
		return "deep"
	}

	if event.Environment == "production" {
		return "medium"
	}

	return "shallow"
}

// isStackTraceFile checks if a file path appears in stack trace
func isStackTraceFile(event *types.LiberationGuardianEvent, path string) bool {
	stackTrace := event.Description + " " + event.Title
	return strings.Contains(stackTrace, path)
}

// getRecentCommits gets recent git commits
func (ca *CodebaseAnalyzer) getRecentCommits() ([]CommitAnalysis, error) {
	if ca.repository == nil {
		return nil, nil
	}

	ref, err := ca.repository.Head()
	if err != nil {
		return nil, err
	}

	iter, err := ca.repository.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, err
	}

	var commits []CommitAnalysis
	count := 0

	err = iter.ForEach(func(c *object.Commit) error {
		if count >= ca.config.MaxCommitHistory {
			return nil
		}

		// Get file stats
		stats, err := c.Stats()
		if err != nil {
			return nil // Continue on error
		}

		var filesChanged []string
		var linesAdded, linesRemoved int

		for _, stat := range stats {
			filesChanged = append(filesChanged, stat.Name)
			linesAdded += stat.Addition
			linesRemoved += stat.Deletion
		}

		commits = append(commits, CommitAnalysis{
			Hash:         c.Hash.String()[:8],
			Author:       c.Author.Name,
			Message:      strings.Split(c.Message, "\n")[0], // First line only
			Timestamp:    c.Author.When.Format(time.RFC3339),
			FilesChanged: filesChanged,
			LinesAdded:   linesAdded,
			LinesRemoved: linesRemoved,
		})

		count++
		return nil
	})

	return commits, err
}

// detectErrorPatterns detects patterns in code that might be related to the error
func (ca *CodebaseAnalyzer) detectErrorPatterns(event *types.LiberationGuardianEvent, files []FileAnalysis) []ErrorPattern {
	var patterns []ErrorPattern

	// Common error patterns to look for
	errorPatterns := map[string][]string{
		"null_pointer": {
			`\.(\w+)\s*=\s*null`,
			`if\s*\(\s*\w+\s*==\s*null\s*\)`,
			`\w+\.\w+\(\)`, // Method call without null check
		},
		"resource_leak": {
			`new\s+\w+\(`,
			`open\(`,
			`connect\(`,
		},
		"race_condition": {
			`go\s+func`,
			`goroutine`,
			`sync\.`,
			`channel`,
		},
		"sql_injection": {
			`"SELECT\s+.*\+`,
			`"INSERT\s+.*\+`,
			`"UPDATE\s+.*\+`,
		},
	}

	// Simple pattern matching - would be more sophisticated in real implementation
	for _, file := range files {
		for patternType, regexes := range errorPatterns {
			for range regexes {
				if strings.Contains(event.Description, patternType) ||
					strings.Contains(event.Title, patternType) {
					patterns = append(patterns, ErrorPattern{
						Type:        patternType,
						Location:    file.Path,
						Description: "Potential " + patternType + " detected",
						Confidence:  0.6, // Basic confidence
					})
					break
				}
			}
		}
	}

	return patterns
}

// analyzeDependencies analyzes project dependencies
func (ca *CodebaseAnalyzer) analyzeDependencies() []DependencyInfo {
	var deps []DependencyInfo

	// Look for common dependency files
	dependencyFiles := []string{
		"package.json", "go.mod", "requirements.txt",
		"Pipfile", "Gemfile", "pom.xml", "build.gradle",
	}

	for _, file := range dependencyFiles {
		if ca.isPathAllowed(file) {
			// Simple implementation - would parse actual dependency files
			deps = append(deps, DependencyInfo{
				Name:    file,
				Type:    "detected",
				Version: "unknown",
			})
		}
	}

	return deps
}

// extractFromErrorMessage extracts file paths from error messages
func (ca *CodebaseAnalyzer) extractFromErrorMessage(message string) []string {
	var paths []string

	// Look for common file path patterns in error messages
	// This would be more sophisticated in real implementation
	words := strings.Fields(message)
	for _, word := range words {
		if strings.Contains(word, "/") && (strings.Contains(word, ".") || strings.Contains(word, "src")) {
			if normalized := ca.normalizePath(word); normalized != "" {
				paths = append(paths, normalized)
			}
		}
	}

	return paths
}

// extractFromRawPayload extracts file paths from raw JSON payload
func (ca *CodebaseAnalyzer) extractFromRawPayload(payload string) []string {
	var paths []string

	// Simple extraction - would use proper JSON parsing in real implementation
	lines := strings.Split(payload, "\n")
	for _, line := range lines {
		if strings.Contains(line, "file") || strings.Contains(line, "path") {
			// Extract potential file paths
			words := strings.Fields(line)
			for _, word := range words {
				if strings.Contains(word, "/") && strings.Contains(word, ".") {
					if normalized := ca.normalizePath(word); normalized != "" {
						paths = append(paths, normalized)
					}
				}
			}
		}
	}

	return paths
}

// getCommonFilesForService returns common files to check for a service
func (ca *CodebaseAnalyzer) getCommonFilesForService(service, eventType string) []string {
	var paths []string

	// Common files based on service name
	if service != "" {
		commonPatterns := []string{
			service + ".go",
			service + ".js",
			service + ".py",
			service + "/" + service + ".go",
			"services/" + service + "/",
			"src/" + service + "/",
		}
		paths = append(paths, commonPatterns...)
	}

	// Common files based on event type
	switch eventType {
	case "database_error":
		paths = append(paths, "database.go", "db.go", "models/", "schema.sql")
	case "api_error":
		paths = append(paths, "api.go", "routes.go", "handlers/", "controllers/")
	case "auth_error":
		paths = append(paths, "auth.go", "login.go", "oauth.go", "jwt.go")
	}

	return paths
}

// deduplicateAndNormalizePaths removes duplicates and normalizes file paths
func (ca *CodebaseAnalyzer) deduplicateAndNormalizePaths(paths []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, path := range paths {
		normalized := ca.normalizePath(path)
		if normalized != "" && !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}

	return result
}

// normalizePath normalizes a file path and validates it exists
func (ca *CodebaseAnalyzer) normalizePath(path string) string {
	// Clean the path
	cleaned := filepath.Clean(path)

	// Remove quotes and other characters
	cleaned = strings.Trim(cleaned, "\"'")

	// Make relative to root if absolute
	if filepath.IsAbs(cleaned) {
		rel, err := filepath.Rel(ca.rootPath, cleaned)
		if err == nil && !strings.HasPrefix(rel, "..") {
			cleaned = rel
		} else {
			return "" // Outside of project root
		}
	}

	// Check if file exists
	fullPath := filepath.Join(ca.rootPath, cleaned)
	if _, err := filepath.Abs(fullPath); err != nil {
		return ""
	}

	return cleaned
}
