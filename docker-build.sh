#!/bin/bash

# Liberation Guardian Docker Build Script
# Builds optimized multi-architecture images ready for production

set -e

# Configuration
IMAGE_NAME="liberation/guardian"
VERSION=${VERSION:-"latest"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "🐳 Building Liberation Guardian Docker Images"
echo "=============================================="
echo "Version: $VERSION"
echo "Build Time: $BUILD_TIME"
echo "Git Commit: $GIT_COMMIT"
echo ""

# Check if buildx is available
if ! docker buildx version >/dev/null 2>&1; then
    echo "❌ Docker Buildx not available. Please install Docker Desktop or enable buildx."
    exit 1
fi

# Create builder if it doesn't exist
if ! docker buildx inspect liberation-builder >/dev/null 2>&1; then
    echo "📦 Creating multi-platform builder..."
    docker buildx create --name liberation-builder --driver docker-container --bootstrap
fi

# Use the builder
docker buildx use liberation-builder

echo "🏗️  Building multi-architecture images..."

# Build and optionally push
if [ "$1" = "--push" ]; then
    echo "📤 Building and pushing to registry..."
    docker buildx build \
        --platform linux/amd64,linux/arm64,linux/arm/v7 \
        --file Dockerfile.optimized \
        --build-arg VERSION="$VERSION" \
        --build-arg BUILD_TIME="$BUILD_TIME" \
        --build-arg GIT_COMMIT="$GIT_COMMIT" \
        --tag "$IMAGE_NAME:$VERSION" \
        --tag "$IMAGE_NAME:latest" \
        --push \
        .
else
    echo "🔨 Building for local testing..."
    docker buildx build \
        --platform linux/amd64 \
        --file Dockerfile.optimized \
        --build-arg VERSION="$VERSION" \
        --build-arg BUILD_TIME="$BUILD_TIME" \
        --build-arg GIT_COMMIT="$GIT_COMMIT" \
        --tag "$IMAGE_NAME:$VERSION" \
        --tag "$IMAGE_NAME:latest" \
        --load \
        .
fi

echo ""
echo "✅ Build complete!"
echo ""
echo "🚀 Quick test:"
echo "docker run --rm -p 9000:9000 -e GOOGLE_API_KEY=your_key $IMAGE_NAME:$VERSION"
echo ""
echo "📋 Image info:"
docker images "$IMAGE_NAME:$VERSION"
echo ""

if [ "$1" = "--push" ]; then
    echo "🌍 Published to Docker Hub:"
    echo "docker pull $IMAGE_NAME:$VERSION"
else
    echo "💡 To push to registry:"
    echo "./docker-build.sh --push"
fi

echo ""
echo "🎯 Liberation Guardian is ready to revolutionize operations!"