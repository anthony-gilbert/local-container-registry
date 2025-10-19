FROM golang:1.23.2

WORKDIR /app

# Copy pre-built binary (built locally to avoid Docker build issues)
COPY local-container-registry .

# Install kubectl for Kubernetes operations
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/

# For now, run as root to access kubeconfig and Docker socket
# TODO: Implement proper security with user permissions
USER root

CMD ["./local-container-registry"]
