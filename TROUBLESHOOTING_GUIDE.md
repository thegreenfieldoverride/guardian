# 🔧 Liberation Guardian Troubleshooting Guide

**Complete troubleshooting guide for Liberation Guardian - the autonomous AI platform that delivers enterprise features at 1/300th the cost**

## 🎯 **Quick Diagnosis**

Start here for fast problem resolution:

```bash
# Health check - should return 200 OK
curl http://localhost:9000/health

# Check all components
curl http://localhost:9000/ready

# View recent logs
docker-compose logs --tail=50 liberation-guardian

# Check AI provider status
curl http://localhost:9000/api/v1/ai/status
```

---

## 🚨 **Common Issues**

### **🔴 Service Won't Start**

#### **Symptoms:**
- Liberation Guardian fails to start
- Health check returns connection refused
- Docker container exits immediately

#### **Diagnosis:**
```bash
# Check container status
docker-compose ps

# View startup logs
docker-compose logs liberation-guardian

# Check port availability
lsof -i :9000
```

#### **Solutions:**

**Port Already in Use:**
```bash
# Find process using port 9000
lsof -i :9000

# Kill process or change port
export LIBERATION_GUARDIAN_PORT=9001
```

**Missing Environment Variables:**
```bash
# Check required variables
env | grep -E "(GOOGLE_API_KEY|GITHUB_TOKEN)"

# Set missing variables
export GOOGLE_API_KEY="your_key_here"
export GITHUB_TOKEN="ghp_your_token"
```

**Configuration File Issues:**
```bash
# Validate YAML syntax
python -c "import yaml; yaml.safe_load(open('liberation-guardian.yml'))"

# Check file permissions
ls -la liberation-guardian.yml
```

---

### **🔴 AI Provider Connection Failed**

#### **Symptoms:**
- "AI provider error" in logs
- Requests timing out
- No AI analysis results

#### **Diagnosis:**
```bash
# Check AI provider status
curl http://localhost:9000/api/v1/ai/status

# Test Google API directly
curl -H "Authorization: Bearer $GOOGLE_API_KEY" \
  https://generativelanguage.googleapis.com/v1beta/models

# Test Anthropic API
curl -H "Authorization: Bearer $ANTHROPIC_API_KEY" \
  https://api.anthropic.com/v1/messages
```

#### **Solutions:**

**Invalid API Key:**
```bash
# Verify API key format
echo $GOOGLE_API_KEY | wc -c  # Should be ~40 characters

# Test key with minimal request
curl -H "Authorization: Bearer $GOOGLE_API_KEY" \
  "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=$GOOGLE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"contents":[{"parts":[{"text":"test"}]}]}'
```

**Rate Limiting:**
```bash
# Check rate limit headers in logs
docker-compose logs liberation-guardian | grep "rate limit"

# Wait and retry, or switch to backup provider
curl -X PUT http://localhost:9000/api/v1/config \
  -H "Content-Type: application/json" \
  -d '{"primary_provider": "anthropic"}'
```

**Network Issues:**
```bash
# Test connectivity
ping generativelanguage.googleapis.com

# Check proxy settings
echo $HTTP_PROXY
echo $HTTPS_PROXY

# Test from container
docker exec liberation-guardian curl -I https://generativelanguage.googleapis.com
```

---

### **🔴 Redis Connection Failed**

#### **Symptoms:**
- "Redis connection refused" in logs
- Event processing not working
- Memory cache disabled

#### **Diagnosis:**
```bash
# Check Redis status
docker-compose ps redis

# Test Redis connection
redis-cli ping

# Check Redis logs
docker-compose logs redis
```

#### **Solutions:**

**Redis Not Running:**
```bash
# Start Redis
docker-compose up -d redis

# Or with explicit restart
docker-compose restart redis
```

**Wrong Redis Configuration:**
```bash
# Check Redis URL
echo $REDIS_URL

# Test with correct URL
REDIS_URL=redis://localhost:6379 ./liberation-guardian
```

**Redis Authentication:**
```bash
# If Redis requires auth
export REDIS_PASSWORD="your_redis_password"

# Or update configuration
redis:
  host: "redis"
  port: 6379
  password: "your_password"
```

---

### **🔴 GitHub Webhook Issues**

