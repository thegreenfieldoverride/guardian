package codebase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/sirupsen/logrus"

	"liberation-guardian/pkg/types"
)

// CodebaseAnalyzer provides read-only analysis of codebase for enhanced AI triage
type CodebaseAnalyzer struct {
	logger     *logrus.Logger
	rootPath   string
	repository *git.Repository
	config     *AnalyzerConfig
}

// AnalyzerConfig controls what the analyzer can access
type AnalyzerConfig struct {
	// Security: Read-only access controls
	AllowedPaths    []string `yaml:"allowed_paths"`    // Paths AI can read
	BlockedPaths    []string `yaml:"blocked_paths"`    // Paths AI cannot access
	BlockedPatterns []string `yaml:"blocked_patterns"` // File patterns to ignore
	MaxFileSize     int64    `yaml:"max_file_size"`    // Max file size to read (bytes)
	MaxFiles        int      `yaml:"max_files"`        // Max files to analyze per request

	// Analysis controls
	IncludeGitHistory bool `yaml:"include_git_history"` // Include recent commits
	MaxCommitHistory  int  `yaml:"max_commit_history"`  // How many recent commits to analyze

	// Trust level (from main config)
	TrustLevel string `yaml:"trust_level"`
}

// CodeContext provides relevant code context for AI triage
type CodeContext struct {
	// Basic file information
	RelevantFiles   []FileAnalysis   `json:"relevant_files"`
	StackTraceFiles []FileAnalysis   `json:"stack_trace_files"`
	RecentChanges   []CommitAnalysis `json:"recent_changes"`
	Dependencies    []DependencyInfo `json:"dependencies"`

	// Contextual analysis
	ErrorPatterns []ErrorPattern `json:"error_patterns"`
	SimilarIssues []SimilarIssue `json:"similar_issues"`
	TestCoverage  *TestCoverage  `json:"test_coverage,omitempty"`

	// Metadata
	AnalysisDepth   string `json:"analysis_depth"` // shallow, medium, deep
	FilesAnalyzed   int    `json:"files_analyzed"`
	SecurityLimited bool   `json:"security_limited"` // Was analysis limited by security?
}

// FileAnalysis contains analysis of a single file
type FileAnalysis struct {
	Path          string `json:"path"`
	Language      string `json:"language"`
	LineCount     int    `json:"line_count"`
	Function      string `json:"function,omitempty"`     // Function containing error
	LineNumber    int    `json:"line_number,omitempty"`  // Specific line if relevant
	CodeSnippet   string `json:"code_snippet,omitempty"` // Relevant code around issue
	Complexity    int    `json:"complexity,omitempty"`   // Cyclomatic complexity
	LastModified  string `json:"last_modified"`
	RecentChanges bool   `json:"recent_changes"`
	IsTestFile    bool   `json:"is_test_file"`
	IsCritical    bool   `json:"is_critical"` // Main files, configs, etc.
}

// CommitAnalysis contains recent git commit information
type CommitAnalysis struct {
	Hash         string   `json:"hash"`
	Author       string   `json:"author"`
	Message      string   `json:"message"`
	Timestamp    string   `json:"timestamp"`
	FilesChanged []string `json:"files_changed"`
	LinesAdded   int      `json:"lines_added"`
	LinesRemoved int      `json:"lines_removed"`
}

// ErrorPattern represents a detected error pattern in code
type ErrorPattern struct {
	Type        string  `json:"type"`     // null_pointer, resource_leak, etc.
	Location    string  `json:"location"` // File:line
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
}

// NewCodebaseAnalyzer creates a new codebase analyzer
func NewCodebaseAnalyzer(logger *logrus.Logger, rootPath string, config *AnalyzerConfig) (*CodebaseAnalyzer, error) {
	// Verify rootPath exists and is accessible
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("codebase path does not exist: %s", rootPath)
	}

	// Try to open git repository (optional)
	repo, err := git.PlainOpen(rootPath)
	if err != nil {
		logger.Warnf("Could not open git repository at %s: %v", rootPath, err)
		// Continue without git - still useful for file analysis
	}

	// Apply security defaults if not configured
	if config == nil {
		config = defaultAnalyzerConfig()
	}

	return &CodebaseAnalyzer{
		logger:     logger,
		rootPath:   rootPath,
		repository: repo,
		config:     config,
	}, nil
}

