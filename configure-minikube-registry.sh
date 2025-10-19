#!/bin/bash

# Configure Minikube to work with local registry at localhost:443

echo "Configuring Minikube for local registry at localhost:443..."

# Enable registry addon in Minikube
minikube addons enable registry

# Add insecure registry to Minikube
minikube start --insecure-registry="localhost:443"

# Alternative: Configure Docker daemon in Minikube to trust the registry
minikube ssh 'echo "{\"insecure-registries\": [\"localhost:443\", \"host.minikube.internal:443\"]}" | sudo tee /etc/docker/daemon.json'
minikube ssh 'sudo systemctl restart docker'

# Get Minikube IP for reference
MINIKUBE_IP=$(minikube ip)
echo "Minikube IP: $MINIKUBE_IP"

# Add host entry for registry access from inside Minikube
minikube ssh "echo '$MINIKUBE_IP host.minikube.internal' | sudo tee -a /etc/hosts"

echo "Configuration complete!"
echo ""
echo "To push images to the registry from Minikube:"
echo "1. Tag your image: docker tag your-image localhost:443/your-image"
echo "2. Push to registry: docker push localhost:443/your-image" 
echo "3. Use in pod spec: localhost:443/your-image"
echo ""
echo "Note: You may need to restart Minikube for changes to take effect:"
echo "minikube stop && minikube start --insecure-registry=\"localhost:443\""
