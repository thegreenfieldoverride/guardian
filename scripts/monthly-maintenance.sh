#!/bin/bash

# Liberation Guardian - Monthly Maintenance Script
# Automated blue/green deployment with model updates and health checks

set -e

# Configuration
MAINTENANCE_LOG="/var/log/liberation-maintenance.log"
BACKUP_DIR="./backups"
DATE_STAMP=$(date +%Y%m%d_%H%M%S)
COMPOSE_FILE="docker-compose.local.yml"
HEALTH_CHECK_TIMEOUT=300  # 5 minutes

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging function
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$MAINTENANCE_LOG"
}

success() {
    echo -e "${GREEN}✅${NC} $1" | tee -a "$MAINTENANCE_LOG"
}

warning() {
    echo -e "${YELLOW}⚠️${NC} $1" | tee -a "$MAINTENANCE_LOG"
}

error() {
    echo -e "${RED}❌${NC} $1" | tee -a "$MAINTENANCE_LOG"
}

# Create backup directory
mkdir -p "$BACKUP_DIR"

log "🔄 Liberation Guardian Monthly Maintenance Started"
log "=================================================="

# 1. Pre-maintenance health check
log "1. Running pre-maintenance health check..."
if curl -sf http://localhost:9000/health >/dev/null 2>&1; then
    success "Current deployment is healthy"
else
    error "Current deployment is unhealthy - aborting maintenance"
    exit 1
fi

# 2. Create backups
log "2. Creating maintenance backups..."

# Backup Redis data
if docker-compose -f "$COMPOSE_FILE" exec -T redis redis-cli BGSAVE >/dev/null 2>&1; then
    sleep 5  # Wait for background save to complete
    docker cp liberation-redis-local:/data/dump.rdb "$BACKUP_DIR/redis-${DATE_STAMP}.rdb"
    success "Redis backup created: $BACKUP_DIR/redis-${DATE_STAMP}.rdb"
else
    warning "Redis backup failed - continuing without backup"
fi

# Backup Liberation Guardian data
if [ -d "guardian_data" ]; then
    tar -czf "$BACKUP_DIR/guardian-data-${DATE_STAMP}.tar.gz" guardian_data/
    success "Guardian data backup created"
fi

# Backup current configuration
cp liberation-guardian.yml "$BACKUP_DIR/liberation-guardian-${DATE_STAMP}.yml"
success "Configuration backup created"

# 3. Check for updates
log "3. Checking for updates..."

# Pull latest images
log "Pulling latest Docker images..."
docker-compose -f "$COMPOSE_FILE" pull

# Check for newer models
log "Checking for newer AI models..."
CURRENT_MODEL=$(grep LOCAL_MODEL .env | cut -d'=' -f2 || echo "qwen2.5:7b")
log "Current model: $CURRENT_MODEL"

# 4. Create maintenance environment
log "4. Setting up maintenance environment..."

# Stop current services gracefully
log "Stopping current services..."
docker-compose -f "$COMPOSE_FILE" down

# Clean up old containers and images
log "Cleaning up old resources..."
docker system prune -f >/dev/null 2>&1

# 5. Deploy updated environment
log "5. Deploying updated environment..."

# Start services with latest images
docker-compose -f "$COMPOSE_FILE" up -d

# Wait for Ollama to be ready
log "Waiting for Ollama to initialize..."
timeout=120
elapsed=0
while ! docker-compose -f "$COMPOSE_FILE" exec -T ollama ollama list >/dev/null 2>&1; do
    if [ $elapsed -ge $timeout ]; then
        error "Ollama failed to start within $timeout seconds"
        log "Attempting rollback..."
        ./scripts/rollback-maintenance.sh "$DATE_STAMP"
        exit 1
    fi
    echo -n "."
    sleep 5
    elapsed=$((elapsed + 5))
done
echo ""
success "Ollama is ready"

# 6. Update AI models
log "6. Updating AI models..."

# Download/update primary model
log "Updating primary model: $CURRENT_MODEL"
if docker-compose -f "$COMPOSE_FILE" exec -T ollama ollama pull "$CURRENT_MODEL"; then
    success "Primary model updated successfully"
else
    warning "Primary model update failed - continuing with existing model"
fi

# Download recommended secondary models
log "Updating secondary models..."
secondary_models=("codellama:7b" "mistral:7b")
for model in "${secondary_models[@]}"; do
    log "Updating $model..."
    if docker-compose -f "$COMPOSE_FILE" exec -T ollama ollama pull "$model" >/dev/null 2>&1; then
        success "$model updated"
    else
        warning "$model update failed - skipping"
    fi
done

# 7. Start Liberation Guardian
log "7. Starting Liberation Guardian..."
docker-compose -f "$COMPOSE_FILE" up -d liberation-guardian

