#!/bin/bash

# Comprehensive Agentic Liberation Guardian Testing
# Tests codebase analysis with various language stack traces and errors

echo "🧪 Agentic Liberation Guardian Stress Testing"
echo "============================================="

BASE_URL="http://localhost:9000/webhook/custom/test"

# Test 1: Go panic (real Liberation Guardian code)
echo "1. Testing Go panic with real codebase..."
curl -X POST $BASE_URL -H "Content-Type: application/json" -d '{
  "event_type": "panic",
  "severity": "critical",
  "message": "panic: runtime error: invalid memory address or nil pointer dereference",
  "description": "goroutine 1 [running]: liberation-guardian/internal/ai.(*TriageEngine).performAITriage(triage.go:247) +0x123",
  "service": "liberation-guardian",
  "environment": "production"
}' && echo

sleep 3

# Test 2: TypeScript error (your ai-integration service)
echo "2. Testing TypeScript error with service context..."
curl -X POST $BASE_URL -H "Content-Type: application/json" -d '{
  "event_type": "TypeError",
  "severity": "high", 
  "message": "Cannot read property \"id\" of undefined",
  "description": "TypeError: Cannot read property \"id\" of undefined\n    at AnthropicProvider.sendRequest (/app/src/providers/anthropic.ts:42:15)\n    at async AIManager.processRequest (/app/src/manager.ts:156:23)",
  "service": "ai-integration",
  "environment": "production"
}' && echo

sleep 3

# Test 3: Database connection error (likely PostgreSQL)
echo "3. Testing database connection error..."
curl -X POST $BASE_URL -H "Content-Type: application/json" -d '{
  "event_type": "database_error",
  "severity": "critical",
  "message": "connection to server at \"localhost\" (127.0.0.1), port 5432 failed",
  "description": "Error: connect ECONNREFUSED 127.0.0.1:5432\n    at Connection.connect (/app/services/core-api/src/database/connection.ts:23:12)",
  "service": "core-api", 
  "environment": "production"
}' && echo

sleep 3

# Test 4: Redis connection failure
echo "4. Testing Redis connection failure..."
curl -X POST $BASE_URL -H "Content-Type: application/json" -d '{
  "event_type": "redis_error",
  "severity": "high",
  "message": "Redis connection lost",
  "description": "Error: Redis connection to 127.0.0.1:6379 failed - connect ECONNREFUSED 127.0.0.1:6379\n    at TaskQueue.connect (/app/services/event-bus/src/core/task-queue.ts:89:17)",
  "service": "event-bus",
  "environment": "staging"
}' && echo

sleep 3

# Test 5: Docker container crash
echo "5. Testing Docker container crash..."
curl -X POST $BASE_URL -H "Content-Type: application/json" -d '{
  "event_type": "container_crash",
  "severity": "critical",
  "message": "Container ai-integration exited with code 1",
  "description": "Container ai-integration crashed: OOMKilled\nLast logs: Error in /app/src/providers/openai.ts:156\nOut of memory exception at line 156",
  "service": "ai-integration",
  "environment": "production"
}' && echo

sleep 3

# Test 6: NPM dependency vulnerability  
echo "6. Testing dependency vulnerability..."
curl -X POST $BASE_URL -H "Content-Type: application/json" -d '{
  "event_type": "security_vulnerability", 
  "severity": "high",
  "message": "High severity vulnerability in lodash@4.17.19",
  "description": "CVE-2021-23337: Command injection in lodash\nAffected files: /app/package.json, /app/services/*/package.json\nVulnerable function: _.template() in multiple files",
  "service": "multiple",
  "environment": "all"
}' && echo

sleep 3

# Test 7: Go build failure
echo "7. Testing Go build failure..."
curl -X POST $BASE_URL -H "Content-Type: application/json" -d '{
  "event_type": "build_failure",
  "severity": "medium",
  "message": "Go build failed: undefined: types.AutoFixPlan",
  "description": "./internal/ai/triage.go:192:20: undefined: types.AutoFixPlan\n./internal/events/processor.go:168:94: unused parameter: result\nBuild failed at go build ./cmd/main.go",
  "service": "liberation-guardian",
  "environment": "development"
}' && echo

sleep 3

# Test 8: Python error (in case you have any Python scripts)
echo "8. Testing Python error..."
curl -X POST $BASE_URL -H "Content-Type: application/json" -d '{
  "event_type": "AttributeError",
  "severity": "medium", 
  "message": "module has no attribute \"parse_config\"",
  "description": "Traceback (most recent call last):\n  File \"/app/scripts/migrate.py\", line 23, in <module>\n    config = yaml.parse_config(config_file)\nAttributeError: module \"yaml\" has no attribute \"parse_config\"",
  "service": "migration-scripts",
  "environment": "development"
}' && echo

sleep 3

# Test 9: JWT/Auth error
echo "9. Testing authentication error..."
curl -X POST $BASE_URL -H "Content-Type: application/json" -d '{
  "event_type": "auth_error",
  "severity": "high",
  "message": "JWT token validation failed",
  "description": "JsonWebTokenError: invalid signature\n    at validateToken (/app/services/core-api/src/routes/auth.ts:67:15)\n    at middleware (/app/services/core-api/src/routes/auth.ts:23:21)",
  "service": "core-api",
  "environment": "production"
}' && echo

sleep 3

# Test 10: Memory leak detection
echo "10. Testing memory leak detection..."
curl -X POST $BASE_URL -H "Content-Type: application/json" -d '{
  "event_type": "memory_leak",
  "severity": "high",
  "message": "Memory usage exceeded 90% threshold",
  "description": "Potential memory leak detected in ai-integration service\nHeap usage: 1.2GB / 1.4GB (85%)\nSuspected leak in /app/src/providers/anthropic.ts around line 89\nGC pressure high, allocation rate: 50MB/s",
  "service": "ai-integration", 
  "environment": "production"
}' && echo

sleep 5

echo ""
echo "🎯 All tests sent! Check liberation-guardian.log for AI analysis results:"
echo "tail -f liberation-guardian.log | grep -E '(AI request completed|codebase analysis|files analyzed|patterns detected)'"
echo ""
echo "Expected behavior:"
echo "✅ AI should analyze relevant source files for each error"
echo "✅ AI should provide enhanced context with actual code understanding"
echo "✅ AI should detect error patterns and suggest fixes"
echo "✅ Different trust levels should produce different response types"