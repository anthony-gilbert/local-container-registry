# Kubernetes Deployment Guide for Local Container Registry

This guide explains how to deploy the Local Container Registry application to Kubernetes.

## 📋 Prerequisites

1. **Minikube** running with registry support
2. **Docker image** built and pushed to localhost:443
3. **MySQL database** accessible from cluster
4. **GitHub token** for API access

## 🚀 Quick Deployment

### 1. Build and Push Image
```bash
# Build the Docker image
make docker-build

# Tag for local registry
docker tag local-container-registry localhost:443/local-container-registry:testy

# Push to registry
docker push localhost:443/local-container-registry:testy
```

### 2. Configure Secrets
```bash
# Create GitHub token secret (replace with your token)
kubectl create secret generic github-credentials \
  --from-literal=token=ghp_your_github_token_here

# Create MySQL credentials (if needed)
kubectl create secret generic mysql-credentials \
  --from-literal=username=root \
  --from-literal=password=your_mysql_password
```

### 3. Deploy Application
```bash
# Option 1: Simple deployment
kubectl apply -f kubernetes-deployment.yaml

# Option 2: Full pod spec with RBAC
kubectl apply -f kubernetes-pod-spec.yaml
```

### 4. Verify Deployment
```bash
# Check pod status
kubectl get pods -l app=local-container-registry

# Check logs
kubectl logs -l app=local-container-registry -f

# Get service info
kubectl get svc local-container-registry-svc
```

## 📁 File Descriptions

### `kubernetes-pod-spec.yaml`
Complete pod specification including:
- **RBAC configuration** (ServiceAccount, ClusterRole, ClusterRoleBinding)
- **Secrets management** for credentials
- **Volume mounts** for kubeconfig and Docker socket
- **Resource limits** and health checks
- **Security context** configuration
- **ConfigMap** for application settings

### `kubernetes-deployment.yaml`
Simplified deployment for quick testing:
- **Basic deployment** with 1 replica
- **Essential environment** variables
- **Minimal RBAC** requirements
- **Service exposure**

## 🔧 Configuration

### Environment Variables
The application uses these key environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `MYSQL_USER` | MySQL username | `root` |
| `MYSQL_ROOT_PASSWORD` | MySQL password | - |
| `gitHubAuth` | GitHub API token | - |
| `GITHUB_OWNER` | GitHub repository owner | `anthonygilbertt` |
| `GITHUB_REPO` | GitHub repository name | `local-container-registry` |
| `KUBERNETES_NAMESPACE` | Target namespace | `default` |

### Volume Mounts
- **`/root/.kube`** - Kubernetes configuration
- **`/var/run/docker.sock`** - Docker socket for image operations
- **`/var/log/local-container-registry`** - Application logs
- **`/tmp`** - Temporary storage

### Ports
- **8080** - HTTP port (if web interface is added)

## 🔒 Security Considerations

### RBAC Permissions
The application requires these Kubernetes permissions:
- **Pods**: get, list, watch, create, update, patch, delete
- **Deployments**: get, list, watch, create, update, patch
- **Services**: get, list, watch
- **Namespaces**: get, list
- **ConfigMaps/Secrets**: get, list

### Security Context
- **runAsUser**: 0 (required for Docker socket access)
- **allowPrivilegeEscalation**: false
- **capabilities**: minimal set (NET_BIND_SERVICE)

## 🐛 Troubleshooting

### Common Issues

#### 1. Image Pull Errors
```bash
# Check if image exists in registry
curl -k https://localhost:443/v2/_catalog

# Verify Minikube registry configuration
minikube ssh 'cat /etc/docker/daemon.json'
```

#### 2. Permission Errors
```bash
# Check RBAC
kubectl auth can-i get pods --as=system:serviceaccount:default:local-container-registry-sa

# Verify service account
kubectl get sa local-container-registry-sa
```

#### 3. Database Connection Issues
```bash
# Check MySQL connectivity from pod
kubectl exec -it local-container-registry -- nc -zv 127.0.0.1 3306
```

#### 4. Kubeconfig Access
```bash
# Verify kubeconfig mount
kubectl exec -it local-container-registry -- ls -la /root/.kube/
```

### Debug Commands
```bash
# Get detailed pod information
kubectl describe pod local-container-registry

# Check events
kubectl get events --sort-by=.metadata.creationTimestamp

# Access pod shell
kubectl exec -it local-container-registry -- /bin/bash

# Check application logs
kubectl logs local-container-registry -f
```

## 🔄 Updates

### Rolling Update
```bash
# Build new image
docker build -t localhost:443/local-container-registry:v2 .
docker push localhost:443/local-container-registry:v2

# Update deployment
kubectl set image deployment/local-container-registry \
  local-container-registry=localhost:443/local-container-registry:v2
```

### Configuration Updates
```bash
# Update secrets
kubectl patch secret github-credentials \
  -p='{"data":{"token":"new_base64_encoded_token"}}'

# Restart deployment
kubectl rollout restart deployment/local-container-registry
```

## 📊 Monitoring

### Health Checks
The pod includes:
- **Liveness probe**: Checks if process is running
- **Readiness probe**: Validates application startup

### Resource Monitoring
```bash
# Check resource usage
kubectl top pod local-container-registry

# Monitor events
kubectl get events -w
```

## 🔗 Related Commands

```bash
# Port forward for local access (if needed)
kubectl port-forward pod/local-container-registry 8080:8080

# Scale deployment
kubectl scale deployment local-container-registry --replicas=2

# Delete deployment
kubectl delete -f kubernetes-deployment.yaml
```
