#!/bin/bash
# Local E2E Setup Script
# Sets up APIGate + Hoster for local production-like E2E testing
#
# Usage:
#   ./scripts/local-e2e-setup.sh
#
# Prerequisites:
#   - Docker and docker-compose installed
#   - Go 1.21+ installed (for building from source)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== Hoster Local E2E Setup ==="
echo "Project root: $PROJECT_ROOT"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_step() {
    echo -e "${GREEN}[STEP]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
print_step "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    print_error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Use 'docker compose' if available, otherwise 'docker-compose'
if docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
else
    COMPOSE_CMD="docker-compose"
fi

echo "Using: $COMPOSE_CMD"

# Create data directories
print_step "Creating data directories..."
mkdir -p "$PROJECT_ROOT/data"

# Check if APIGate image exists or needs to be pulled
print_step "Checking APIGate image..."
if ! docker image inspect ghcr.io/artpar/apigate:latest &> /dev/null; then
    print_warning "APIGate image not found locally. It will be pulled on first run."
fi

# Build Hoster image
print_step "Building Hoster image..."
cd "$PROJECT_ROOT"

# Check if Dockerfile exists
if [ ! -f "Dockerfile" ]; then
    print_warning "Dockerfile not found. Creating a basic one..."
    cat > Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o hoster ./cmd/hoster

FROM alpine:latest
RUN apk --no-cache add ca-certificates wget
WORKDIR /app
COPY --from=builder /app/hoster .

EXPOSE 8080 9091
CMD ["./hoster"]
EOF
fi

docker build -t hoster:local -f Dockerfile .

# Start services
print_step "Starting services..."
cd "$PROJECT_ROOT/deploy"
$COMPOSE_CMD -f docker-compose.local.yml up -d

# Wait for services to be healthy
print_step "Waiting for services to be healthy..."
echo "Waiting for APIGate..."
for i in {1..30}; do
    if curl -sf http://localhost:8082/health > /dev/null 2>&1; then
        echo "APIGate is ready!"
        break
    fi
    sleep 1
    echo -n "."
done

echo ""
echo "Waiting for Hoster..."
for i in {1..30}; do
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        echo "Hoster is ready!"
        break
    fi
    sleep 1
    echo -n "."
done

echo ""
echo ""
echo "=== Setup Complete ==="
echo ""
echo "Services running:"
echo "  - APIGate Portal: http://localhost:8082/portal"
echo "  - Hoster API:     http://localhost:8080/api/v1"
echo "  - App Proxy:      http://localhost:9091"
echo ""
echo "Next steps:"
echo "  1. Open http://localhost:8082/portal"
echo "  2. Sign up for a new account"
echo "  3. Create an API key"
echo "  4. Use the API key to create templates and deployments"
echo ""
echo "To stop: make local-e2e-down"
echo "To view logs: make local-e2e-logs"
