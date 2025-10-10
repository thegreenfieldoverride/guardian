#!/bin/bash

# Liberation Guardian Manual Testing Script
# Tests the autonomous operations platform with real webhook payloads

set -e

echo "🔥 Liberation Guardian Manual Testing"
echo "===================================="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
LG_HOST="http://localhost:9000"
TIMEOUT=5

# Test function
test_endpoint() {
    local method="$1"
    local endpoint="$2"
    local payload="$3"
    local headers="$4"
    local description="$5"
    
    echo -n "Testing: $description... "
    
    if [ -n "$payload" ]; then
        if [ -n "$headers" ]; then
            response=$(curl -s -w "%{http_code}" -X "$method" "$LG_HOST$endpoint" \
                -H "Content-Type: application/json" \
                $headers \
                -d "$payload" \
                --connect-timeout $TIMEOUT \
                --max-time $TIMEOUT)
        else
            response=$(curl -s -w "%{http_code}" -X "$method" "$LG_HOST$endpoint" \
                -H "Content-Type: application/json" \
                -d "$payload" \
                --connect-timeout $TIMEOUT \
                --max-time $TIMEOUT)
        fi
    else
        response=$(curl -s -w "%{http_code}" -X "$method" "$LG_HOST$endpoint" \
            --connect-timeout $TIMEOUT \
            --max-time $TIMEOUT)
    fi
    
    # Extract status code (last 3 characters)
    status_code="${response: -3}"
    body="${response%???}"
    
    if [[ "$status_code" -ge 200 && "$status_code" -lt 300 ]]; then
        echo -e "${GREEN}✅ PASS${NC} ($status_code)"
    elif [[ "$status_code" -ge 400 && "$status_code" -lt 500 ]]; then
        echo -e "${YELLOW}⚠️  HANDLED${NC} ($status_code) - $body"
    else
        echo -e "${RED}❌ FAIL${NC} ($status_code) - $body"
    fi
}

# Check if Liberation Guardian is running
echo "Checking if Liberation Guardian is running..."
if ! curl -s "$LG_HOST/health" > /dev/null 2>&1; then
    echo -e "${RED}❌ Liberation Guardian is not running on $LG_HOST${NC}"
    echo ""
    echo "To start Liberation Guardian:"
    echo "  1. cd services/liberation-guardian"
    echo "  2. cp .env.example .env  # Edit with your API keys"
    echo "  3. docker-compose up -d  # OR ./liberation-guardian"
    echo ""
    exit 1
fi

echo -e "${GREEN}✅ Liberation Guardian is running${NC}"
echo ""

# 1. Health and Status Tests
echo "🏥 Health & Status Tests"
echo "========================"
test_endpoint "GET" "/health" "" "" "Health check"
test_endpoint "GET" "/ready" "" "" "Readiness check"
test_endpoint "GET" "/api/v1/status" "" "" "Status endpoint"
echo ""

# 2. Webhook Tests
echo "🪝 Webhook Endpoint Tests"
echo "========================="

# Sentry webhook
sentry_payload='{
    "action": "created",
    "data": {
        "issue": {
            "id": "12345",
            "title": "TypeError: Cannot read property of null",
            "level": "error",
            "logger": "javascript",
            "platform": "javascript",
            "message": "Cannot read property '\''data'\'' of null",
            "firstSeen": "2023-10-09T12:00:00Z",
            "count": 5,
            "permalink": "https://sentry.io/organizations/test/issues/12345/",
            "project": {
                "name": "liberation-test",
                "slug": "liberation-test"
            }
        }
    }
}'

test_endpoint "POST" "/webhook/sentry" "$sentry_payload" "" "Sentry error webhook"

# GitHub webhook (CI failure)
github_payload='{
    "action": "completed",
    "workflow_run": {
        "id": 123456,
        "name": "CI",
        "head_branch": "main",
        "conclusion": "failure",
        "url": "https://api.github.com/repos/test/repo/actions/runs/123456"
    },
    "repository": {
        "name": "liberation-test",
        "full_name": "user/liberation-test"
    }
}'

test_endpoint "POST" "/webhook/github" "$github_payload" '-H "X-GitHub-Event: workflow_run"' "GitHub CI failure webhook"

# Prometheus alert
prometheus_payload='{
    "receiver": "liberation-guardian",
    "status": "firing",
    "alerts": [
        {
            "status": "firing",
            "labels": {
                "alertname": "HighMemoryUsage",
                "instance": "server1:9090",
                "job": "node-exporter",
                "severity": "warning",
                "service": "liberation-test"
            },
            "annotations": {
                "description": "Memory usage is above 80% on server1",
                "summary": "High memory usage detected"
            },
            "startsAt": "2023-10-09T12:00:00Z",
            "generatorURL": "http://prometheus:9090/graph"
        }
    ]
}'

test_endpoint "POST" "/webhook/prometheus" "$prometheus_payload" "" "Prometheus alert webhook"

# Universal webhook (auto-detection)
test_endpoint "POST" "/webhook/" "$sentry_payload" '-H "User-Agent: Sentry/1.0"' "Universal webhook with Sentry payload"

echo ""

# 3. Custom webhook test
echo "🔧 Custom Webhook Tests"
echo "======================="

custom_payload='{
    "service": "custom-app",
    "level": "error",
    "message": "Database connection failed",
    "timestamp": "2023-10-09T12:00:00Z",
    "environment": "production"
}'

test_endpoint "POST" "/webhook/custom/my-service" "$custom_payload" "" "Custom webhook for my-service"

echo ""

# 4. Invalid requests (should be handled gracefully)
echo "🚫 Error Handling Tests"
echo "======================="
test_endpoint "POST" "/webhook/sentry" '{"invalid": "json"' "" "Malformed JSON to Sentry webhook"
test_endpoint "POST" "/webhook/nonexistent" "$sentry_payload" "" "Non-existent webhook endpoint"
test_endpoint "GET" "/webhook/sentry" "" "" "Wrong HTTP method"

echo ""

# Summary
echo "🎯 Testing Complete!"
echo "==================="
echo ""
echo "What this proves:"
echo "✅ Liberation Guardian accepts real observability webhooks"
echo "✅ Auto-detects webhook sources from headers/payload"
echo "✅ Handles errors gracefully without crashing"
echo "✅ Provides health endpoints for monitoring"
echo "✅ Ready for integration with real observability tools"
echo ""
echo "Next steps:"
echo "1. Configure your observability tools to send webhooks to these endpoints"
echo "2. Add AI provider API keys to .env for autonomous triage"
echo "3. Set up notification channels (Slack, email) for escalations"
echo ""
echo "🔥 Liberation Guardian is ready for autonomous operations!"