# Local Container Registry Usage Guide

This guide explains how to push and manage images in your local Docker registry.

## Prerequisites

Make sure your local registry is running:

```bash
# Start the registry services
docker compose up -d

# Verify services are running
docker ps
```

You should see containers for:
- `local-container-registry-registry-1` (running on port 5000)
- `local-container-registry-nginx-1` (running on port 443)

## Registry Configuration

Your local registry is configured as follows:
- **Registry API**: `http://localhost:5000`
- **Nginx Proxy**: `https://localhost:443` (with SSL)
- **Storage**: Local filesystem in `./data` directory
- **Authentication**: Disabled for development

## Pushing Images to Local Registry

### 1. Tag an Existing Image

Before pushing, you need to tag your image with the registry hostname:

```bash
# Tag an existing image for your local registry
docker tag <source-image> localhost:5000/<image-name>:<tag>

# Example: Tag nginx image
docker tag nginx:latest localhost:5000/my-nginx:latest

# Example: Tag a custom built image
docker tag my-app:v1.0 localhost:5000/my-app:v1.0
```

### 2. Push the Image

```bash
# Push the tagged image to your local registry
docker push localhost:5000/<image-name>:<tag>

# Example:
docker push localhost:5000/my-nginx:latest
docker push localhost:5000/my-app:v1.0
```

### 3. Build and Push in One Step

You can also build and tag an image directly for your registry:

```bash
# Build and tag for local registry
docker build -t localhost:5000/my-app:latest .

# Then push
docker push localhost:5000/my-app:latest
```

## Verifying Images in Registry

### Using curl (Registry API)

```bash
# List all repositories in your registry
curl -s http://localhost:5000/v2/_catalog

# List tags for a specific repository
curl -s http://localhost:5000/v2/<repository-name>/tags/list

# Example:
curl -s http://localhost:5000/v2/my-app/tags/list
```

### Using your TUI Application

1. Run your local container registry application:
   ```bash
   go run .
   ```

2. Navigate to the **Docker** tab
3. Your local registry images should be displayed automatically

## Using Images from Local Registry

### In Docker

```bash
# Pull from your local registry
docker pull localhost:5000/<image-name>:<tag>

# Run a container from your local registry
docker run localhost:5000/<image-name>:<tag>
```

### In Kubernetes

When deploying through your TUI application, images will be automatically:
1. Pulled from your local registry
2. Loaded into Minikube (if using Minikube)
3. Deployed with `ImagePullPolicy: Never`

Manual Kubernetes deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: app
        image: localhost:5000/my-app:latest
        imagePullPolicy: Never  # Important for local images
        ports:
        - containerPort: 80
```

## Common Workflows

### Workflow 1: Push Application Updates

```bash
# 1. Build your application
docker build -t localhost:5000/my-app:v2.0 .

# 2. Push to registry
docker push localhost:5000/my-app:v2.0

# 3. Deploy using your TUI application
go run .
# Navigate to Docker tab → Select image → Press Enter → Deploy
```

### Workflow 2: Push External Images

```bash
# 1. Pull an external image
docker pull redis:7-alpine

# 2. Tag for your local registry
docker tag redis:7-alpine localhost:5000/redis:7-alpine

# 3. Push to local registry
docker push localhost:5000/redis:7-alpine

# 4. Use in your deployments
kubectl create deployment redis --image=localhost:5000/redis:7-alpine
```

## Troubleshooting

### Registry Not Accessible

```bash
# Check if registry is running
curl -s http://localhost:5000/v2/_catalog

# If fails, restart registry
docker compose down && docker compose up -d
```

### Push Fails with "connection refused"

```bash
# Verify port mapping
docker ps | grep registry

# Should show: 0.0.0.0:5000->5000/tcp
```

### Images Not Showing in TUI

1. Verify images are in registry:
   ```bash
   curl -s http://localhost:5000/v2/_catalog
   ```

2. Restart your application:
   ```bash
   go run .
   ```

### Kubernetes Can't Pull Images (Minikube)

```bash
# Load image into Minikube manually
docker pull localhost:5000/my-app:latest
minikube image load localhost:5000/my-app:latest

# Verify image is in Minikube
minikube ssh docker images | grep my-app
```

## Registry Management

### View Registry Contents

```bash
# List all repositories
curl -s http://localhost:5000/v2/_catalog | jq .

# List tags for specific repository
curl -s http://localhost:5000/v2/my-app/tags/list | jq .
```

### Clear Registry Data

```bash
# Stop registry
docker compose down

# Remove all stored images
sudo rm -rf ./data/*

# Restart registry
docker compose up -d
```

### Registry Storage Location

Images are stored in: `./data/docker/registry/v2/repositories/`

## Best Practices

1. **Use consistent tagging**: Include version numbers or git SHA
   ```bash
   docker tag my-app localhost:5000/my-app:$(git rev-parse --short HEAD)
   ```

2. **Clean up old images**: Regularly remove unused images to save space
   ```bash
   docker system prune -a
   ```

3. **Use semantic versioning**: Tag images with meaningful versions
   ```bash
   docker tag my-app localhost:5000/my-app:1.2.3
   docker tag my-app localhost:5000/my-app:latest
   ```

4. **Verify before deploying**: Always check your images are available
   ```bash
   curl -s http://localhost:5000/v2/my-app/tags/list
   ```

## Integration with TUI Application

Your TUI application automatically:
- ✅ **Discovers** images from `localhost:5000`
- ✅ **Displays** them in the Docker tab
- ✅ **Loads** images into Minikube when deploying
- ✅ **Creates** Kubernetes deployments with correct image references
- ✅ **Sets** `ImagePullPolicy: Never` for local images

Simply push images to `localhost:5000/<name>:<tag>` and they'll appear in your application!
