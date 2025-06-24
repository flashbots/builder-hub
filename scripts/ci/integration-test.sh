#!/bin/bash
set -eu

SCRIPT_DIR=$(dirname "$0")
cd "$SCRIPT_DIR/../.."

# Build the Docker images
echo "Building Docker images..."
docker compose -f docker/docker-compose.yaml build

# Start the Docker containers
echo "Starting Docker containers..."
docker compose -f docker/docker-compose.yaml up -d

# Waiting for the service to be ready
echo "Waiting for containers to be ready..."
printf 'GET http://localhost:8888/\nHTTP 404' | hurl --retry 60 > /dev/null;

# Waiting 10 more seconds to ensure everything is up
echo "Waiting for additional 10 seconds..."
sleep 10

# Run the tests
echo "Running integration tests..."

# Enable failing for this command
set +e
hurl --test scripts/ci/e2e-test.hurl
set -e

# Cleanup after tests
if [ $? -ne 0 ]; then
    echo "Integration tests failed ❌ - see above for details."
    docker compose -f docker/docker-compose.yaml down -v
    exit 1
fi

echo "Integration tests completed successfully ✅"
docker compose -f docker/docker-compose.yaml down -v