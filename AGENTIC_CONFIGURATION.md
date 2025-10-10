# Liberation Guardian: Agentic Configuration Guide

## 🎛️ **Configuring Your AI Trust Level**

Add this to your `liberation-guardian.yml`:

```yaml
# Agentic AI Configuration
agentic:
  # Trust level determines AI autonomy
  trust_level: "cautious"  # paranoid | cautious | balanced | confident | autonomous
  
  # Codebase analysis configuration
  codebase_analysis:
    enabled: true
    root_path: "."  # Path to analyze (relative to Liberation Guardian)
    
    # Security controls
    allowed_paths:
      - "src/"
      - "internal/"
      - "pkg/"
      - "lib/"
      - "app/"
      - "tests/"
      - "docs/"
      - "*.md"
      - "*.go"
      - "*.js"
      - "*.ts"
      - "*.py"
      
    blocked_paths:
      - ".env"
      - ".secret"
      - ".key"
      - ".pem"
      - ".git/"
      - "node_modules/"
      - "vendor/"
      - "build/"
      - "dist/"
      - "logs/"
      
    blocked_patterns:
      - ".*\\.env.*"
      - ".*secret.*"
      - ".*password.*"
      - ".*\\.log$"
      
    # Analysis limits
    max_file_size: 102400  # 100KB max per file
    max_files: 20          # Max files per analysis
    include_git_history: true
    max_commit_history: 10
    
  # Auto-fix capabilities  
  auto_fix:
    enabled: true
    
    # What AI can auto-fix based on trust level
    allowed_actions:
      - "dependency_update"      # Update vulnerable dependencies
      - "lint_fix"              # Fix linting issues
      - "test_fix"              # Fix broken tests
      - "documentation_update"   # Update docs
      - "config_fix"            # Fix configuration issues
      
    # Constraints based on trust level
    max_files_per_fix: 3
    max_lines_changed: 50
    require_tests: true
    require_ci_pass: true
    require_review: true      # Create PR instead of direct merge
    
    # Auto-merge criteria (for higher trust levels)
    auto_merge:
      enabled: false          # Only enabled for balanced+ trust levels
      confidence_threshold: 0.9
      max_risk_score: 0.3     # Lower = safer
      require_all_checks: true
      
  # Safety controls
  safety:
    # Circuit breakers
    max_auto_fixes_per_hour: 5
    max_auto_fixes_per_day: 20
    failure_rate_threshold: 0.2  # Pause if >20% of fixes fail
    
    # Emergency controls
    emergency_stop: false
    pause_on_incident: true
    rollback_on_failure: true
    
    # Notification channels for escalation
    escalation:
      email: "ops@yourcompany.com"
      slack: "#ops-alerts"
      github_team: "@ops-team"
```

## 🛡️ **Trust Level Behaviors**

### **Paranoid** 🔒
```yaml
trust_level: "paranoid"
# Behavior:
# - AI only suggests fixes, never applies
# - All decisions require human approval
# - Maximum security restrictions
# - Read-only codebase analysis
```

### **Cautious** ⚠️ (Recommended Start)
```yaml
trust_level: "cautious"  
# Behavior:
# - AI can auto-acknowledge low-risk events
# - AI creates PRs for fixes, human reviews
# - Limited file access
# - Conservative change limits
```

### **Balanced** ⚖️
```yaml
trust_level: "balanced"
# Behavior:
# - AI can auto-fix low-risk issues
# - Auto-merge dependency updates if tests pass
# - Moderate file access
# - Can modify up to 3 files, 50 lines
```

### **Confident** 🚀
```yaml
trust_level: "confident"
# Behavior:
# - AI can auto-fix most issues
# - Auto-merge if confidence > 90%
# - Broader file access
# - Can modify up to 5 files, 100 lines
```

### **Autonomous** 🤖
```yaml
trust_level: "autonomous"
# Behavior:
# - AI operates with minimal oversight
# - Auto-fix and auto-merge most issues
# - Maximum file access (within security bounds)
# - Can modify up to 10 files, 500 lines
```

## 🎯 **Custom Trust Overrides**

Fine-tune AI behavior for specific scenarios:

```yaml
agentic:
  trust_level: "balanced"  # Default
  
  # Override trust for specific repositories
  custom_trust:
    repositories:
      - repo: "critical-payment-service"
        trust_level: "paranoid"
        require_manual_approval: true
        
      - repo: "dev-tools"
        trust_level: "autonomous"
        auto_merge_enabled: true
        
    # Override trust for file patterns
    file_patterns:
      - pattern: "*.test.js"
        trust_level: "confident"   # More liberal with test files
        
      - pattern: "database/migrations/*"
        trust_level: "paranoid"    # Very careful with DB changes
        
      - pattern: "docs/**"
        trust_level: "autonomous"  # Free rein with documentation
        
    # Override trust by event severity
    severity_overrides:
      critical: "paranoid"     # Always escalate critical issues
      high: "cautious"         # Conservative with high severity
      medium: "balanced"       # Normal behavior
      low: "confident"         # More autonomous for low severity
      
    # Override trust by environment
    environment_overrides:
      production: "cautious"   # Extra careful in production
      staging: "balanced"      # Normal in staging
      development: "confident" # More liberal in dev
```

## 🚨 **Emergency Controls**

```yaml
# Quick commands to control AI behavior
agentic:
  safety:
    # Emergency stop - disables all AI automation
    emergency_stop: false
    
    # Pause automation (can be resumed)  
    pause_automation: false
    
    # Pause on incidents (auto-pause if system issues detected)
    pause_on_incident: true
    
    # Rollback settings
    auto_rollback: true
    rollback_timeout: "5m"
```

**CLI Emergency Controls:**
```bash
# Emergency stop all AI operations
liberation-guardian trust emergency-stop

# Pause automation temporarily  
liberation-guardian trust pause

# Resume automation
liberation-guardian trust resume

# Check current status
liberation-guardian trust status
```

## 📊 **Monitoring AI Decisions**

```yaml
agentic:
  monitoring:
    # Log all AI decisions
    log_decisions: true
    log_level: "debug"
    
    # Metrics tracking
    track_metrics: true
    metrics_retention: "30d"
    
    # Decision audit trail
    audit_trail:
      enabled: true
      include_reasoning: true
      include_confidence: true
      include_context: true
      
    # Alerts for AI behavior
    alerts:
      low_confidence_threshold: 0.7
      high_failure_rate: 0.15
      unusual_patterns: true
```

## 🎛️ **Getting Started**

### **Conservative Start (Recommended)**
```yaml
agentic:
  trust_level: "cautious"
  codebase_analysis:
    enabled: true
  auto_fix:
    enabled: true
    require_review: true
    auto_merge:
      enabled: false
```

### **Gradually Increase Trust**
1. Start with `cautious` for 1-2 weeks
2. Monitor AI decisions and accuracy
3. Upgrade to `balanced` if comfortable
4. Eventually reach `confident` or `autonomous`

### **Monitor and Adjust**
- Check daily AI decision reports
- Review failed fixes and adjust confidence thresholds
- Use custom overrides for specific scenarios
- Emergency controls always available

**Liberation Philosophy: You are always in control. The AI serves you, not the other way around.**