#### **Symptoms:**
- Webhooks not received
- "Webhook verification failed" errors
- Dependabot PRs not processed

#### **Diagnosis:**
```bash
# Check webhook endpoint
curl -X POST http://localhost:9000/webhook/github \
  -H "Content-Type: application/json" \
  -d '{"test": true}'

# Check GitHub webhook settings
# Go to GitHub repo → Settings → Webhooks
# Look for delivery failures
```

#### **Solutions:**

**Webhook Secret Mismatch:**
```bash
# Generate new secret
openssl rand -hex 32

# Update in both GitHub and environment
export GITHUB_WEBHOOK_SECRET="new_secret_here"

# Restart Liberation Guardian
docker-compose restart liberation-guardian
```

**URL Not Accessible:**
```bash
# Test external access
curl -I https://your-domain.com/webhook/github

# Check firewall rules
sudo ufw status

# For local testing, use ngrok
ngrok http 9000
# Then use https://xxxx.ngrok.io/webhook/github in GitHub
```

**Wrong Content Type:**
- Ensure GitHub webhook is set to `application/json`
- Not `application/x-www-form-urlencoded`

---

### **🔴 Dependency Analysis Failing**

#### **Symptoms:**
- PRs not getting analyzed
- "Analysis failed" in logs
- Dependency updates stuck in "pending"

#### **Diagnosis:**
```bash
# Check dependency stats
curl http://localhost:9000/api/v1/dependencies/stats

# Test manual analysis
curl -X POST http://localhost:9000/api/v1/dependencies/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "package_name": "lodash",
    "current_version": "4.17.20",
    "new_version": "4.17.21",
    "ecosystem": "npm"
  }'
```

#### **Solutions:**

**Parsing Issues:**
```bash
# Check PR title format
# Should be: "Bump package from X.X.X to Y.Y.Y"

# Manual debug
docker-compose exec liberation-guardian \
  go run cmd/debug.go --parse-pr "Bump lodash from 4.17.20 to 4.17.21"
```

**Trust Level Too Restrictive:**
```bash
# Check current trust level
curl http://localhost:9000/api/v1/config | jq .trust_level

# Temporarily increase for testing
curl -X PUT http://localhost:9000/api/v1/config/trust-level \
  -H "Content-Type: application/json" \
  -d '{"trust_level": 3}'
```

---

### **🔴 Local AI (Ollama) Issues**

#### **Symptoms:**
- "Ollama connection failed"
- Models not loading
- Slow response times

#### **Diagnosis:**
```bash
# Check Ollama status
curl http://localhost:11434/api/tags

# Check available models
docker exec ollama ollama list

# Check Ollama logs
docker-compose logs ollama
```

#### **Solutions:**

**Models Not Downloaded:**
```bash
# Download required models
docker exec ollama ollama pull qwen2.5:7b
docker exec ollama ollama pull llama3.1:8b

# Or use setup script
./scripts/setup-local-ai.sh
```

**Insufficient Memory:**
```bash
# Check available memory
free -h

# For smaller models
docker exec ollama ollama pull qwen2.5:1.5b

# Increase Docker memory limit
# Docker Desktop → Settings → Resources → Memory → 8GB+
```

**Slow Performance:**
```bash
# Check system resources
htop

# Use GPU acceleration (if available)
docker-compose -f docker-compose.local-gpu.yml up -d

# Or use smaller model
curl -X PUT http://localhost:9000/api/v1/config \
  -d '{"ai_providers": {"triage_agent": {"model": "qwen2.5:1.5b"}}}'
```

---

## 🔍 **Advanced Debugging**

### **Enable Debug Logging**
```bash
# Environment variable
export LOG_LEVEL=debug

# Configuration file
echo "core:
  log_level: debug" >> liberation-guardian.yml

# Runtime change
curl -X PUT http://localhost:9000/api/v1/config/log-level \
  -d '{"level": "debug"}'
```

### **Memory and Performance Issues**
```bash
# Check memory usage
docker stats liberation-guardian

# Check Go memory usage
curl http://localhost:9000/debug/pprof/heap > heap.profile

# Check for memory leaks
go tool pprof heap.profile
```

