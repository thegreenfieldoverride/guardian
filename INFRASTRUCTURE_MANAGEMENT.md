# Liberation Guardian: Infrastructure Management

## 🔄 **Production Infrastructure Patterns**

Local AI deployments require thoughtful infrastructure management for reliability and performance.

## 🟢🔵 **Blue/Green Deployment Strategy**

### **Why Blue/Green for Local AI?**
- **Model updates** without downtime
- **Resource isolation** - models are resource-heavy 
- **Quick rollback** if new model performs poorly
- **Testing safety** - validate model performance before switching

### **Blue/Green Implementation**

```yaml
# docker-compose.blue-green.yml
version: '3.8'

services:
  # Blue Environment (Current Production)
  liberation-guardian-blue:
    image: liberation/guardian:stable
    environment:
      - AI_MODE=local
      - LOCAL_AI_BASE_URL=http://ollama-blue:11434
      - ENVIRONMENT=production-blue
    profiles: ["blue"]
    
  ollama-blue:
    image: ollama/ollama:latest
    volumes:
      - ollama_models_blue:/root/.ollama
    profiles: ["blue"]
    
  # Green Environment (New Version/Model)
  liberation-guardian-green:
    image: liberation/guardian:latest
    environment:
      - AI_MODE=local
      - LOCAL_AI_BASE_URL=http://ollama-green:11434
      - ENVIRONMENT=production-green
    profiles: ["green"]
    
  ollama-green:
    image: ollama/ollama:latest
    volumes:
      - ollama_models_green:/root/.ollama
    profiles: ["green"]
    
  # Load Balancer / Traffic Manager
  nginx:
    image: nginx:alpine
    ports:
      - "9000:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - liberation-guardian-blue
```

### **Deployment Commands**

```bash
# Deploy to green environment
docker-compose --profile green up -d

# Test green environment
curl http://localhost:9001/health

# Switch traffic to green (update nginx config)
./scripts/switch-to-green.sh

# Verify green is working
./scripts/health-check.sh

# Tear down blue environment
docker-compose --profile blue down
```

## 📅 **Monthly Rebuild Schedule**

### **Why Monthly Rebuilds?**
- **Model updates** - Newer, better local models released regularly
- **Security patches** - Base image and OS updates
- **Memory cleanup** - Fresh containers prevent memory fragmentation
- **Configuration drift** - Reset to known good state
- **Performance optimization** - Clear caches, optimize resource allocation

### **Automated Monthly Maintenance**

```bash
#!/bin/bash
# scripts/monthly-maintenance.sh

echo "🔄 Liberation Guardian Monthly Maintenance"
echo "=========================================="

# 1. Pre-maintenance health check
./scripts/health-check.sh

# 2. Create maintenance backup
docker-compose exec redis redis-cli BGSAVE
docker cp liberation-redis:/data/dump.rdb ./backups/redis-$(date +%Y%m%d).rdb

# 3. Pull latest images
docker-compose pull

# 4. Deploy to green environment with latest
docker-compose --profile green up -d

# 5. Download latest recommended models
docker-compose --profile green exec ollama ollama pull qwen2.5:7b
docker-compose --profile green exec ollama ollama pull codellama:7b

# 6. Run integration tests on green
./scripts/test-ai-functionality.sh green

# 7. Switch traffic to green
./scripts/switch-to-green.sh

# 8. Monitor for 1 hour
./scripts/monitor-deployment.sh 3600

# 9. Clean up old blue environment
docker-compose --profile blue down
docker volume rm liberation_ollama_models_blue

# 10. Update blue to be new standby
docker-compose --profile blue up -d

echo "✅ Monthly maintenance complete!"
```

### **Automated Scheduling**

```bash
# Add to crontab for monthly execution
# crontab -e

# Run monthly maintenance on first Sunday of each month at 2 AM
0 2 * * 0 [ $(date +\%d) -le 7 ] && /path/to/liberation-guardian/scripts/monthly-maintenance.sh >> /var/log/liberation-maintenance.log 2>&1
```

## 🔍 **Health Monitoring & Alerts**

### **Continuous Health Checks**

```yaml
# monitoring/docker-compose.monitoring.yml
services:
  # Health check service
  health-monitor:
    image: liberation/guardian:monitor
    environment:
      - BLUE_URL=http://liberation-guardian-blue:9000
      - GREEN_URL=http://liberation-guardian-green:9000
      - ALERT_WEBHOOK=${SLACK_WEBHOOK_URL}
    volumes:
      - ./monitoring/health-checks.yml:/app/health-checks.yml
    restart: unless-stopped
```

