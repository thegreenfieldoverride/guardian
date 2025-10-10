package codebase

// DependencyInfo represents a dependency analysis
type DependencyInfo struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Type             string `json:"type"` // direct, indirect, dev
	HasVulnerability bool   `json:"has_vulnerability"`
	SecurityRisk     string `json:"security_risk"` // low, medium, high, critical
}

// SimilarIssue represents a similar issue found in knowledge base
type SimilarIssue struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Resolution  string  `json:"resolution"`
	Confidence  float64 `json:"confidence"`
	Timestamp   string  `json:"timestamp"`
}

// TestCoverage represents test coverage information
type TestCoverage struct {
	Overall      float64            `json:"overall"`
	ByFile       map[string]float64 `json:"by_file"`
	TestFiles    []string           `json:"test_files"`
	MissingTests []string           `json:"missing_tests"`
}
