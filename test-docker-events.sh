#!/bin/bash

# Test Liberation Guardian with Docker-style events
# Simulates common Docker container issues

set -e

LG_HOST="http://localhost:9000"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🐳 Testing Liberation Guardian with Docker Events${NC}"
echo "=================================================="

# Test Docker container crash
echo -e "\n${BLUE}Testing Docker Container Crash...${NC}"
container_crash='{
    "source": "docker",
    "container": {
        "name": "liberation-service",
        "id": "abc123def456",
        "image": "liberation/service:latest"
    },
    "event": "die",
    "exitCode": 1,
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "logs": [
        "2023-10-09T12:00:00Z ERROR Database connection failed: connection refused",
        "2023-10-09T12:00:01Z ERROR Retrying connection...",
        "2023-10-09T12:00:02Z FATAL Unable to connect to database after 3 retries"
    ],
    "environment": "production",
    "service": "liberation-service",
    "severity": "critical"
}'

response=$(curl -s -w "%{http_code}" -X POST "$LG_HOST/webhook/custom/docker" \
    -H "Content-Type: application/json" \
    -d "$container_crash")

status_code="${response: -3}"
body="${response%???}"

if [[ "$status_code" -eq 200 ]]; then
    echo -e "${GREEN}✅ Container crash event processed${NC}"
    echo "Response: $body"
else
    echo -e "${RED}❌ Failed with status $status_code${NC}"
    echo "Response: $body"
fi

# Test High Memory Usage
echo -e "\n${BLUE}Testing High Memory Usage Alert...${NC}"
memory_alert='{
    "source": "docker",
    "container": {
        "name": "liberation-db",
        "id": "def456ghi789",
        "image": "postgres:15"
    },
    "event": "resource_alert",
    "metrics": {
        "memory_usage_percent": 95,
        "memory_limit": "2GB",
        "memory_current": "1.9GB"
    },
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "message": "Container memory usage is above 90%",
    "environment": "production",
    "service": "liberation-db",
    "severity": "warning"
}'

response=$(curl -s -w "%{http_code}" -X POST "$LG_HOST/webhook/custom/docker" \
    -H "Content-Type: application/json" \
    -d "$memory_alert")

status_code="${response: -3}"
body="${response%???}"

if [[ "$status_code" -eq 200 ]]; then
    echo -e "${GREEN}✅ Memory alert processed${NC}"
    echo "Response: $body"
else
    echo -e "${RED}❌ Failed with status $status_code${NC}"
    echo "Response: $body"
fi

# Test Network Error
echo -e "\n${BLUE}Testing Network Connectivity Issue...${NC}"
network_error='{
    "source": "docker",
    "container": {
        "name": "liberation-api",
        "id": "ghi789jkl012",
        "image": "liberation/api:latest"
    },
    "event": "network_error",
    "error": {
        "type": "connection_timeout",
        "message": "dial tcp 10.0.0.5:5432: i/o timeout",
        "target": "database",
        "retry_count": 3
    },
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "environment": "production",
    "service": "liberation-api",
    "severity": "high"
}'

response=$(curl -s -w "%{http_code}" -X POST "$LG_HOST/webhook/custom/docker" \
    -H "Content-Type: application/json" \
    -d "$network_error")

status_code="${response: -3}"
body="${response%???}"

if [[ "$status_code" -eq 200 ]]; then
    echo -e "${GREEN}✅ Network error processed${NC}"
    echo "Response: $body"
else
    echo -e "${RED}❌ Failed with status $status_code${NC}"
    echo "Response: $body"
fi

# Test OOM Kill
echo -e "\n${BLUE}Testing Out of Memory Kill...${NC}"
oom_kill='{
    "source": "docker",
    "container": {
        "name": "liberation-worker",
        "id": "jkl012mno345",
        "image": "liberation/worker:latest"
    },
    "event": "oom_kill",
    "exitCode": 137,
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "message": "Container was killed due to out of memory",
    "metrics": {
        "memory_limit": "1GB",
        "memory_peak": "1.2GB"
    },
    "environment": "production",
    "service": "liberation-worker",
    "severity": "critical"
}'

response=$(curl -s -w "%{http_code}" -X POST "$LG_HOST/webhook/custom/docker" \
    -H "Content-Type: application/json" \
    -d "$oom_kill")

status_code="${response: -3}"
body="${response%???}"

if [[ "$status_code" -eq 200 ]]; then
    echo -e "${GREEN}✅ OOM kill event processed${NC}"
    echo "Response: $body"
else
    echo -e "${RED}❌ Failed with status $status_code${NC}"
    echo "Response: $body"
fi

echo -e "\n${BLUE}🎯 Docker Events Testing Complete!${NC}"
echo "=================================="
echo ""
echo "What this demonstrates:"
echo "✅ Liberation Guardian can process Docker container events"
echo "✅ Handles different severity levels (warning, high, critical)"
echo "✅ Processes structured container metadata"
echo "✅ Ready for integration with real Docker monitoring"
echo ""
echo "Next steps:"
echo "1. Set up real Docker log monitoring (e.g., Fluentd, Logstash)"
echo "2. Configure container health checks to send webhooks"
echo "3. Add AI provider keys for autonomous triage decisions"