### **Network Debugging**
```bash
# Enable network tracing
export GODEBUG=http2debug=1

# Check connection pools
curl http://localhost:9000/debug/pprof/goroutine?debug=1

# Test with verbose curl
curl -v http://localhost:9000/health
```

### **Database Issues**
```bash
# Check Redis memory
redis-cli info memory

# Check Redis keyspace
redis-cli info keyspace

# Clear Redis cache (if needed)
redis-cli flushdb
```

---

## 📊 **Monitoring & Alerting**

### **Set Up Alerts**
```yaml
# prometheus-alerts.yml
groups:
- name: liberation-guardian
  rules:
  - alert: LibertationGuardianDown
    expr: up{job="liberation-guardian"} == 0
    for: 1m
    annotations:
      description: "Liberation Guardian is down"

  - alert: HighAICosts
    expr: liberation_guardian_ai_cost_total > 10
    for: 5m
    annotations:
      description: "AI costs exceeded $10"
```

### **Log Analysis**
```bash
# Error rate analysis
docker-compose logs liberation-guardian | grep ERROR | wc -l

# AI provider performance
docker-compose logs liberation-guardian | grep "AI request completed" | \
  awk '{print $NF}' | sort -n

# Top error types
docker-compose logs liberation-guardian | grep ERROR | \
  cut -d'"' -f4 | sort | uniq -c | sort -nr
```

---

## 🆘 **Emergency Procedures**

### **Service Recovery**
```bash
# Quick restart
docker-compose restart liberation-guardian

# Full rebuild
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# Reset to safe mode
export TRUST_LEVEL=0  # Paranoid mode
docker-compose restart liberation-guardian
```

### **Data Recovery**
```bash
# Backup current state
docker exec redis redis-cli save
docker cp redis:/data/dump.rdb ./backup-$(date +%Y%m%d).rdb

# Restore from backup
docker cp backup-20231009.rdb redis:/data/dump.rdb
docker-compose restart redis
```

### **Rollback Configuration**
```bash
# Restore previous config
git checkout HEAD~1 liberation-guardian.yml
docker-compose restart liberation-guardian

# Emergency disable automation
curl -X PUT http://localhost:9000/api/v1/config/trust-level \
  -d '{"trust_level": 0, "reason": "Emergency disable"}'
```

---

## 📞 **Getting Help**

### **Self-Service Resources**
1. **Check logs first**: `docker-compose logs liberation-guardian`
2. **Review configuration**: Validate YAML and environment variables
3. **Test components**: AI providers, Redis, GitHub connectivity
4. **Check GitHub issues**: [Known issues and solutions](https://github.com/liberation-guardian/issues)

### **Community Support**
- 📧 [Email Support](mailto:support@greenfieldoverride.com) - Direct help from the team
- 📖 [Documentation](README.md) - Complete guides in this repo
- 🐛 [GitHub Issues](https://github.com/greenfieldoverride/liberation-guardian/issues) - Bug reports
- 💡 [GitHub Discussions](https://github.com/greenfieldoverride/liberation-guardian/discussions) - Q&A and feature requests

---

## 📝 **Diagnostic Report Template**

When reporting issues, include this information:

```bash
# System Information
echo "=== System Info ==="
uname -a
docker --version
docker-compose --version

echo "=== Liberation Guardian Status ==="
curl -s http://localhost:9000/health | jq .

echo "=== Configuration ==="
cat liberation-guardian.yml

echo "=== Environment Variables ==="
env | grep -E "(GOOGLE|GITHUB|ANTHROPIC|REDIS)" | sed 's/=.*/=***/'

echo "=== Recent Logs ==="
docker-compose logs --tail=20 liberation-guardian

echo "=== Resource Usage ==="
docker stats --no-stream
```

### **Issue Template**
```markdown
## Issue Description
Brief description of the problem

## Environment
- Deployment type: (Docker/Kubernetes/Local)
- Liberation Guardian version: 
- Operating System:

## Steps to Reproduce
1. 
2. 
3. 

## Expected Behavior
What should happen

## Actual Behavior  
What actually happens

## Diagnostic Output
```
[Paste diagnostic report here]
```

## Additional Context
Any other relevant information
```

---

**Most issues can be resolved quickly with the right diagnosis. When in doubt, check the logs first!** 🔍