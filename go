#!/bin/bash

# Custom go wrapper script
# This intercepts "go run ." and builds Docker instead

if [ "$1" == "run" ] && [ "$2" == "." ]; then
    # Set environment variable and run the application
    DOCKER_BUILD=true go run .
else
    # Pass through to real go command
    /usr/bin/go "$@"
fi
