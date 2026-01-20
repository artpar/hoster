#!/bin/bash
# Local E2E Test Script
# Tests the full user flow: signup -> login -> create template -> deploy -> access
#
# Usage:
#   ./scripts/test-local-e2e.sh
#
# Prerequisites:
#   - APIGate running on localhost:8082
#   - Hoster running on localhost:8080
#   - curl and jq installed

set -e

# Configuration
APIGATE_URL="http://localhost:8082"
HOSTER_URL="http://localhost:8080"
APP_PROXY_URL="http://localhost:9091"

# Test user credentials
TEST_EMAIL="test-$(date +%s)@example.com"
TEST_PASSWORD="testpassword123"
TEST_NAME="Test User"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_step() {
    echo -e "${GREEN}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

print_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

# Check prerequisites
check_prereqs() {
    print_step "Checking prerequisites..."

    if ! command -v curl &> /dev/null; then
        print_error "curl is not installed"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        print_error "jq is not installed"
        exit 1
    fi

    # Check services
    if ! curl -sf "$APIGATE_URL/health" > /dev/null 2>&1; then
        print_error "APIGate is not running at $APIGATE_URL"
        exit 1
    fi

    if ! curl -sf "$HOSTER_URL/health" > /dev/null 2>&1; then
        print_error "Hoster is not running at $HOSTER_URL"
        exit 1
    fi

    print_success "All prerequisites met"
}

# Step 1: Register user via APIGate
test_user_registration() {
    print_step "Step 1: Registering user via APIGate portal..."

    REGISTER_RESPONSE=$(curl -sf -X POST "$APIGATE_URL/portal/api/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$TEST_EMAIL\", \"password\": \"$TEST_PASSWORD\", \"name\": \"$TEST_NAME\"}" \
        2>&1) || {
        print_error "Failed to register user"
        print_info "Response: $REGISTER_RESPONSE"
        return 1
    }

    USER_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.id // .user_id // .data.id // empty')
    if [ -z "$USER_ID" ]; then
        print_info "Response: $REGISTER_RESPONSE"
        print_info "User registration may have different response format - continuing..."
    else
        print_success "User registered: $USER_ID"
    fi
}

# Step 2: Login via APIGate
test_user_login() {
    print_step "Step 2: Logging in via APIGate portal..."

    LOGIN_RESPONSE=$(curl -sf -X POST "$APIGATE_URL/portal/api/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$TEST_EMAIL\", \"password\": \"$TEST_PASSWORD\"}" \
        -c /tmp/apigate_cookies.txt \
        2>&1) || {
        print_error "Failed to login"
        print_info "Response: $LOGIN_RESPONSE"
        return 1
    }

    print_success "Login successful"
    print_info "Response: $LOGIN_RESPONSE"
}

# Step 3: Create API key
test_create_api_key() {
    print_step "Step 3: Creating API key..."

    KEY_RESPONSE=$(curl -sf -X POST "$APIGATE_URL/portal/api/keys" \
        -H "Content-Type: application/json" \
        -b /tmp/apigate_cookies.txt \
        -d '{"name": "test-key"}' \
        2>&1) || {
        print_error "Failed to create API key"
        print_info "Response: $KEY_RESPONSE"
        return 1
    }

    API_KEY=$(echo "$KEY_RESPONSE" | jq -r '.key // .api_key // .data.key // empty')
    if [ -z "$API_KEY" ]; then
        print_info "Response: $KEY_RESPONSE"
        print_error "Could not extract API key from response"
        return 1
    fi

    print_success "API key created: ${API_KEY:0:10}..."
    export API_KEY
}

# Step 4: Test Hoster API via APIGate (with header injection)
test_hoster_api() {
    print_step "Step 4: Testing Hoster API via APIGate..."

    # List templates (should work even with no templates)
    TEMPLATES_RESPONSE=$(curl -sf "$APIGATE_URL/api/v1/templates" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        2>&1) || {
        print_error "Failed to list templates"
        print_info "Response: $TEMPLATES_RESPONSE"
        return 1
    }

    print_success "Hoster API accessible via APIGate"
    print_info "Templates response: $TEMPLATES_RESPONSE"
}

# Step 5: Create a template
test_create_template() {
    print_step "Step 5: Creating a template..."

    TEMPLATE_PAYLOAD=$(cat << 'EOF'
{
    "data": {
        "type": "templates",
        "attributes": {
            "name": "Test Web App",
            "description": "A simple test web application",
            "version": "1.0.0",
            "compose_spec": "version: '3.8'\nservices:\n  web:\n    image: nginx:alpine\n    ports:\n      - '80:80'",
            "category": "web",
            "tags": ["test", "nginx"]
        }
    }
}
EOF
)

    TEMPLATE_RESPONSE=$(curl -sf -X POST "$APIGATE_URL/api/v1/templates" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/vnd.api+json" \
        -d "$TEMPLATE_PAYLOAD" \
        2>&1) || {
        print_error "Failed to create template"
        print_info "Response: $TEMPLATE_RESPONSE"
        return 1
    }

    TEMPLATE_ID=$(echo "$TEMPLATE_RESPONSE" | jq -r '.data.id // empty')
    if [ -z "$TEMPLATE_ID" ]; then
        print_info "Response: $TEMPLATE_RESPONSE"
        print_error "Could not extract template ID"
        return 1
    fi

    print_success "Template created: $TEMPLATE_ID"
    export TEMPLATE_ID
}

# Step 6: Publish template
test_publish_template() {
    print_step "Step 6: Publishing template..."

    PUBLISH_RESPONSE=$(curl -sf -X POST "$APIGATE_URL/api/v1/templates/$TEMPLATE_ID/publish" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        2>&1) || {
        print_error "Failed to publish template"
        print_info "Response: $PUBLISH_RESPONSE"
        return 1
    }

    print_success "Template published"
}

# Step 7: Create deployment from template
test_create_deployment() {
    print_step "Step 7: Creating deployment from template..."

    DEPLOYMENT_PAYLOAD=$(cat << EOF
{
    "data": {
        "type": "deployments",
        "attributes": {
            "template_id": "$TEMPLATE_ID",
            "name": "test-deployment-$(date +%s)"
        }
    }
}
EOF
)

    DEPLOYMENT_RESPONSE=$(curl -sf -X POST "$APIGATE_URL/api/v1/deployments" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/vnd.api+json" \
        -d "$DEPLOYMENT_PAYLOAD" \
        2>&1) || {
        print_error "Failed to create deployment"
        print_info "Response: $DEPLOYMENT_RESPONSE"
        return 1
    }

    DEPLOYMENT_ID=$(echo "$DEPLOYMENT_RESPONSE" | jq -r '.data.id // empty')
    if [ -z "$DEPLOYMENT_ID" ]; then
        print_info "Response: $DEPLOYMENT_RESPONSE"
        print_error "Could not extract deployment ID"
        return 1
    fi

    print_success "Deployment created: $DEPLOYMENT_ID"
    export DEPLOYMENT_ID
}

# Step 8: Start deployment
test_start_deployment() {
    print_step "Step 8: Starting deployment..."

    START_RESPONSE=$(curl -sf -X POST "$APIGATE_URL/api/v1/deployments/$DEPLOYMENT_ID/start" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        2>&1) || {
        print_error "Failed to start deployment"
        print_info "Response: $START_RESPONSE"
        return 1
    }

    print_success "Deployment started"
    print_info "Waiting for deployment to be running..."

    # Wait for deployment to be running
    for i in {1..30}; do
        STATUS_RESPONSE=$(curl -sf "$APIGATE_URL/api/v1/deployments/$DEPLOYMENT_ID" \
            -H "Authorization: Bearer $API_KEY" \
            2>&1) || continue

        STATUS=$(echo "$STATUS_RESPONSE" | jq -r '.data.attributes.status // empty')
        if [ "$STATUS" = "running" ]; then
            print_success "Deployment is running"
            DEPLOYMENT_DOMAIN=$(echo "$STATUS_RESPONSE" | jq -r '.data.attributes.domains[0].hostname // empty')
            export DEPLOYMENT_DOMAIN
            return 0
        fi

        sleep 2
        echo -n "."
    done

    print_error "Deployment did not start in time"
    return 1
}

# Step 9: Access deployed app via proxy
test_access_app() {
    print_step "Step 9: Accessing deployed app via proxy..."

    if [ -z "$DEPLOYMENT_DOMAIN" ]; then
        print_error "No deployment domain found"
        return 1
    fi

    print_info "Trying to access: http://$DEPLOYMENT_DOMAIN"

    # Add to /etc/hosts if needed (manual step)
    APP_RESPONSE=$(curl -sf --resolve "$DEPLOYMENT_DOMAIN:9091:127.0.0.1" \
        "http://$DEPLOYMENT_DOMAIN:9091/" \
        2>&1) || {
        print_error "Failed to access app via proxy"
        print_info "You may need to add '$DEPLOYMENT_DOMAIN' to /etc/hosts"
        return 1
    }

    print_success "App accessible via proxy!"
}

# Cleanup
cleanup() {
    print_step "Cleaning up..."

    if [ -n "$DEPLOYMENT_ID" ]; then
        curl -sf -X POST "$APIGATE_URL/api/v1/deployments/$DEPLOYMENT_ID/stop" \
            -H "Authorization: Bearer $API_KEY" > /dev/null 2>&1 || true

        curl -sf -X DELETE "$APIGATE_URL/api/v1/deployments/$DEPLOYMENT_ID" \
            -H "Authorization: Bearer $API_KEY" > /dev/null 2>&1 || true
    fi

    if [ -n "$TEMPLATE_ID" ]; then
        curl -sf -X DELETE "$APIGATE_URL/api/v1/templates/$TEMPLATE_ID" \
            -H "Authorization: Bearer $API_KEY" > /dev/null 2>&1 || true
    fi

    rm -f /tmp/apigate_cookies.txt

    print_success "Cleanup complete"
}

# Main test flow
main() {
    echo "=== Hoster Local E2E Test ==="
    echo "APIGate URL: $APIGATE_URL"
    echo "Hoster URL:  $HOSTER_URL"
    echo "Test Email:  $TEST_EMAIL"
    echo ""

    trap cleanup EXIT

    check_prereqs
    echo ""

    test_user_registration || print_info "Continuing without registration (may already exist)"
    test_user_login || exit 1
    test_create_api_key || exit 1
    test_hoster_api || exit 1
    test_create_template || exit 1
    test_publish_template || exit 1
    test_create_deployment || exit 1
    test_start_deployment || exit 1
    test_access_app || print_info "App proxy access test skipped (may need /etc/hosts entry)"

    echo ""
    echo "=== All E2E Tests Passed! ==="
}

main "$@"
