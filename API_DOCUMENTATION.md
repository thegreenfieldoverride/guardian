# 🔌 Liberation Guardian API Documentation

**Complete API reference for Liberation Guardian's REST endpoints, webhooks, and integrations**

## 📋 **Table of Contents**

- [Authentication](#authentication)
- [Health & Status](#health--status)
- [Webhook Endpoints](#webhook-endpoints)
- [Management API](#management-api)
- [Dependency Management](#dependency-management)
- [AI Operations](#ai-operations)
- [Response Formats](#response-formats)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)

---

## 🔐 **Authentication**

Liberation Guardian uses multiple authentication methods depending on the endpoint:

### **Webhook Authentication**
```http
POST /webhook/github
X-GitHub-Event: pull_request
X-GitHub-Delivery: 12345-67890
X-Hub-Signature-256: sha256=...
Content-Type: application/json
```

### **API Key Authentication**
```http
GET /api/v1/status
Authorization: Bearer your-api-key
Content-Type: application/json
```

### **Environment Variables**
```bash
# GitHub Integration
GITHUB_TOKEN=ghp_your_github_token
GITHUB_WEBHOOK_SECRET=your_webhook_secret

# AI Providers
GOOGLE_API_KEY=your_gemini_api_key
ANTHROPIC_API_KEY=your_claude_api_key

# Optional Services
SENTRY_WEBHOOK_SECRET=your_sentry_secret
SLACK_WEBHOOK_URL=your_slack_webhook
```

---

## 🏥 **Health & Status**

### **Health Check**
```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2023-10-09T15:30:00Z",
  "components": {
    "redis": "healthy",
    "ai_providers": "healthy",
    "github_integration": "healthy"
  },
  "uptime_seconds": 3600
}
```

### **Readiness Check**
```http
GET /ready
```

**Response:**
```json
{
  "ready": true,
  "checks": {
    "database": true,
    "ai_client": true,
    "external_services": true
  }
}
```

### **Detailed Status**
```http
GET /api/v1/status
Authorization: Bearer your-api-key
```

**Response:**
```json
{
  "service": "liberation-guardian",
  "version": "1.0.0",
  "environment": "production",
  "uptime": "24h30m15s",
  "stats": {
    "events_processed": 1247,
    "automations_executed": 342,
    "cost_savings": 2847.50,
    "ai_requests": 892,
    "total_cost": 4.23
  },
  "trust_level": 2,
  "active_integrations": ["github", "sentry", "prometheus"]
}
```

---

## 📡 **Webhook Endpoints**

### **GitHub Webhooks**
Process GitHub events including Dependabot PRs, issues, and releases.

```http
POST /webhook/github
X-GitHub-Event: pull_request
X-GitHub-Delivery: unique-delivery-id
X-Hub-Signature-256: sha256=signature
Content-Type: application/json
```

**Supported Events:**
- `pull_request` - Dependabot PRs, manual PRs
- `push` - Code changes, releases
- `issues` - Issue creation, updates
- `release` - New releases
- `workflow_run` - CI/CD status

**Example Dependabot PR Payload:**
```json
{
  "action": "opened",
  "number": 123,
  "pull_request": {
    "id": 123456,
    "number": 123,
    "title": "Bump lodash from 4.17.20 to 4.17.21",
    "body": "Security update fixing CVE-2021-23337",
    "user": {
      "login": "dependabot[bot]",
      "type": "Bot"
    },
    "head": {
      "ref": "dependabot/npm/lodash-4.17.21",
      "sha": "abc123def456"
    },
    "base": {
      "ref": "main"
    },
    "html_url": "https://github.com/org/repo/pull/123"
  },
  "repository": {
    "id": 456789,
    "name": "my-repo",
    "full_name": "myorg/my-repo"
  }
}
```

**Response:**
```json
{
  "event_id": "evt_abc123",
  "status": "processed",
  "action_taken": "auto_approved",
  "confidence": 0.92,
  "reasoning": "Security update with high confidence and no breaking changes detected",
  "processing_time_ms": 1247,
  "cost": 0.003
}
```

### **Sentry Webhooks**
Process error and performance alerts from Sentry.

```http
POST /webhook/sentry
X-Sentry-Hook-Resource: error
X-Sentry-Hook-Timestamp: 1696857000
Content-Type: application/json
```

**Example Payload:**
```json
{
  "action": "triggered",
  "data": {
    "issue": {
      "id": "123456",
      "title": "TypeError: Cannot read property 'id' of null",
      "level": "error",
      "platform": "javascript",
      "count": 15,
      "url": "https://sentry.io/issues/123456/",
      "project": {
        "name": "my-frontend",
        "slug": "my-frontend"
      }
    }
  }
}
```

### **Prometheus Webhooks**
Process alerts from Prometheus Alertmanager.

```http
POST /webhook/prometheus
Content-Type: application/json
```

**Example Payload:**
```json
{
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "HighMemoryUsage",
        "instance": "web-server-1:8080",
        "severity": "warning"
      },
      "annotations": {
        "summary": "High memory usage detected",
        "description": "Memory usage is above 90% for 5 minutes"
      },
      "startsAt": "2023-10-09T15:30:00Z"
    }
  ]
}
```

---

## ⚙️ **Management API**

### **Update Trust Level**
```http
PUT /api/v1/config/trust-level
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "trust_level": 3,
  "reason": "Increasing automation for faster deployments"
}
```

**Response:**
```json
{
  "success": true,
  "previous_level": 2,
  "new_level": 3,
  "description": "PROGRESSIVE: Auto-approve most updates with high confidence",
  "updated_at": "2023-10-09T15:30:00Z"
}
```

### **Get Configuration**
```http
GET /api/v1/config
Authorization: Bearer your-api-key
```

**Response:**
```json
{
  "trust_level": 2,
  "trust_description": "BALANCED: Auto-approve patch + minor security updates",
  "dependencies": {
    "security_auto_approve": true,
    "patch_auto_approve": true,
    "minor_auto_approve": false,
    "major_auto_approve": false,
    "min_confidence": 0.80,
    "min_test_coverage": 0.70
  },
  "ai_providers": {
    "primary": "google/gemini-2.0-flash",
    "fallback": "anthropic/claude-3-5-haiku",
    "local": "ollama/qwen2.5:7b"
  },
  "integrations": {
    "github": {"enabled": true, "auto_merge": true},
    "sentry": {"enabled": true, "auto_acknowledge": false},
    "prometheus": {"enabled": true}
  }
}
```

### **Add Custom Rule**
```http
POST /api/v1/config/rules
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "name": "Critical Security Updates",
  "pattern": ".*",
  "update_type": "security",
  "action": "approve",
  "conditions": {
    "severity": ["high", "critical"]
  },
  "description": "Auto-approve all critical security updates"
}
```

### **List Events**
```http
GET /api/v1/events?limit=50&offset=0&type=dependency_update
Authorization: Bearer your-api-key
```

**Response:**
```json
{
  "events": [
    {
      "id": "evt_abc123",
      "type": "dependency_update",
      "source": "github",
      "severity": "high",
      "title": "Security update: lodash 4.17.21",
      "timestamp": "2023-10-09T15:30:00Z",
      "status": "processed",
      "automation_result": {
        "action": "auto_approved",
        "confidence": 0.92,
        "cost": 0.003
      }
    }
  ],
  "pagination": {
    "total": 156,
    "limit": 50,
    "offset": 0,
    "has_more": true
  }
}
```

---

## 📦 **Dependency Management**

### **Analyze Dependency Update**
Manually trigger analysis of a dependency update.

```http
POST /api/v1/dependencies/analyze
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "package_name": "lodash",
  "current_version": "4.17.20",
  "new_version": "4.17.21",
  "ecosystem": "npm",
  "repository": "myorg/my-repo",
  "update_type": "patch",
  "cve_fixed": ["CVE-2021-23337"],
  "changelog": "Security fixes and performance improvements"
}
```

**Response:**
```json
{
  "analysis_id": "ana_xyz789",
  "recommendation": "approve",
  "confidence": 0.94,
  "security_impact": "high",
  "breaking_changes": false,
  "reasoning": "Security update fixing critical vulnerability with no breaking changes detected",
  "risk_factors": ["security_vulnerabilities_fixed", "popular_package"],
  "community_metrics": {
    "weekly_downloads": 50000000,
    "github_stars": 55000,
    "test_coverage": 0.95
  },
  "auto_fix_suggestion": {
    "type": "dependency_update",
    "description": "Update lodash from 4.17.20 to 4.17.21",
    "estimated_time_minutes": 5,
    "requires_approval": false
  },
  "processing_time_ms": 892,
  "cost": 0.004,
  "ai_provider": "google/gemini-2.0-flash"
}
```

### **Get Dependency Statistics**
```http
GET /api/v1/dependencies/stats
Authorization: Bearer your-api-key
```

**Response:**
```json
{
  "total_prs_processed": 342,
  "auto_approved": 258,
  "auto_merged": 186,
  "human_review_required": 71,
  "rejected": 13,
  "average_confidence": 0.87,
  "average_cost_per_pr": 0.003,
  "total_cost_savings": 5240.00,
  "security_updates_fixed": 89,
  "breaking_changes_detected": 24,
  "ecosystems": {
    "npm": 156,
    "pip": 87,
    "go_modules": 65,
    "cargo": 34
  }
}
```

---

## 🤖 **AI Operations**

### **Get AI Provider Status**
```http
GET /api/v1/ai/status
Authorization: Bearer your-api-key
```

**Response:**
```json
{
  "providers": {
    "google": {
      "status": "healthy",
      "model": "gemini-2.0-flash",
      "requests_today": 234,
      "cost_today": 0.00,
      "rate_limit": {
        "remaining": 4766,
        "reset_at": "2023-10-09T16:00:00Z"
      }
    },
    "anthropic": {
      "status": "healthy",
      "model": "claude-3-5-haiku",
      "requests_today": 12,
      "cost_today": 0.024
    },
    "ollama": {
      "status": "healthy",
      "model": "qwen2.5:7b",
      "requests_today": 45,
      "cost_today": 0.00
    }
  },
  "total_cost_today": 0.024,
  "total_requests_today": 291
}
```

### **Test AI Provider**
```http
POST /api/v1/ai/test
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "provider": "google",
  "prompt": "Analyze this dependency update for security risks: lodash 4.17.20 to 4.17.21"
}
```

**Response:**
```json
{
  "success": true,
  "response": "This is a security update that fixes CVE-2021-23337...",
  "tokens_used": 156,
  "cost": 0.002,
  "processing_time_ms": 743,
  "provider": "google/gemini-2.0-flash"
}
```

---

## 📄 **Response Formats**

### **Standard API Response**
```json
{
  "success": true,
  "data": {...},
  "metadata": {
    "timestamp": "2023-10-09T15:30:00Z",
    "request_id": "req_abc123",
    "processing_time_ms": 247
  }
}
```

### **Error Response**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Trust level must be between 0 and 4",
    "details": {
      "field": "trust_level",
      "provided": 5,
      "valid_range": [0, 4]
    }
  },
  "metadata": {
    "timestamp": "2023-10-09T15:30:00Z",
    "request_id": "req_def456"
  }
}
```

### **Webhook Response**
```json
{
  "event_id": "evt_abc123",
  "status": "processed|ignored|error",
  "action_taken": "auto_approved|commented|escalated|none",
  "confidence": 0.92,
  "reasoning": "Detailed explanation of the decision",
  "processing_time_ms": 1247,
  "cost": 0.003,
  "metadata": {
    "trust_level": 2,
    "ai_provider": "google/gemini-2.0-flash"
  }
}
```

---

## ⚠️ **Error Handling**

### **HTTP Status Codes**

| Code | Meaning | Description |
|------|---------|-------------|
| 200 | OK | Request successful |
| 201 | Created | Resource created successfully |
| 400 | Bad Request | Invalid request parameters |
| 401 | Unauthorized | Authentication required |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource not found |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server error |
| 503 | Service Unavailable | Service temporarily unavailable |

### **Error Codes**

| Code | Description |
|------|-------------|
| `INVALID_REQUEST` | Request validation failed |
| `AUTHENTICATION_FAILED` | Invalid credentials |
| `RATE_LIMIT_EXCEEDED` | Too many requests |
| `AI_PROVIDER_ERROR` | AI service unavailable |
| `CONFIGURATION_ERROR` | Invalid configuration |
| `WEBHOOK_VERIFICATION_FAILED` | Webhook signature invalid |
| `DEPENDENCY_ANALYSIS_FAILED` | Unable to analyze dependency |

### **Error Response Example**
```json
{
  "success": false,
  "error": {
    "code": "AI_PROVIDER_ERROR",
    "message": "Primary AI provider is unavailable, falling back to secondary provider",
    "details": {
      "primary_provider": "google/gemini-2.0-flash",
      "fallback_provider": "anthropic/claude-3-5-haiku",
      "retry_after": 300
    },
    "recoverable": true
  },
  "metadata": {
    "timestamp": "2023-10-09T15:30:00Z",
    "request_id": "req_error123"
  }
}
```

---

## 🚦 **Rate Limiting**

Liberation Guardian implements intelligent rate limiting to protect against abuse while allowing legitimate usage.

### **Rate Limits**

| Endpoint Category | Limit | Window |
|------------------|-------|---------|
| Health checks | 1000/hour | Per IP |
| Webhooks | 10000/hour | Per source |
| Management API | 100/hour | Per API key |
| AI operations | 1000/day | Per API key |

### **Rate Limit Headers**
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 73
X-RateLimit-Reset: 1696857600
X-RateLimit-Window: 3600
```

### **Rate Limit Exceeded Response**
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded for this endpoint",
    "details": {
      "limit": 100,
      "window_seconds": 3600,
      "retry_after": 247
    }
  }
}
```

---

## 🔧 **SDK & Integration Examples**

### **JavaScript/Node.js**
```javascript
const { LiberationGuardian } = require('@liberation-guardian/sdk');

const guardian = new LiberationGuardian({
  apiKey: process.env.LIBERATION_GUARDIAN_API_KEY,
  baseUrl: 'https://your-guardian.example.com'
});

// Analyze a dependency update
const analysis = await guardian.dependencies.analyze({
  packageName: 'lodash',
  currentVersion: '4.17.20',
  newVersion: '4.17.21',
  ecosystem: 'npm'
});

console.log('Recommendation:', analysis.recommendation);
console.log('Confidence:', analysis.confidence);
```

### **Python**
```python
from liberation_guardian import LiberationGuardian

guardian = LiberationGuardian(
    api_key=os.environ['LIBERATION_GUARDIAN_API_KEY'],
    base_url='https://your-guardian.example.com'
)

# Get current status
status = guardian.status.get()
print(f"Trust Level: {status.trust_level}")
print(f"Events Processed: {status.stats.events_processed}")
```

### **curl Examples**
```bash
# Health check
curl -X GET https://your-guardian.example.com/health

# Get status with authentication
curl -X GET https://your-guardian.example.com/api/v1/status \
  -H "Authorization: Bearer your-api-key"

# Update trust level
curl -X PUT https://your-guardian.example.com/api/v1/config/trust-level \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"trust_level": 3, "reason": "Increasing automation"}'
```

---

## 📚 **Additional Resources**

- [**Setup Guide**](DEPLOYMENT_GUIDE.md) - Step-by-step deployment instructions
- [**Configuration Reference**](CONFIGURATION_GUIDE.md) - Complete configuration options
- [**Webhook Security**](WEBHOOK_SECURITY.md) - Securing webhook endpoints
- [**Troubleshooting**](TROUBLESHOOTING_GUIDE.md) - Common issues and solutions

---

**Need help?** Email [support@greenfieldoverride.com](mailto:support@greenfieldoverride.com) or check the [GitHub issues](https://github.com/greenfieldoverride/liberation-guardian/issues).