// AnalyzeForEvent analyzes codebase relevant to a specific event
func (ca *CodebaseAnalyzer) AnalyzeForEvent(ctx context.Context, event *types.LiberationGuardianEvent) (*CodeContext, error) {
	ca.logger.Infof("Starting codebase analysis for event %s from %s", event.ID, event.Source)

	context := &CodeContext{
		AnalysisDepth: ca.determineAnalysisDepth(event),
		FilesAnalyzed: 0,
	}

	// Extract file paths from event (stack traces, error messages, etc.)
	relevantPaths := ca.extractRelevantPaths(event)

	// Analyze relevant files
	for _, path := range relevantPaths {
		if context.FilesAnalyzed >= ca.config.MaxFiles {
			context.SecurityLimited = true
			break
		}

		if ca.isPathAllowed(path) {
			analysis, err := ca.analyzeFile(path)
			if err != nil {
				ca.logger.Warnf("Failed to analyze file %s: %v", path, err)
				continue
			}

			if isStackTraceFile(event, path) {
				context.StackTraceFiles = append(context.StackTraceFiles, *analysis)
			} else {
				context.RelevantFiles = append(context.RelevantFiles, *analysis)
			}
			context.FilesAnalyzed++
		}
	}

	// Add git history if enabled and available
	if ca.config.IncludeGitHistory && ca.repository != nil {
		commits, err := ca.getRecentCommits()
		if err != nil {
			ca.logger.Warnf("Failed to get recent commits: %v", err)
		} else {
			context.RecentChanges = commits
		}
	}

	// Detect error patterns
	context.ErrorPatterns = ca.detectErrorPatterns(event, context.RelevantFiles)

	// Find dependencies if package files present
	context.Dependencies = ca.analyzeDependencies()

	ca.logger.Infof("Codebase analysis complete: %d files analyzed, %d patterns detected",
		context.FilesAnalyzed, len(context.ErrorPatterns))

	return context, nil
}

// extractRelevantPaths extracts file paths from event data
func (ca *CodebaseAnalyzer) extractRelevantPaths(event *types.LiberationGuardianEvent) []string {
	var paths []string

	// Extract from stack traces
	stackTracePaths := ca.extractFromStackTrace(event.Description)
	paths = append(paths, stackTracePaths...)

	// Extract from error messages
	errorPaths := ca.extractFromErrorMessage(event.Title + " " + event.Description)
	paths = append(paths, errorPaths...)

	// Extract from raw payload if available
	if len(event.RawPayload) > 0 {
		payloadPaths := ca.extractFromRawPayload(string(event.RawPayload))
		paths = append(paths, payloadPaths...)
	}

	// Add common files based on event type and service
	commonPaths := ca.getCommonFilesForService(event.Service, event.Type)
	paths = append(paths, commonPaths...)

	// Remove duplicates and normalize
	return ca.deduplicateAndNormalizePaths(paths)
}

// extractFromStackTrace extracts file paths from stack traces
func (ca *CodebaseAnalyzer) extractFromStackTrace(stackTrace string) []string {
	var paths []string

	// Common stack trace patterns
	patterns := []string{
		`at\s+[\w\.]+\(([^:]+):(\d+)\)`,    // Java stack traces
		`File\s+"([^"]+)",\s+line\s+(\d+)`, // Python stack traces
		`([^:\s]+):(\d+):(\d+)`,            // Go stack traces
		`\s+([^:\s]+):(\d+):\d+`,           // TypeScript/JavaScript
		`\s+in\s+([^:\s]+):(\d+)`,          // Ruby stack traces
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(stackTrace, -1)

		for _, match := range matches {
			if len(match) >= 2 {
				filePath := match[1]
				// Normalize and validate path
				if normalizedPath := ca.normalizePath(filePath); normalizedPath != "" {
					paths = append(paths, normalizedPath)
				}
			}
		}
	}

	return paths
}

