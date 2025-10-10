# Liberation Guardian AI Model Recommendations

## Recommended Configuration by Use Case

### **Cost-Optimized (Recommended for Testing)**
```yaml
ai_providers:
  triage_agent:
    provider: "anthropic"
    model: "claude-3-5-haiku"     # Fast, cheap triage decisions
    api_key_env: "ANTHROPIC_API_KEY"
    max_tokens: 1000
    temperature: 0.1
    
  analysis_agent:
    provider: "anthropic" 
    model: "claude-3-5-sonnet"    # Deeper analysis when needed
    api_key_env: "ANTHROPIC_API_KEY"
    max_tokens: 4000
    temperature: 0.2
```

### **Performance-Optimized (Production)**
```yaml
ai_providers:
  triage_agent:
    provider: "anthropic"
    model: "claude-3-5-sonnet"    # Best balance for triage
    api_key_env: "ANTHROPIC_API_KEY"
    max_tokens: 2000
    temperature: 0.1
    
  analysis_agent:
    provider: "anthropic"
    model: "claude-3-opus"        # Most capable for complex analysis
    api_key_env: "ANTHROPIC_API_KEY"
    max_tokens: 8000
    temperature: 0.15
```

### **Maximum Intelligence (High-Stakes Production)**
```yaml
ai_providers:
  triage_agent:
    provider: "anthropic"
    model: "claude-3-opus"        # Best model for all decisions
    api_key_env: "ANTHROPIC_API_KEY"
    max_tokens: 4000
    temperature: 0.1
```

## Model Characteristics

### **Claude 3.5 Haiku**
- **Speed**: ⚡⚡⚡ Fastest
- **Cost**: 💰 Cheapest (~$0.25/million input tokens)
- **Intelligence**: ⭐⭐⭐ Good for simple triage
- **Use Case**: High-volume triage decisions

### **Claude 3.5 Sonnet**
- **Speed**: ⚡⚡ Fast
- **Cost**: 💰💰 Moderate (~$3/million input tokens)
- **Intelligence**: ⭐⭐⭐⭐ Excellent for most tasks
- **Use Case**: Balanced triage and analysis

### **Claude 3 Opus**
- **Speed**: ⚡ Slower
- **Cost**: 💰💰💰 Most expensive (~$15/million input tokens)
- **Intelligence**: ⭐⭐⭐⭐⭐ Highest reasoning capability
- **Use Case**: Complex analysis, critical decisions

## Testing Recommendations

### **Start with Haiku for Testing**
1. Set `triage_agent` to `claude-3-5-haiku`
2. Test basic functionality with low cost
3. Upgrade to Sonnet for production

### **Production Deployment**
1. Use `claude-3-5-sonnet` for triage (good balance)
2. Use `claude-3-opus` for analysis agent (complex problems)
3. Monitor costs and adjust based on volume

## Temperature Settings

### **Conservative (Recommended)**
- **Triage**: `temperature: 0.1` (consistent decisions)
- **Analysis**: `temperature: 0.2` (slightly more creative)

### **Creative**
- **Triage**: `temperature: 0.3` (more varied responses)
- **Analysis**: `temperature: 0.4` (creative problem solving)

## Token Limits

### **Efficient**
- **Triage**: `max_tokens: 1000` (concise decisions)
- **Analysis**: `max_tokens: 4000` (detailed analysis)

### **Verbose**
- **Triage**: `max_tokens: 2000` (detailed reasoning)
- **Analysis**: `max_tokens: 8000` (comprehensive analysis)

The current Liberation Guardian configuration uses Claude 3.5 Sonnet for triage and Claude 3 Opus for analysis - a solid production setup!