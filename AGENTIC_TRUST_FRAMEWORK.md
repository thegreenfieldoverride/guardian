# Liberation Guardian: Agentic Trust & Safety Framework

## 🎯 **User-Controlled AI Trust Levels**

Users configure their comfort level with AI autonomy. Liberation Guardian adapts its behavior accordingly.

### **Trust Level Configuration**

```yaml
# liberation-guardian.yml
agentic_config:
  # User-selected trust level
  trust_level: "cautious"  # paranoid | cautious | balanced | confident | autonomous
  
  # Custom trust overrides
  custom_trust:
    # Repository-specific trust
    repositories:
      - repo: "critical-service"
        trust_level: "paranoid"
      - repo: "dev-tools" 
        trust_level: "autonomous"
    
    # File pattern-based trust
    file_patterns:
      - pattern: "*.config"
        trust_level: "cautious"
      - pattern: "test/**"
        trust_level: "confident" 
      - pattern: "docs/**"
        trust_level: "autonomous"
        
    # Severity-based trust
    severity_overrides:
      critical: "paranoid"    # Always require human approval
      high: "cautious"        # AI suggests, human decides
      medium: "balanced"      # AI can auto-fix with constraints
      low: "confident"        # AI can auto-fix freely
      
  # Advanced controls
  constraints:
    max_files_per_fix: 3
    require_tests_for_autofix: true
    require_ci_pass: true
    auto_rollback_on_failure: true
```

### **Trust Level Behaviors**

```go
type TrustLevel int

const (
    TrustParanoid TrustLevel = iota    // AI suggests, human decides everything
    TrustCautious                     // AI can auto-acknowledge, human approves fixes
    TrustBalanced                     // AI can auto-fix low-risk issues
    TrustConfident                    // AI can auto-fix most issues
    TrustAutonomous                   // AI operates with minimal human oversight
)

type TrustProfile struct {
    Level               TrustLevel
    MaxFilesPerFix      int
    MaxLinesChanged     int
    RequireTests        bool
    RequireCI          bool
    RequireReview      bool
    AutoMergeThreshold float64
    EscalationRules    []EscalationRule
}

// Trust level behaviors
var TrustProfiles = map[TrustLevel]TrustProfile{
    TrustParanoid: {
        Level:               TrustParanoid,
        MaxFilesPerFix:      0,  // No auto-fixes
        RequireReview:      true,
        AutoMergeThreshold: 0.0, // Never auto-merge
        EscalationRules: []EscalationRule{
            {Condition: "any_change", Action: "require_human_approval"},
        },
    },
    TrustCautious: {
        Level:               TrustCautious,
        MaxFilesPerFix:      1,
        MaxLinesChanged:     10,
        RequireTests:       true,
        RequireCI:          true,
        RequireReview:      true,
        AutoMergeThreshold: 0.0, // No auto-merge, but can create PRs
    },
    TrustBalanced: {
        Level:               TrustBalanced,
        MaxFilesPerFix:      2,
        MaxLinesChanged:     25,
        RequireTests:       true,
        RequireCI:          true,
        RequireReview:      false, // Can auto-merge if confidence high
        AutoMergeThreshold: 0.9,
    },
    TrustConfident: {
        Level:               TrustConfident,
        MaxFilesPerFix:      5,
        MaxLinesChanged:     100,
        RequireTests:       true,
        RequireCI:          true,
        RequireReview:      false,
        AutoMergeThreshold: 0.8,
    },
    TrustAutonomous: {
        Level:               TrustAutonomous,
        MaxFilesPerFix:      10,
        MaxLinesChanged:     500,
        RequireTests:       false, // Can create tests if needed
        RequireCI:          true,
        RequireReview:      false,
        AutoMergeThreshold: 0.7,
    },
}
```

## 🧠 **Intelligent Auto-Merge Rubric**

The AI uses multiple factors to determine if a fix should be auto-merged:

