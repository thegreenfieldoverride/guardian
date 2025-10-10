#!/bin/bash

# Liberation Guardian Integration Script
# Adds Liberation Guardian to your existing Liberation stack

set -e

echo "🚀 Liberation Guardian Integration Script"
echo "=========================================="

# Check if we're in the right directory
if [[ ! -f "docker-compose.yml" ]]; then
    echo "❌ Error: docker-compose.yml not found. Please run from your Liberation stack root."
    exit 1
fi

# Check if Liberation Guardian exists
if [[ ! -d "services/liberation-guardian" ]]; then
    echo "❌ Error: Liberation Guardian not found. Please ensure it's in services/liberation-guardian/"
    exit 1
fi

echo "✅ Found Liberation stack and Liberation Guardian"

# Backup existing docker-compose.yml
cp docker-compose.yml docker-compose.yml.backup
echo "✅ Backed up existing docker-compose.yml"

# Add Liberation Guardian to docker-compose.yml
echo ""
echo "📝 Adding Liberation Guardian to docker-compose.yml..."

cat >> docker-compose.yml << 'EOF'

  # Liberation Guardian - Autonomous AI Operations
  liberation-guardian:
    build: ./services/liberation-guardian
    ports:
      - "9000:9000"
    environment:
      - GOOGLE_API_KEY=${GOOGLE_API_KEY}
      - ANTHROPIC_API_KEY=${DEFAULT_ANTHROPIC_API_KEY}
      - REDIS_HOST=${REDIS_HOST:-redis}
      - REDIS_PORT=${REDIS_PORT:-6379}
      - REDIS_DB=1
      - ENVIRONMENT=${NODE_ENV:-development}
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - PORT=9000
    volumes:
      - ./services/liberation-guardian/liberation-guardian.yml:/app/liberation-guardian.yml
    depends_on:
      - redis
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:9000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - default
EOF

echo "✅ Added Liberation Guardian to docker-compose.yml"

# Update .env file with Liberation Guardian variables
echo ""
echo "📝 Updating .env file..."

# Add Liberation Guardian section to .env if not exists
if ! grep -q "# Liberation Guardian" .env 2>/dev/null; then
    cat >> .env << 'EOF'

# Liberation Guardian Configuration
GOOGLE_API_KEY=your_google_gemini_api_key_here
# Optional: Additional AI providers
# ANTHROPIC_API_KEY=your_anthropic_api_key_here
# OPENAI_API_KEY=your_openai_api_key_here

# Liberation Guardian Webhooks (optional)
# SLACK_WEBHOOK_URL=your_slack_webhook_url
# SENTRY_WEBHOOK_SECRET=your_sentry_webhook_secret
# GITHUB_WEBHOOK_SECRET=your_github_webhook_secret
EOF
    echo "✅ Added Liberation Guardian configuration to .env"
else
    echo "✅ Liberation Guardian configuration already exists in .env"
fi

# Create monitoring configuration
echo ""
echo "📝 Setting up monitoring configuration..."

mkdir -p monitoring
cat > monitoring/prometheus.yml << 'EOF'
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'liberation-guardian'
    static_configs:
      - targets: ['liberation-guardian:9000']
    metrics_path: '/metrics'
    
  - job_name: 'core-api'
    static_configs:
      - targets: ['core-api:3000']
    metrics_path: '/metrics'
    
  - job_name: 'ai-integration'
    static_configs:
      - targets: ['ai-integration:3001']
    metrics_path: '/metrics'
    
  - job_name: 'event-bus'
    static_configs:
      - targets: ['event-bus:3002']
    metrics_path: '/metrics'

# Alerting rules for Liberation Guardian
rule_files:
  - "alert_rules.yml"

# Send alerts to Liberation Guardian
alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - liberation-guardian:9000
      scheme: http
      api_version: v1
      timeout: 30s
EOF

cat > monitoring/alert_rules.yml << 'EOF'
groups:
  - name: liberation_stack_alerts
    rules:
    - alert: HighCPUUsage
      expr: rate(cpu_usage_seconds_total[5m]) > 0.8
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High CPU usage detected"
        description: "CPU usage is above 80% for 5 minutes"
        
    - alert: HighMemoryUsage
      expr: memory_usage_percent > 85
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High memory usage detected"
        description: "Memory usage is above 85% for 5 minutes"
        
    - alert: ServiceDown
      expr: up == 0
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "Service is down"
        description: "{{ $labels.instance }} has been down for more than 1 minute"
EOF

echo "✅ Created monitoring configuration"

# Build and start Liberation Guardian
echo ""
echo "🔨 Building Liberation Guardian..."
cd services/liberation-guardian
if [[ ! -f liberation-guardian ]]; then
    go build -o liberation-guardian ./cmd/main.go
fi
cd ../..

echo "✅ Liberation Guardian built successfully"

# Instructions
echo ""
echo "🎉 Liberation Guardian Integration Complete!"
echo "==========================================="
echo ""
echo "Next steps:"
echo "1. Set your Google API key in .env:"
echo "   GOOGLE_API_KEY=your_actual_api_key_here"
echo ""
echo "2. Start your Liberation stack with Guardian:"
echo "   docker-compose up -d liberation-guardian"
echo ""
echo "3. Verify Liberation Guardian is running:"
echo "   curl http://localhost:9000/health"
echo ""
echo "4. Send a test webhook:"
echo "   curl -X POST http://localhost:9000/webhook/custom/test \\"
echo "        -H 'Content-Type: application/json' \\"
echo "        -d '{\"event_type\":\"test\",\"message\":\"Integration test\"}'"
echo ""
echo "5. Configure your monitoring tools to send webhooks to:"
echo "   - Prometheus: http://localhost:9000/webhook/prometheus"
echo "   - Grafana: http://localhost:9000/webhook/grafana"
echo "   - Sentry: http://localhost:9000/webhook/sentry"
echo "   - GitHub: http://localhost:9000/webhook/github"
echo "   - Custom: http://localhost:9000/webhook/custom/[source]"
echo ""
echo "🚀 Liberation Guardian will now provide autonomous AI operations for your stack!"
echo ""
echo "Backup files:"
echo "- Original docker-compose.yml saved as docker-compose.yml.backup"
echo ""
echo "For more info: https://github.com/liberation/guardian/blob/main/DEPLOYMENT_GUIDE.md"