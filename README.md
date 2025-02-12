# Local Container registry

A Golang application that monitors Dockerfile builds, tracks Docker images, and automates deployments to DigitalOcean.

## Features

- **Automated Build Tracking:** Monitors directories for Dockerfile changes and automatically builds Docker images.
- **Image Registry Integration:** Tracks and manages built Docker images.
- **DigitalOcean Deployment:** Automates the deployment process to DigitalOcean Droplets using the DigitalOcean API.
- **Configurable:** Customize Docker registry settings, build options, and deployment parameters.

## Prerequisites

- **Golang:** Version 1.18 or higher.
- **Docker Engine:** Installed and running.
- **DigitalOcean Account:** With an API token that has the necessary permissions.
- **Environment:** Internet access for API calls and Docker registry communication.

## Installation

1. **Clone the repository:**

   ```bash
   git clone https://github.com/yourusername/docker-deployment-tracker.git
   cd docker-deployment-tracker
