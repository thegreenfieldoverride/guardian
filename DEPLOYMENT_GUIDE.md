# 🚀 Liberation Guardian Deployment Guide

**Complete deployment guide for Liberation Guardian - proving enterprise AI automation doesn't require enterprise budgets**

## 📋 **Table of Contents**

- [Quick Start Options](#quick-start-options)
- [Local Development](#local-development)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Cloud Deployment](#cloud-deployment)
- [Local AI Setup](#local-ai-setup)
- [Configuration Guide](#configuration-guide)
- [Security Setup](#security-setup)
- [Monitoring & Observability](#monitoring--observability)

---

## ⚡ **Quick Start Options**

Choose your deployment path based on your needs:

| Option | Time | Cost | Use Case |
|--------|------|------|----------|
| [**Docker Compose**](#docker-deployment) | 5 min | $5-20/month | Development, small teams |
| [**Kubernetes**](#kubernetes-deployment) | 15 min | $20-100/month | Production, scalability |
| [**Cloud Services**](#cloud-deployment) | 10 min | $10-50/month | Managed infrastructure |
| [**Local AI**](#local-ai-setup) | 20 min | $0/month | Complete privacy |

---

## 💻 **Local Development**

Perfect for testing and development.

### **Prerequisites**
- Go 1.21+ ([Download](https://golang.org/dl/))
- Redis ([Download](https://redis.io/download) or `brew install redis`)
- Git

### **Step 1: Clone Repository**
```bash
git clone https://github.com/liberation-guardian/liberation-guardian
cd liberation-guardian
```

### **Step 2: Install Dependencies**
```bash
go mod download
```

### **Step 3: Configure Environment**
```bash
cp .env.example .env
```

Edit `.env` with your settings:
```bash
# Required: AI Provider
GOOGLE_API_KEY=your_gemini_api_key_here

# Required: GitHub Integration  
GITHUB_TOKEN=ghp_your_github_token
GITHUB_WEBHOOK_SECRET=your_webhook_secret

# Optional: Additional Providers
ANTHROPIC_API_KEY=your_claude_api_key
SENTRY_WEBHOOK_SECRET=your_sentry_secret
SLACK_WEBHOOK_URL=your_slack_webhook
```

### **Step 4: Start Redis**
```bash
# macOS with Homebrew
brew services start redis

# Linux
sudo systemctl start redis

# Docker
docker run -d -p 6379:6379 redis:alpine
```

### **Step 5: Run Liberation Guardian**
```bash
go run cmd/main.go
```

### **Step 6: Verify Installation**
```bash
curl http://localhost:9000/health
```

**Expected Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "components": {
    "redis": "healthy",
    "ai_providers": "healthy"
  }
}
```

---

## 🐳 **Docker Deployment**

Recommended for most production deployments.

### **Option 1: Standard Deployment**
For cloud AI providers (Gemini, Claude, GPT).

```bash
# Clone repository
git clone https://github.com/liberation-guardian/liberation-guardian
cd liberation-guardian

# Configure environment
cp .env.example .env
# Edit .env with your API keys

# Start with Docker Compose
docker-compose up -d

# Check status
docker-compose logs -f liberation-guardian
curl http://localhost:9000/health
```

### **Option 2: Local AI Deployment**
For 100% private processing with Ollama.

```bash
# Start with local AI stack
docker-compose -f docker-compose.local.yml up -d

# Wait for Ollama to download models (first time only)
docker-compose -f docker-compose.local.yml logs -f ollama

# Check status
curl http://localhost:9000/health
```

### **Option 3: Production Deployment**
For production environments with monitoring.

```bash
# Use production compose file
docker-compose -f docker-compose.production.yml up -d

# Check all services
docker-compose -f docker-compose.production.yml ps
```

### **Docker Environment Variables**
```bash
# .env file for Docker
GOOGLE_API_KEY=your_gemini_api_key
GITHUB_TOKEN=ghp_your_github_token
GITHUB_WEBHOOK_SECRET=random_secret_string
REDIS_URL=redis://redis:6379
LIBERATION_GUARDIAN_PORT=9000
TRUST_LEVEL=2
```

### **Docker Compose Services**
The deployment includes:
- **liberation-guardian**: Main application
- **redis**: Event queue and caching
- **ollama** (optional): Local AI processing
- **prometheus** (optional): Metrics collection
- **grafana** (optional): Monitoring dashboard

---

## ☸️ **Kubernetes Deployment**

For production environments requiring high availability and scaling.

### **Step 1: Install with Helm**
```bash
# Add Helm repository
helm repo add liberation-guardian https://charts.liberation-guardian.dev
helm repo update

# Create namespace
kubectl create namespace liberation-guardian

# Install with custom values
helm install liberation-guardian liberation-guardian/liberation-guardian \
  --namespace liberation-guardian \
  --set config.ai.googleApiKey="your_gemini_api_key" \
  --set config.github.token="ghp_your_github_token" \
  --set config.github.webhookSecret="your_webhook_secret" \
  --set config.trustLevel=2
```

### **Step 2: Configure Ingress**
```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: liberation-guardian
  namespace: liberation-guardian
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - guardian.yourdomain.com
    secretName: liberation-guardian-tls
  rules:
  - host: guardian.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: liberation-guardian
            port:
              number: 9000
```

### **Step 3: Apply Configuration**
```bash
kubectl apply -f ingress.yaml

# Check deployment
kubectl get pods -n liberation-guardian
kubectl get ingress -n liberation-guardian
```

### **Kubernetes Scaling**
```bash
# Scale horizontally
kubectl scale deployment liberation-guardian --replicas=3 -n liberation-guardian

# Enable autoscaling
kubectl autoscale deployment liberation-guardian \
  --cpu-percent=70 \
  --min=2 \
  --max=10 \
  -n liberation-guardian
```

---

## ☁️ **Cloud Deployment**

### **AWS Deployment**

#### **Option 1: ECS Fargate**
```bash
# Build and push image
docker build -t your-account.dkr.ecr.region.amazonaws.com/liberation-guardian .
docker push your-account.dkr.ecr.region.amazonaws.com/liberation-guardian

# Deploy with Terraform
cd terraform/aws
terraform init
terraform plan -var="image_url=your-account.dkr.ecr.region.amazonaws.com/liberation-guardian"
terraform apply
```

#### **Option 2: EKS**
```bash
# Create EKS cluster
eksctl create cluster --name liberation-guardian --region us-west-2

# Deploy with Helm
helm install liberation-guardian liberation-guardian/liberation-guardian \
  --set config.cloud.provider=aws \
  --set config.cloud.region=us-west-2
```

### **Google Cloud Deployment**

#### **Option 1: Cloud Run**
```bash
# Build and deploy
gcloud builds submit --tag gcr.io/your-project/liberation-guardian
gcloud run deploy liberation-guardian \
  --image gcr.io/your-project/liberation-guardian \
  --platform managed \
  --region us-central1 \
  --set-env-vars GOOGLE_API_KEY=your_key,GITHUB_TOKEN=your_token
```

#### **Option 2: GKE**
```bash
# Create GKE cluster
gcloud container clusters create liberation-guardian \
  --zone us-central1-a \
  --num-nodes 3

# Deploy with Helm
helm install liberation-guardian liberation-guardian/liberation-guardian \
  --set config.cloud.provider=gcp \
  --set config.cloud.region=us-central1
```

### **Azure Deployment**

#### **Container Instances**
```bash
# Create resource group
az group create --name liberation-guardian --location eastus

# Deploy container
az container create \
  --resource-group liberation-guardian \
  --name liberation-guardian \
  --image liberationguardian/liberation-guardian:latest \
  --environment-variables \
    GOOGLE_API_KEY=your_key \
    GITHUB_TOKEN=your_token \
  --ports 9000
```

---

## 🤖 **Local AI Setup**

For complete privacy and zero external dependencies.

### **Step 1: Start Local AI Stack**
```bash
# Start Ollama and Liberation Guardian
docker-compose -f docker-compose.local.yml up -d

# Wait for Ollama to start
sleep 30
```

### **Step 2: Download AI Models**
```bash
# Download recommended models
./scripts/setup-local-ai.sh

# Or manually:
docker exec -it ollama ollama pull qwen2.5:7b
docker exec -it ollama ollama pull llama3.1:8b
docker exec -it ollama ollama pull mistral:7b
```

### **Step 3: Test Local AI**
```bash
# Test Ollama connection
curl http://localhost:11434/api/tags

# Test Liberation Guardian with local AI
curl -X POST http://localhost:9000/api/v1/ai/test \
  -H "Content-Type: application/json" \
  -d '{"provider": "ollama", "prompt": "Test local AI"}'
```

### **Local AI Configuration**
```yaml
# liberation-guardian.yml
ai_providers:
  triage_agent:
    provider: "ollama"
    model: "qwen2.5:7b"
    api_key_env: ""  # Not needed for local
    local_config:
      base_url: "http://ollama:11434"
      health_check_interval: "30s"
      startup_timeout: "5m"
```

### **Model Recommendations**

| Model | Size | Speed | Quality | Use Case |
|-------|------|-------|---------|----------|
| **qwen2.5:7b** | 4.4GB | Fast | High | Recommended default |
| **llama3.1:8b** | 4.7GB | Fast | High | General purpose |
| **mistral:7b** | 4.1GB | Very Fast | Good | Quick responses |
| **codellama:7b** | 3.8GB | Fast | High | Code analysis |

---

## ⚙️ **Configuration Guide**

### **Core Configuration**
```yaml
# liberation-guardian.yml
core:
  name: "Liberation Guardian Production"
  environment: "production"
  log_level: "info"
  port: 9000

# Redis for event processing
redis:
  host: "redis"
  port: 6379
  password: ""
  db: 0
```

### **AI Provider Configuration**
```yaml
ai_providers:
  # Primary: FREE Gemini
  triage_agent:
    provider: "google"
    model: "gemini-2.0-flash"
    api_key_env: "GOOGLE_API_KEY"
    max_tokens: 2000
    temperature: 0.1

  # Fallback: Claude Haiku  
  backup_agent:
    provider: "anthropic"
    model: "claude-3-5-haiku"
    api_key_env: "ANTHROPIC_API_KEY"
    max_tokens: 1000
    temperature: 0.1
```

### **Trust Level Configuration**
```yaml
dependencies:
  trust_level: 2  # BALANCED
  security_auto_approve: true
  patch_auto_approve: true
  minor_auto_approve: false
  major_auto_approve: false
  min_confidence: 0.80
  min_test_coverage: 0.70
```

### **Integration Configuration**
```yaml
integrations:
  source_control:
    github:
      enabled: true
      token_env: "GITHUB_TOKEN"
      webhook_secret_env: "GITHUB_WEBHOOK_SECRET"
      auto_merge_enabled: true

  observability:
    sentry:
      enabled: true
      webhook_secret_env: "SENTRY_WEBHOOK_SECRET"
      auto_acknowledge: false

    prometheus:
      enabled: true
      scrape_url: "http://prometheus:9090"
```

---

## 🔒 **Security Setup**

### **API Key Management**
Never hardcode API keys. Use environment variables or secret management:

```bash
# Environment variables
export GOOGLE_API_KEY="your_key_here"
export GITHUB_TOKEN="ghp_your_token"

# Kubernetes secrets
kubectl create secret generic liberation-guardian-secrets \
  --from-literal=google-api-key="your_key" \
  --from-literal=github-token="ghp_your_token"

# Docker secrets
echo "your_api_key" | docker secret create google_api_key -
```

### **Webhook Security**
Secure webhook endpoints with proper verification:

```yaml
# liberation-guardian.yml
integrations:
  source_control:
    github:
      webhook_secret_env: "GITHUB_WEBHOOK_SECRET"  # Required!
  observability:
    sentry:
      webhook_secret_env: "SENTRY_WEBHOOK_SECRET"  # Required!
```

Generate strong secrets:
```bash
# Generate random secrets
openssl rand -hex 32  # For GITHUB_WEBHOOK_SECRET
openssl rand -hex 32  # For SENTRY_WEBHOOK_SECRET
```

### **Network Security**
```yaml
# docker-compose.yml
services:
  liberation-guardian:
    networks:
      - liberation-guardian-network
    # Only expose port 9000 externally
    ports:
      - "9000:9000"

  redis:
    networks:
      - liberation-guardian-network
    # Redis only accessible internally
    # No external ports exposed

networks:
  liberation-guardian-network:
    driver: bridge
```

### **TLS/SSL Configuration**
```nginx
# nginx.conf
server {
    listen 443 ssl http2;
    server_name guardian.yourdomain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 📊 **Monitoring & Observability**

### **Prometheus Metrics**
Liberation Guardian exposes metrics at `/metrics`:

```bash
# Check metrics endpoint
curl http://localhost:9000/metrics

# Key metrics:
# - liberation_guardian_events_processed_total
# - liberation_guardian_ai_requests_total  
# - liberation_guardian_automations_executed_total
# - liberation_guardian_cost_total
```

### **Grafana Dashboard**
Import the provided dashboard:

```bash
# Start monitoring stack
docker-compose -f docker-compose.production.yml up -d

# Access Grafana at http://localhost:3000
# Username: admin, Password: admin

# Import dashboard from:
# https://grafana.com/dashboards/liberation-guardian
```

### **Log Aggregation**
```yaml
# docker-compose.yml
services:
  liberation-guardian:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
    # Or use centralized logging:
    # logging:
    #   driver: fluentd
    #   options:
    #     fluentd-address: localhost:24224
```

### **Health Checks**
```yaml
# docker-compose.yml
services:
  liberation-guardian:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

---

## 🔧 **Post-Deployment Steps**

### **1. Configure GitHub Webhooks**
Add webhook to your repositories:

- **URL**: `https://your-domain.com/webhook/github`
- **Content type**: `application/json`
- **Secret**: Your `GITHUB_WEBHOOK_SECRET`
- **Events**: `Pull requests`, `Pushes`, `Issues`

### **2. Test Dependency Automation**
```bash
# Run test suite
./test-dependabot-automation.sh --demo

# Create test PR to verify automation
```

### **3. Monitor Initial Operation**
```bash
# Watch logs
docker-compose logs -f liberation-guardian

# Check metrics
curl http://localhost:9000/api/v1/status

# Verify AI providers
curl http://localhost:9000/api/v1/ai/status
```

### **4. Adjust Trust Level**
Start conservative and increase gradually:

```bash
# Start with CONSERVATIVE
curl -X PUT http://localhost:9000/api/v1/config/trust-level \
  -H "Content-Type: application/json" \
  -d '{"trust_level": 1}'

# After 1 week, increase to BALANCED
curl -X PUT http://localhost:9000/api/v1/config/trust-level \
  -H "Content-Type: application/json" \
  -d '{"trust_level": 2}'
```

---

## 🆘 **Troubleshooting**

### **Common Issues**

#### **AI Provider Connection Failed**
```bash
# Check API key
echo $GOOGLE_API_KEY

# Test API directly
curl -H "Authorization: Bearer $GOOGLE_API_KEY" \
  https://generativelanguage.googleapis.com/v1beta/models
```

#### **Redis Connection Failed**
```bash
# Check Redis status
docker-compose ps redis

# Test Redis connection
redis-cli ping
```

#### **Webhook Verification Failed**
```bash
# Check webhook secret
echo $GITHUB_WEBHOOK_SECRET

# Verify webhook configuration in GitHub
```

### **Debug Mode**
```bash
# Enable debug logging
export LOG_LEVEL=debug

# Or in configuration
echo "core:
  log_level: debug" >> liberation-guardian.yml
```

### **Get Support**
- 📖 [Troubleshooting Guide](TROUBLESHOOTING_GUIDE.md)
- 💬 [Discord Community](https://discord.gg/liberation-guardian)
- 🐛 [GitHub Issues](https://github.com/liberation-guardian/issues)

---

## 🎯 **Next Steps**

1. **Configure additional repositories** for dependency automation
2. **Set up monitoring dashboards** for operational visibility
3. **Customize trust levels** based on your risk tolerance
4. **Integrate with CI/CD pipelines** for full automation
5. **Scale horizontally** as your usage grows

**Congratulations! Liberation Guardian is now protecting your infrastructure with AI-powered automation.** 🚀

---

**Need help with deployment?** Email us at [support@greenfieldoverride.com](mailto:support@greenfieldoverride.com)