### **Performance Monitoring**

```bash
# scripts/performance-monitor.sh

# Check model response times
curl -w "@curl-format.txt" -o /dev/null -s "http://localhost:9000/webhook/test"

# Check memory usage
docker stats --no-stream | grep ollama

# Check model accuracy (run test scenarios)
./scripts/test-ai-accuracy.sh

# Alert if performance degrades
if [ "$RESPONSE_TIME" -gt 5000 ]; then
    ./scripts/alert-slow-response.sh
fi
```

## 🎯 **Resource Management**

### **Model Lifecycle**

```yaml
# Model update strategy
model_management:
  # Keep 2 versions of each model
  retention_policy: "keep_last_2"
  
  # Update schedule
  update_schedule:
    - model: "qwen2.5:7b"
      check_frequency: "weekly"
      auto_update: false  # Manual approval required
      
    - model: "codellama:7b" 
      check_frequency: "monthly"
      auto_update: true   # Safe to auto-update
      
  # Resource limits per model
  resource_limits:
    memory: "8GB"
    storage: "20GB"
    cpu_cores: 4
```

### **Cleanup Automation**

```bash
# scripts/cleanup-old-models.sh

echo "🧹 Cleaning up old models and containers..."

# Remove unused models (older than 60 days)
docker images | grep ollama | awk '$6 ~ /[6-9][0-9]|[1-9][0-9][0-9]/ {print $3}' | xargs docker rmi

# Clean up old volumes
docker volume ls -q | grep ollama | grep -v blue | grep -v green | xargs docker volume rm

# Clean up old container logs
docker system prune -f

# Clean up model cache
docker-compose exec ollama sh -c "find /root/.ollama -name '*.tmp' -delete"

echo "✅ Cleanup complete"
```

## 🚨 **Rollback Procedures**

### **Emergency Rollback**

```bash
# scripts/emergency-rollback.sh

echo "🚨 EMERGENCY ROLLBACK - Switching back to blue environment"

# Immediate traffic switch
./scripts/switch-to-blue.sh

# Restart blue if needed
docker-compose --profile blue restart

# Verify blue is healthy
timeout 60 ./scripts/health-check.sh blue

# Alert team
./scripts/alert-rollback.sh "Emergency rollback executed"

echo "✅ Rollback complete - investigate green environment"
```

### **Rollback Triggers**

```bash
# Automatic rollback conditions
if [ "$ERROR_RATE" -gt 5 ]; then
    ./scripts/emergency-rollback.sh "High error rate: $ERROR_RATE%"
fi

if [ "$RESPONSE_TIME" -gt 10000 ]; then
    ./scripts/emergency-rollback.sh "Response time too slow: ${RESPONSE_TIME}ms"
fi

if ! ./scripts/health-check.sh green; then
    ./scripts/emergency-rollback.sh "Health check failed"
fi
```

## 📋 **Infrastructure Checklist**

### **Monthly Tasks**
- [ ] Update base images
- [ ] Update AI models
- [ ] Performance testing
- [ ] Security scanning
- [ ] Backup verification
- [ ] Resource optimization
- [ ] Documentation updates

### **Weekly Tasks**
- [ ] Health check validation
- [ ] Performance metrics review
- [ ] Log analysis
- [ ] Resource usage monitoring

### **Daily Tasks**
- [ ] Automated health checks
- [ ] Error rate monitoring
- [ ] Response time tracking
- [ ] Resource alerts

## 🎯 **Production Recommendations**

### **For Small Teams (< 10 developers)**
- **Simple monthly rebuild** - Take 5-minute maintenance window
- **Basic health monitoring** - Daily automated checks
- **Manual model updates** - Review and approve new models

### **For Larger Teams (10+ developers)**
- **Blue/Green deployments** - Zero downtime updates
- **Automated rollbacks** - Immediate recovery from issues
- **Comprehensive monitoring** - Real-time performance tracking
- **Staged model updates** - Test → Staging → Production

### **For Enterprise**
- **Multi-environment** - Dev/Test/Staging/Prod isolation
- **Advanced monitoring** - APM, alerting, dashboards
- **Automated everything** - CI/CD, testing, deployment
- **Compliance tracking** - Audit trails, change management

**Liberation Guardian infrastructure should be as autonomous as the AI it runs!** 🚀