# Liberation Guardian: Deployment Options

## 🎯 **Choose Your Liberation Level**

Liberation Guardian offers multiple deployment strategies. Pick what works for YOUR situation.

## ☁️ **Cloud AI (Recommended for Most)**

**Best for:** Small teams, getting started, minimal resource usage

```yaml
# docker-compose.yml
services:
  liberation-guardian:
    image: liberation/guardian:latest
    environment:
      - GOOGLE_API_KEY=${GOOGLE_API_KEY}
    deploy:
      resources:
        limits:
          memory: 256M
          cpus: '0.5'
```

**Trade-offs:**
- ✅ **Tiny footprint** - 150MB image, 256MB RAM
- ✅ **Fast startup** - Ready in 30 seconds
- ✅ **Latest models** - Always using cutting-edge AI
- ✅ **Zero maintenance** - No model updates needed
- 💰 **Cost**: $0-5/month (free with Gemini tier)
- 📡 **Privacy**: Data sent to AI providers (Google/Anthropic)

## 🏠 **Local AI (Maximum Privacy)**

**Best for:** Compliance requirements, air-gapped environments, ultimate privacy

```yaml
# docker-compose.local.yml
services:
  liberation-guardian:
    image: liberation/guardian:local
    environment:
      - AI_PROVIDER=local
      - LOCAL_MODEL_URL=http://ollama:11434
    depends_on:
      - ollama
      
  ollama:
    image: ollama/ollama:latest
    volumes:
      - models:/root/.ollama
    command: ollama serve qwen2.5:7b
    deploy:
      resources:
        limits:
          memory: 12G
          cpus: '4.0'
```

**Trade-offs:**
- 🔒 **Complete privacy** - Zero external API calls
- 💰 **Zero ongoing cost** - No API fees ever
- 🛡️ **Air-gapped capable** - Works with no internet
- 🎛️ **Full control** - Own your models and data
- 📦 **Large footprint** - 6-12GB RAM, 4+ CPU cores
- ⏱️ **Slower startup** - 5-10 minutes for model loading
- 🔧 **More maintenance** - Model updates, resource tuning

## ⚖️ **Hybrid AI (Best of Both)**

**Best for:** Teams that want flexibility and redundancy

```yaml
# docker-compose.hybrid.yml
services:
  liberation-guardian:
    image: liberation/guardian:hybrid
    environment:
      # Primary: Local model
      - AI_PROVIDER=local
      - LOCAL_MODEL_URL=http://ollama:11434
      
      # Fallback: Cloud models
      - FALLBACK_PROVIDER=google
      - GOOGLE_API_KEY=${GOOGLE_API_KEY}
      
      # When to fallback
      - FALLBACK_ON_HIGH_LOAD=true
      - FALLBACK_ON_MODEL_UNAVAILABLE=true
```

**Trade-offs:**
- 🎯 **Local-first** - Uses private model when possible
- 🔄 **Cloud fallback** - Switches to cloud if local overloaded
- 💰 **Variable cost** - $0-5/month depending on fallback usage
- 📦 **Medium footprint** - Local model + small cloud usage

## 📊 **Comparison Matrix**

| Feature | Cloud AI | Local AI | Hybrid AI |
|---------|----------|----------|-----------|
| **Monthly Cost** | $0-5 | $0 | $0-3 |
| **RAM Usage** | 256MB | 6-12GB | 6-12GB |
| **CPU Usage** | Low | High | High |
| **Startup Time** | 30s | 5-10min | 5-10min |
| **Privacy** | Medium | Maximum | High |
| **Compliance** | Depends | ✅ Full | ✅ Full |
| **Maintenance** | None | Medium | Medium |
| **Internet Required** | Yes | No | Optional |

## 🎯 **Recommendation Guide**

### **Choose Cloud AI If:**
- 🏃 **Getting started** - Want to try Liberation Guardian quickly
- 💻 **Resource constrained** - Limited RAM/CPU available
- 🌐 **Internet always available** - Reliable external connectivity
- 👥 **Small team** - Under 10 developers
- 💰 **Cost conscious** - Want predictable low monthly cost

### **Choose Local AI If:**
- 🔒 **Compliance requirements** - HIPAA, SOC2, government
- 🛡️ **Maximum privacy** - Data cannot leave your infrastructure
- 🏢 **Enterprise security** - Air-gapped or restricted networks
- 💰 **High volume** - Processing 1000s of events daily
- 🎛️ **Complete control** - Want to own entire AI stack

### **Choose Hybrid AI If:**
- ⚖️ **Best of both** - Want local privacy with cloud reliability
- 📈 **Variable load** - Sometimes high volume, sometimes low
- 🔄 **Redundancy** - Want backup if local model fails
- 🧪 **Experimenting** - Testing local models before full commitment

## 🚀 **Quick Start Commands**

### **Cloud AI (5 seconds)**
```bash
curl -sSL https://get.liberation.dev | bash -s -- --cloud
```

### **Local AI (5 minutes)**
```bash
curl -sSL https://get.liberation.dev | bash -s -- --local --model qwen2.5:7b
```

### **Hybrid AI (5 minutes)**
```bash
curl -sSL https://get.liberation.dev | bash -s -- --hybrid
```

## 💡 **Resource Planning**

### **Cloud AI Requirements**
- **RAM**: 512MB minimum, 1GB recommended
- **CPU**: 1 core minimum, 2 cores recommended
- **Storage**: 1GB for logs and data
- **Network**: Reliable internet connection

### **Local AI Requirements**
- **RAM**: 8GB minimum, 16GB+ recommended
- **CPU**: 4 cores minimum, 8+ cores recommended  
- **Storage**: 20GB for models, 5GB for data
- **Network**: Optional (can run air-gapped)

### **Hybrid AI Requirements**
- Same as Local AI for hardware
- Reliable internet for fallback scenarios

## 🎛️ **Liberation Principle**

**You choose what works for YOUR situation.** 

- Need to start fast and cheap? → Cloud AI
- Need maximum privacy? → Local AI  
- Want flexibility? → Hybrid AI
- Want to change later? → Easy migration between all options

**No vendor lock-in. No forced upgrades. No hidden costs. Just working autonomous operations that respect your constraints.** 🚀