// isPathAllowed checks if a path is allowed by security configuration
func (ca *CodebaseAnalyzer) isPathAllowed(path string) bool {
	// Check against blocked paths
	for _, blocked := range ca.config.BlockedPaths {
		if strings.Contains(path, blocked) {
			return false
		}
	}

	// Check against blocked patterns
	for _, pattern := range ca.config.BlockedPatterns {
		matched, err := regexp.MatchString(pattern, path)
		if err != nil {
			ca.logger.Warnf("Invalid blocked pattern '%s': %v", pattern, err)
			continue
		}
		if matched {
			return false
		}
	}

	// Check against allowed paths (if specified)
	if len(ca.config.AllowedPaths) > 0 {
		for _, allowed := range ca.config.AllowedPaths {
			if strings.HasPrefix(path, allowed) {
				return true
			}
		}
		return false // Not in allowed paths
	}

	return true // Allowed by default if no restrictions
}

// defaultAnalyzerConfig returns secure default configuration
func defaultAnalyzerConfig() *AnalyzerConfig {
	return &AnalyzerConfig{
		// Security defaults - very restrictive
		AllowedPaths: []string{
			"src/", "lib/", "app/", "internal/", "pkg/",
			"tests/", "test/", "__tests__/",
			"docs/", "README.md", "*.md",
		},
		BlockedPaths: []string{
			".env", ".secret", ".key", ".pem", ".p12",
			"node_modules/", "vendor/", ".git/",
			"build/", "dist/", "target/", "bin/",
		},
		BlockedPatterns: []string{
			`.*\.env.*`,    // Environment files
			`.*secret.*`,   // Secret files
			`.*key.*`,      // Key files
			`.*password.*`, // Password files
			`.*\.log$`,     // Log files (may contain secrets)
		},
		MaxFileSize:       100 * 1024, // 100KB max per file
		MaxFiles:          20,         // Max 20 files per analysis
		IncludeGitHistory: true,
		MaxCommitHistory:  10,
		TrustLevel:        "cautious",
	}
}

// analyzeFile performs analysis on a single file
func (ca *CodebaseAnalyzer) analyzeFile(path string) (*FileAnalysis, error) {
	fullPath := filepath.Join(ca.rootPath, path)

	// Check file size
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	if info.Size() > ca.config.MaxFileSize {
		return nil, fmt.Errorf("file %s exceeds max size limit", path)
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	analysis := &FileAnalysis{
		Path:         path,
		Language:     detectLanguage(path),
		LineCount:    strings.Count(string(content), "\n"),
		LastModified: info.ModTime().Format("2006-01-02 15:04:05"),
		IsTestFile:   isTestFile(path),
		IsCritical:   isCriticalFile(path),
	}

	// Calculate complexity for code files
	if isCodeFile(path) {
		analysis.Complexity = calculateComplexity(string(content), analysis.Language)
	}

	return analysis, nil
}

// Helper functions
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc":
		return "cpp"
	case ".rs":
		return "rust"
	default:
		return "unknown"
	}
}

func isTestFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "test") ||
		strings.Contains(lower, "spec") ||
		strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".spec.js")
}

func isCriticalFile(path string) bool {
	lower := strings.ToLower(path)
	criticalPatterns := []string{
		"main.go", "main.js", "index.js", "app.js",
		"config", "setting", "environment",
		"database", "migration", "schema",
		"docker", "makefile", "package.json",
	}

	for _, pattern := range criticalPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

func isCodeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	codeExtensions := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".py": true,
		".java": true, ".rb": true, ".php": true, ".c": true,
		".cpp": true, ".rs": true, ".jsx": true, ".tsx": true,
	}
	return codeExtensions[ext]
}

func calculateComplexity(content, language string) int {
	// Simple cyclomatic complexity calculation
	// Count decision points: if, for, while, switch, case, &&, ||
	patterns := []string{
		`\bif\b`, `\bfor\b`, `\bwhile\b`, `\bswitch\b`, `\bcase\b`,
		`&&`, `\|\|`, `\btry\b`, `\bcatch\b`,
	}

	complexity := 1 // Base complexity
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(content, -1)
		complexity += len(matches)
	}

	return complexity
}

// Additional helper methods would be implemented here...
// getRecentCommits, detectErrorPatterns, analyzeDependencies, etc.