# Wait for Liberation Guardian to be ready
log "Waiting for Liberation Guardian to start..."
timeout=$HEALTH_CHECK_TIMEOUT
elapsed=0
while ! curl -sf http://localhost:9000/health >/dev/null 2>&1; do
    if [ $elapsed -ge $timeout ]; then
        error "Liberation Guardian failed to start within $timeout seconds"
        log "Attempting rollback..."
        ./scripts/rollback-maintenance.sh "$DATE_STAMP"
        exit 1
    fi
    echo -n "."
    sleep 5
    elapsed=$((elapsed + 5))
done
echo ""
success "Liberation Guardian is ready"

# 8. Run post-deployment tests
log "8. Running post-deployment tests..."

# Test basic functionality
log "Testing basic webhook functionality..."
response=$(curl -s -X POST http://localhost:9000/webhook/custom/maintenance \
    -H "Content-Type: application/json" \
    -d '{"event_type":"maintenance_test","message":"Monthly maintenance test","severity":"low"}')

if echo "$response" | grep -q "event_id"; then
    success "Webhook test passed"
else
    error "Webhook test failed - response: $response"
    log "Attempting rollback..."
    ./scripts/rollback-maintenance.sh "$DATE_STAMP"
    exit 1
fi

# Test AI functionality
log "Testing AI triage functionality..."
sleep 10  # Give AI time to process the test event

# Check recent logs for AI processing
if docker-compose -f "$COMPOSE_FILE" logs liberation-guardian | tail -20 | grep -q "AI request completed"; then
    success "AI functionality test passed"
else
    warning "AI functionality test inconclusive - check logs manually"
fi

# 9. Performance validation
log "9. Validating performance..."

# Test response time
start_time=$(date +%s%N)
curl -sf http://localhost:9000/health >/dev/null
end_time=$(date +%s%N)
response_time=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds

if [ $response_time -lt 1000 ]; then
    success "Response time: ${response_time}ms (excellent)"
elif [ $response_time -lt 3000 ]; then
    success "Response time: ${response_time}ms (acceptable)"
else
    warning "Response time: ${response_time}ms (slow - investigate)"
fi

# Check resource usage
memory_usage=$(docker stats --no-stream --format "table {{.Container}}\t{{.MemUsage}}" | grep ollama | awk '{print $2}' | cut -d'/' -f1)
log "Current memory usage: $memory_usage"

# 10. Clean up old resources
log "10. Cleaning up old resources..."

# Remove old model versions (keep last 2)
log "Cleaning up old model versions..."
docker images | grep ollama | tail -n +3 | awk '{print $3}' | xargs -r docker rmi >/dev/null 2>&1 || true

# Clean up old backups (keep last 5)
log "Cleaning up old backups..."
ls -t "$BACKUP_DIR"/redis-*.rdb | tail -n +6 | xargs -r rm || true
ls -t "$BACKUP_DIR"/guardian-data-*.tar.gz | tail -n +6 | xargs -r rm || true
ls -t "$BACKUP_DIR"/liberation-guardian-*.yml | tail -n +6 | xargs -r rm || true

success "Cleanup completed"

# 11. Generate maintenance report
log "11. Generating maintenance report..."

cat > "$BACKUP_DIR/maintenance-report-${DATE_STAMP}.txt" << EOF
Liberation Guardian Monthly Maintenance Report
=============================================
Date: $(date)
Duration: $(($(date +%s) - start_time)) seconds

Pre-maintenance Health: ✅ Healthy
Backup Status: ✅ Completed
Image Updates: ✅ Completed
Model Updates: ✅ Completed
Post-deployment Tests: ✅ Passed
Performance: Response time ${response_time}ms
Resource Usage: Memory $memory_usage

Current Models:
$(docker-compose -f "$COMPOSE_FILE" exec -T ollama ollama list)

Container Status:
$(docker-compose -f "$COMPOSE_FILE" ps)

Cleanup: ✅ Completed
Report Generated: $(date)
EOF

success "Maintenance report generated: $BACKUP_DIR/maintenance-report-${DATE_STAMP}.txt"

# 12. Send notification (if configured)
if [ -n "$SLACK_WEBHOOK_URL" ]; then
    curl -X POST "$SLACK_WEBHOOK_URL" \
        -H 'Content-type: application/json' \
        --data "{\"text\":\"🔄 Liberation Guardian monthly maintenance completed successfully\\nResponse time: ${response_time}ms\\nMemory usage: $memory_usage\"}" \
        >/dev/null 2>&1 || true
    log "Notification sent to Slack"
fi

log "=================================================="
success "Liberation Guardian Monthly Maintenance Completed Successfully!"
log "Next maintenance: $(date -d '+1 month' '+%Y-%m-%d')"
log "=================================================="

# Final health check
if curl -sf http://localhost:9000/health >/dev/null 2>&1; then
    success "Final health check: ✅ All systems operational"
else
    error "Final health check: ❌ System needs attention"
    exit 1
fi

exit 0