### **Auto-Merge Decision Matrix**

```go
type AutoMergeDecision struct {
    ShouldAutoMerge bool
    Confidence     float64
    Reasoning      string
    Factors        AutoMergeFactors
}

type AutoMergeFactors struct {
    // Risk Assessment (0.0 = high risk, 1.0 = low risk)
    RiskScore           float64 `json:"risk_score"`
    ChangeComplexity    float64 `json:"change_complexity"`    // Lines, files, function complexity
    TestCoverage        float64 `json:"test_coverage"`        // Existing + new test coverage
    SimilarityToPast    float64 `json:"similarity_to_past"`   // How similar to successful past fixes
    
    // Confidence Factors (0.0 = low confidence, 1.0 = high confidence)
    AIConfidence        float64 `json:"ai_confidence"`        // AI's confidence in the fix
    PatternMatches      float64 `json:"pattern_matches"`      // How well it matches known patterns
    CodeQuality         float64 `json:"code_quality"`         // Static analysis score
    
    // Environmental Factors
    Environment         string  `json:"environment"`          // dev, staging, production
    Severity           string  `json:"severity"`             // critical, high, medium, low
    BusinessHours      bool    `json:"business_hours"`       // Is it business hours?
    RecentFailures     int     `json:"recent_failures"`      // How many recent failed auto-merges?
}

func (am *AutoMergeEngine) ShouldAutoMerge(fix *AutoFix, profile TrustProfile) AutoMergeDecision {
    factors := am.calculateFactors(fix)
    
    // Weighted scoring based on trust level
    weights := am.getWeightsForTrustLevel(profile.Level)
    
    score := (factors.RiskScore * weights.Risk) +
             (factors.AIConfidence * weights.Confidence) +
             (factors.TestCoverage * weights.Testing) +
             (factors.SimilarityToPast * weights.Historical)
    
    // Apply hard constraints
    if !am.meetsHardConstraints(fix, profile) {
        return AutoMergeDecision{
            ShouldAutoMerge: false,
            Confidence:     0.0,
            Reasoning:      "Failed hard constraints check",
        }
    }
    
    // Check against threshold
    shouldMerge := score >= profile.AutoMergeThreshold
    
    return AutoMergeDecision{
        ShouldAutoMerge: shouldMerge,
        Confidence:     score,
        Reasoning:      am.explainDecision(factors, weights, score),
        Factors:        factors,
    }
}
```

### **Risk Assessment Categories**

```go
type RiskCategory struct {
    Name        string
    Weight      float64
    Evaluator   func(*AutoFix) float64
}

var RiskCategories = []RiskCategory{
    {
        Name:   "file_criticality",
        Weight: 0.3,
        Evaluator: func(fix *AutoFix) float64 {
            // Critical files: main.go, config files, database migrations
            // Low risk: test files, documentation, dev tools
            for _, file := range fix.ModifiedFiles {
                if isCriticalFile(file) {
                    return 0.2 // High risk
                }
            }
            return 0.8 // Low risk
        },
    },
    {
        Name:   "change_scope",
        Weight: 0.25,
        Evaluator: func(fix *AutoFix) float64 {
            lines := fix.LinesChanged
            files := len(fix.ModifiedFiles)
            
            if lines > 100 || files > 5 {
                return 0.3 // High risk
            } else if lines > 20 || files > 2 {
                return 0.6 // Medium risk  
            }
            return 0.9 // Low risk
        },
    },
    {
        Name:   "function_complexity",
        Weight: 0.2,
        Evaluator: func(fix *AutoFix) float64 {
            // Analyze cyclomatic complexity of modified functions
            complexity := fix.CyclomaticComplexity
            if complexity > 10 {
                return 0.2 // High complexity = high risk
            } else if complexity > 5 {
                return 0.6
            }
            return 0.9 // Low complexity = low risk
        },
    },
    {
        Name:   "dependency_impact",
        Weight: 0.15,
        Evaluator: func(fix *AutoFix) float64 {
            // Check if fix affects dependencies, public APIs, etc.
            if fix.AffectsPublicAPI || fix.AffectsDependencies {
                return 0.2
            }
            return 0.9
        },
    },
    {
        Name:   "environment_risk",
        Weight: 0.1,
        Evaluator: func(fix *AutoFix) float64 {
            switch fix.Environment {
            case "production":
                return 0.3
            case "staging":
                return 0.7
            case "development":
                return 0.9
            default:
                return 0.5
            }
        },
    },
}
```

