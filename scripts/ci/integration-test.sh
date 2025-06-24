#!/bin/bash
set -eu

SCRIPT_DIR=$(dirname "$0")
cd "$SCRIPT_DIR/../.."

# Build the Docker images
docker compose -f docker/docker-compose.yaml build

# Start the Docker containers
docker compose -f docker/docker-compose.yaml up -d

# Waiting for the service to be ready
echo "Waiting for containers to be ready..."
printf 'GET http://localhost:8888/\nHTTP 404' | hurl --retry 60 > /dev/null;

# Run the tests
echo "Running integration tests..."
hurl --test scripts/ci/setup.hurl
echo "Integration tests completed successfully."

# Stop and remove the Docker containers
docker compose -f docker/docker-compose.yaml down -v