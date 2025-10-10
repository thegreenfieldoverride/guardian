#!/bin/bash

# 🚀 Liberation Guardian - Dependabot Automation Test Script
# This script demonstrates the autonomous dependency update capabilities

set -e

echo "🤖 Liberation Guardian - Dependabot Automation Test"
echo "===================================================="

GUARDIAN_URL="http://localhost:9000"
WEBHOOK_ENDPOINT="$GUARDIAN_URL/webhook/github"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test 1: Security Update (Should auto-approve)
test_security_update() {
    print_status "Test 1: Security Update - Should AUTO-APPROVE"
    
    curl -X POST "$WEBHOOK_ENDPOINT" \
         -H "Content-Type: application/json" \
         -H "X-GitHub-Event: pull_request" \
         -H "X-GitHub-Delivery: test-security-$(date +%s)" \
         -d '{
           "action": "opened",
           "number": 123,
           "pull_request": {
             "id": 123456,
             "number": 123,
             "title": "Bump lodash from 4.17.20 to 4.17.21",
             "body": "Security update fixing CVE-2021-23337\n\nThis update fixes a critical security vulnerability.",
             "user": {
               "login": "dependabot[bot]",
               "type": "Bot"
             },
             "head": {
               "ref": "dependabot/npm/lodash-4.17.21",
               "sha": "abc123"
             },
             "base": {
               "ref": "main"
             },
             "html_url": "https://github.com/test/repo/pull/123",
             "created_at": "2023-10-09T10:00:00Z",
             "updated_at": "2023-10-09T10:00:00Z"
           },
           "repository": {
             "id": 456789,
             "name": "test-repo",
             "full_name": "testorg/test-repo",
             "owner": {
               "login": "testorg"
             }
           }
         }' 
    
    print_success "Security update test sent"
}

# Test 2: Patch Update (Should auto-approve)
test_patch_update() {
    print_status "Test 2: Patch Update - Should AUTO-APPROVE"
    
    curl -X POST "$WEBHOOK_ENDPOINT" \
         -H "Content-Type: application/json" \
         -H "X-GitHub-Event: pull_request" \
         -H "X-GitHub-Delivery: test-patch-$(date +%s)" \
         -d '{
           "action": "opened",
           "number": 124,
           "pull_request": {
             "id": 123457,
             "number": 124,
             "title": "Bump express from 4.18.0 to 4.18.1",
             "body": "Updates express from 4.18.0 to 4.18.1\n\nRelease notes and changelog available.",
             "user": {
               "login": "dependabot[bot]",
               "type": "Bot"
             },
             "head": {
               "ref": "dependabot/npm/express-4.18.1",
               "sha": "def456"
             },
             "base": {
               "ref": "main"
             },
             "html_url": "https://github.com/test/repo/pull/124",
             "created_at": "2023-10-09T10:05:00Z",
             "updated_at": "2023-10-09T10:05:00Z"
           },
           "repository": {
             "id": 456789,
             "name": "test-repo",
             "full_name": "testorg/test-repo",
             "owner": {
               "login": "testorg"
             }
           }
         }'
    
    print_success "Patch update test sent"
}

# Test 3: Major Update (Should require review)
test_major_update() {
    print_status "Test 3: Major Update - Should REQUIRE REVIEW"
    
    curl -X POST "$WEBHOOK_ENDPOINT" \
         -H "Content-Type: application/json" \
         -H "X-GitHub-Event: pull_request" \
         -H "X-GitHub-Delivery: test-major-$(date +%s)" \
         -d '{
           "action": "opened",
           "number": 125,
           "pull_request": {
             "id": 123458,
             "number": 125,
             "title": "Bump react from 17.0.2 to 18.0.0",
             "body": "Updates react from 17.0.2 to 18.0.0\n\nThis is a major version update with breaking changes.",
             "user": {
               "login": "dependabot[bot]",
               "type": "Bot"
             },
             "head": {
               "ref": "dependabot/npm/react-18.0.0",
               "sha": "ghi789"
             },
             "base": {
               "ref": "main"
             },
             "html_url": "https://github.com/test/repo/pull/125",
             "created_at": "2023-10-09T10:10:00Z",
             "updated_at": "2023-10-09T10:10:00Z"
           },
           "repository": {
             "id": 456789,
             "name": "test-repo",
             "full_name": "testorg/test-repo",
             "owner": {
               "login": "testorg"
             }
           }
         }'
    
    print_success "Major update test sent"
}

