#!/bin/bash

# Build the Docker image
echo "Building Docker image..."
docker build -t local-container-registry .

# Check if build was successful
if [ $? -eq 0 ]; then
    echo "Build successful! Running container..."
    # Run the container
    docker run --rm -it local-container-registry
else
    echo "Build failed!"
    exit 1
fi
