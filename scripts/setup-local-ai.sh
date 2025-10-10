#!/bin/bash

# Liberation Guardian - Local AI Setup Script
# Downloads and configures local AI models for complete privacy

set -e

echo "🤖 Liberation Guardian - Local AI Setup"
echo "======================================="
echo "Setting up completely private AI operations..."
echo ""

# Configuration
DEFAULT_MODEL="qwen2.5:7b"
MODEL=${1:-$DEFAULT_MODEL}
COMPOSE_FILE="docker-compose.local.yml"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}▶${NC} $1"
}

print_success() {
    echo -e "${GREEN}✅${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠️${NC} $1"
}

print_error() {
    echo -e "${RED}❌${NC} $1"
}

# Check prerequisites
print_status "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

print_success "Docker and Docker Compose are available"

# Check available resources
print_status "Checking system resources..."

# Get available memory in GB
AVAILABLE_RAM=$(free -g | awk '/^Mem:/{print $7}')
if [ "$AVAILABLE_RAM" -lt 8 ]; then
    print_warning "Available RAM: ${AVAILABLE_RAM}GB - Recommended: 8GB+"
    print_warning "Local AI models may run slowly or fail to start"
    echo ""
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    print_success "Available RAM: ${AVAILABLE_RAM}GB - Sufficient for local AI"
fi

# Check available disk space
AVAILABLE_DISK=$(df -BG . | awk 'NR==2 {print $4}' | sed 's/G//')
if [ "$AVAILABLE_DISK" -lt 20 ]; then
    print_warning "Available disk: ${AVAILABLE_DISK}GB - Recommended: 20GB+"
    print_warning "May not have enough space for larger models"
fi

print_success "System check complete"
echo ""

# Model selection
echo "🤖 AI Model Selection"
echo "===================="
echo "Available models for Liberation Guardian:"
echo ""
echo "1. qwen2.5:7b (Recommended) - 4.7GB - Fast, good reasoning"
echo "2. llama3.1:8b - 4.7GB - Strong general performance"  
echo "3. codellama:7b - 3.8GB - Optimized for code analysis"
echo "4. mistral:7b - 4.1GB - Balanced performance"
echo "5. Custom model name"
echo ""

if [ "$#" -eq 0 ]; then
    echo "Which model would you like to use?"
    read -p "Enter choice (1-5) or press Enter for default: " choice
    
    case $choice in
        1|"") MODEL="qwen2.5:7b" ;;
        2) MODEL="llama3.1:8b" ;;
        3) MODEL="codellama:7b" ;;
        4) MODEL="mistral:7b" ;;
        5) 
            read -p "Enter custom model name: " MODEL
            if [ -z "$MODEL" ]; then
                print_error "No model specified"
                exit 1
            fi
            ;;
        *)
            print_error "Invalid choice"
            exit 1
            ;;
    esac
fi

print_success "Selected model: $MODEL"
echo ""

# Start services
print_status "Starting Liberation Guardian with local AI..."

# Set environment variable for the model
export LOCAL_MODEL="$MODEL"

# Stop any existing services
docker-compose -f "$COMPOSE_FILE" down 2>/dev/null || true

# Start Ollama first
print_status "Starting Ollama model server..."
docker-compose -f "$COMPOSE_FILE" up -d ollama redis

# Wait for Ollama to be ready
print_status "Waiting for Ollama to start..."
timeout=120
elapsed=0
while ! docker-compose -f "$COMPOSE_FILE" exec -T ollama ollama list &>/dev/null; do
    if [ $elapsed -ge $timeout ]; then
        print_error "Ollama failed to start within $timeout seconds"
        print_error "Check logs: docker-compose -f $COMPOSE_FILE logs ollama"
        exit 1
    fi
    echo -n "."
    sleep 5
    elapsed=$((elapsed + 5))
done
echo ""
print_success "Ollama is running"

# Download the model
print_status "Downloading AI model: $MODEL"
print_warning "This may take 5-15 minutes depending on your internet connection..."

if docker-compose -f "$COMPOSE_FILE" exec -T ollama ollama pull "$MODEL"; then
    print_success "Model $MODEL downloaded successfully"
else
    print_error "Failed to download model $MODEL"
    print_error "Check your internet connection and model name"
    exit 1
fi

# Start Liberation Guardian
print_status "Starting Liberation Guardian with local AI..."
docker-compose -f "$COMPOSE_FILE" up -d liberation-guardian

# Wait for Liberation Guardian to be ready
print_status "Waiting for Liberation Guardian to start..."
timeout=60
elapsed=0
while ! curl -sf http://localhost:9000/health &>/dev/null; do
    if [ $elapsed -ge $timeout ]; then
        print_error "Liberation Guardian failed to start within $timeout seconds"
        print_error "Check logs: docker-compose -f $COMPOSE_FILE logs liberation-guardian"
        exit 1
    fi
    echo -n "."
    sleep 5
    elapsed=$((elapsed + 5))
done
echo ""
print_success "Liberation Guardian is running with local AI!"

echo ""
echo "🎉 Setup Complete!"
echo "=================="
echo ""
print_success "Liberation Guardian is now running with complete privacy:"
echo ""
echo "📊 Status Dashboard: http://localhost:9000"
echo "🔍 Health Check: curl http://localhost:9000/health"
echo "🤖 AI Model: $MODEL (running locally)"
echo "💰 Monthly Cost: \$0 (no external API calls)"
echo "🔒 Privacy: 100% (all data stays on your machine)"
echo ""
echo "🧪 Test your autonomous AI operations:"
echo 'curl -X POST http://localhost:9000/webhook/custom/test \\'
echo '  -H "Content-Type: application/json" \\'
echo '  -d '"'"'{"event_type":"test","message":"Local AI test","severity":"medium"}'"'"''
echo ""
echo "📋 Useful commands:"
echo "View logs:    docker-compose -f $COMPOSE_FILE logs -f"
echo "Stop:         docker-compose -f $COMPOSE_FILE down"
echo "Restart:      docker-compose -f $COMPOSE_FILE restart liberation-guardian"
echo ""
print_success "Welcome to the future of private, autonomous operations! 🚀"