# Test 4: Unknown Package (Should analyze with AI)
test_unknown_package() {
    print_status "Test 4: Unknown Package - Should ANALYZE WITH AI"
    
    curl -X POST "$WEBHOOK_ENDPOINT" \
         -H "Content-Type: application/json" \
         -H "X-GitHub-Event: pull_request" \
         -H "X-GitHub-Delivery: test-unknown-$(date +%s)" \
         -d '{
           "action": "opened",
           "number": 126,
           "pull_request": {
             "id": 123459,
             "number": 126,
             "title": "Bump obscure-package from 1.0.0 to 1.1.0",
             "body": "Updates obscure-package from 1.0.0 to 1.1.0\n\nNew features and improvements.",
             "user": {
               "login": "dependabot[bot]",
               "type": "Bot"
             },
             "head": {
               "ref": "dependabot/npm/obscure-package-1.1.0",
               "sha": "jkl012"
             },
             "base": {
               "ref": "main"
             },
             "html_url": "https://github.com/test/repo/pull/126",
             "created_at": "2023-10-09T10:15:00Z",
             "updated_at": "2023-10-09T10:15:00Z"
           },
           "repository": {
             "id": 456789,
             "name": "test-repo", 
             "full_name": "testorg/test-repo",
             "owner": {
               "login": "testorg"
             }
           }
         }'
    
    print_success "Unknown package test sent"
}

# Check if Liberation Guardian is running
check_guardian_status() {
    print_status "Checking Liberation Guardian status..."
    
    if curl -s "$GUARDIAN_URL/health" > /dev/null; then
        print_success "Liberation Guardian is running at $GUARDIAN_URL"
    else
        print_error "Liberation Guardian is not running. Please start it first:"
        echo "  cd /path/to/liberation-guardian"
        echo "  ./liberation-guardian-dependabot"
        exit 1
    fi
}

# Display current configuration
show_config() {
    print_status "Current Dependency Automation Configuration:"
    echo "  Trust Level: BALANCED (Auto-approve patches + security)"
    echo "  Security Updates: AUTO-APPROVE ✅"
    echo "  Patch Updates: AUTO-APPROVE ✅"
    echo "  Minor Updates: REVIEW REQUIRED ⚠️"
    echo "  Major Updates: REVIEW REQUIRED ⚠️"
    echo "  AI Provider: FREE Gemini 2.0 Flash 🚀"
    echo "  Estimated Cost: $0.00 - $0.05 per PR analysis 💰"
    echo ""
}

# Run all tests
run_all_tests() {
    print_status "Running Dependabot automation tests..."
    echo ""
    
    test_security_update
    sleep 2
    
    test_patch_update
    sleep 2
    
    test_major_update
    sleep 2
    
    test_unknown_package
    sleep 2
    
    print_success "All tests completed!"
    echo ""
    print_status "Check Liberation Guardian logs to see AI decisions:"
    echo "  tail -f liberation-guardian.log"
    echo ""
    print_status "Expected Results:"
    echo "  🟢 Security Update: AUTO-APPROVED (High confidence)"
    echo "  🟢 Patch Update: AUTO-APPROVED (Safe update)"
    echo "  🟡 Major Update: COMMENT added, human review required"
    echo "  🟡 Unknown Package: AI analysis, recommendation based on risk"
}

# Display Liberation Philosophy
show_liberation_philosophy() {
    echo ""
    echo "🔥 LIBERATION GUARDIAN PHILOSOPHY 🔥"
    echo "====================================="
    echo ""
    echo "💰 COST REVOLUTION:"
    echo "  Traditional: $500-5000/month enterprise dependency management"
    echo "  Liberation: $0-25/month with superior AI analysis"
    echo ""
    echo "🤖 AI TRANSPARENCY:"
    echo "  Every decision explained with confidence scores"
    echo "  Full audit trail of automated actions"
    echo "  Human override available at any time"
    echo ""
    echo "🚀 USER SOVEREIGNTY:"
    echo "  5 trust levels from Paranoid to Autonomous"
    echo "  Custom rules for your specific needs"
    echo "  Works with ANY Git provider (no vendor lock-in)"
    echo ""
    echo "🛡️ SECURITY FIRST:"
    echo "  Security updates prioritized and auto-approved"
    echo "  Breaking change detection with AI analysis"
    echo "  Rollback plans for all automated changes"
    echo ""
}

# Main execution
main() {
    echo ""
    show_liberation_philosophy
    echo ""
    show_config
    check_guardian_status
    echo ""
    
    if [[ "$1" == "--demo" ]]; then
        run_all_tests
    else
        print_status "Run with --demo flag to execute test cases:"
        echo "  ./test-dependabot-automation.sh --demo"
        echo ""
        print_status "Or test individual scenarios:"
        echo "  Functions available: test_security_update, test_patch_update, test_major_update, test_unknown_package"
    fi
}

# Execute main function
main "$@"