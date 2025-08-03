#!/bin/bash

# Helper script to load Docker images into Minikube for Kubernetes deployment

if [ $# -eq 0 ]; then
    echo "Usage: $0 <image-name:tag>"
    echo "Example: $0 my-app:latest"
    echo ""
    echo "This script will:"
    echo "1. Check if the image exists locally"
    echo "2. Load the image into Minikube's Docker daemon"
    echo "3. Verify the image is available in Minikube"
    exit 1
fi

IMAGE_NAME="$1"

echo "🔍 Checking if image '$IMAGE_NAME' exists locally..."
if ! docker images --format "table {{.Repository}}:{{.Tag}}" | grep -q "^$IMAGE_NAME$"; then
    echo "❌ Image '$IMAGE_NAME' not found locally."
    echo "Available images:"
    docker images --format "table {{.Repository}}:{{.Tag}}" | head -10
    exit 1
fi

echo "✅ Image '$IMAGE_NAME' found locally."

echo "🚀 Loading image into Minikube..."
if minikube image load "$IMAGE_NAME"; then
    echo "✅ Successfully loaded '$IMAGE_NAME' into Minikube."
else
    echo "❌ Failed to load image into Minikube."
    echo "Make sure Minikube is running: minikube status"
    exit 1
fi

echo "🔍 Verifying image is available in Minikube..."
if minikube image ls | grep -q "$IMAGE_NAME"; then
    echo "✅ Image '$IMAGE_NAME' is now available in Minikube."
    echo ""
    echo "🎉 You can now deploy this image to Kubernetes!"
else
    echo "⚠️  Image may not be properly loaded. Check with: minikube image ls"
fi