## 🚨 **Safety Guardrails & Circuit Breakers**

```go
type SafetyGuardrails struct {
    // Circuit breakers
    MaxAutoMergesPerHour    int     `yaml:"max_auto_merges_per_hour"`
    MaxAutoMergesPerDay     int     `yaml:"max_auto_merges_per_day"`
    FailureRateThreshold    float64 `yaml:"failure_rate_threshold"`
    
    // Quality gates
    MinTestCoverage         float64 `yaml:"min_test_coverage"`
    RequiredChecks          []string `yaml:"required_checks"`
    ForbiddenPatterns       []string `yaml:"forbidden_patterns"`
    
    // Emergency controls
    EmergencyStop           bool    `yaml:"emergency_stop"`
    PauseOnIncident         bool    `yaml:"pause_on_incident"`
    RollbackOnFailure       bool    `yaml:"rollback_on_failure"`
}

func (sg *SafetyGuardrails) CheckSafety(fix *AutoFix) SafetyResult {
    // Check circuit breakers
    if sg.isCircuitBreakerTripped() {
        return SafetyResult{
            Allowed: false,
            Reason:  "Circuit breaker tripped - too many recent failures",
        }
    }
    
    // Check rate limits
    if sg.exceedsRateLimit() {
        return SafetyResult{
            Allowed: false,
            Reason:  "Rate limit exceeded for auto-merges",
        }
    }
    
    // Check forbidden patterns
    if sg.containsForbiddenPatterns(fix) {
        return SafetyResult{
            Allowed: false,
            Reason:  "Contains forbidden code patterns",
        }
    }
    
    return SafetyResult{Allowed: true}
}
```

## 🎛️ **User Control Interface**

```go
// CLI commands for trust management
func main() {
    app := &cli.App{
        Commands: []*cli.Command{
            {
                Name: "trust",
                Subcommands: []*cli.Command{
                    {
                        Name: "set",
                        Usage: "Set AI trust level",
                        Action: func(c *cli.Context) error {
                            level := c.Args().First()
                            return setTrustLevel(level)
                        },
                    },
                    {
                        Name: "status",
                        Usage: "Show current trust settings",
                        Action: showTrustStatus,
                    },
                    {
                        Name: "pause",
                        Usage: "Pause all AI automation",
                        Action: pauseAutomation,
                    },
                    {
                        Name: "emergency-stop",
                        Usage: "Emergency stop all AI operations",
                        Action: emergencyStop,
                    },
                },
            },
        },
    }
}
```

## 🎯 **Implementation Roadmap**

### **Phase 1: Trust Framework Foundation**
- ✅ User-configurable trust levels
- ✅ Basic safety guardrails  
- ✅ Read-only codebase analysis
- ✅ Enhanced triage with code context

### **Phase 2: Intelligent Auto-Fix**
- ✅ Risk assessment engine
- ✅ Auto-merge decision matrix
- ✅ PR creation and management
- ✅ Circuit breakers and rate limiting

### **Phase 3: Advanced Agentic Operations**
- ✅ Proactive issue detection
- ✅ Self-healing systems
- ✅ Learning from user feedback
- ✅ Cross-repository pattern recognition

**Liberation Philosophy:** Maximum autonomy with maximum user control. The user is always in charge.