#!/bin/bash

# Agent System Comprehensive Test Script
# Tests: agent-contact API, set/revoke agent status, downline reassignment

API_BASE="http://localhost:8080/api/v1"
ADMIN_TOKEN=""
TEST_COUNT=0
PASS_COUNT=0
FAIL_COUNT=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

log_test() {
    TEST_COUNT=$((TEST_COUNT + 1))
    echo "[$TEST_COUNT] $1"
}

log_pass() {
    PASS_COUNT=$((PASS_COUNT + 1))
    echo -e "${GREEN}  ✓ PASS${NC}"
}

log_fail() {
    FAIL_COUNT=$((FAIL_COUNT + 1))
    echo -e "${RED}  ✗ FAIL: $1${NC}"
}

# Get admin token
get_admin_token() {
    cd /Users/a/Desktop/Usb2api/sub2api/backend
    ADMIN_TOKEN=$(go run ./cmd/jwtgen/main.go 2>/dev/null | grep "JWT=" | cut -d'=' -f2)
    if [ -z "$ADMIN_TOKEN" ]; then
        echo "Failed to get admin token"
        exit 1
    fi
    echo "Admin token obtained"
}

# Test agent-contact API for a user
test_agent_contact() {
    local user_id=$1
    local email=$2
    local expected_agent=$3

    log_test "Testing agent-contact for user $user_id ($email), expecting agent: $expected_agent"

    # Get token for user
    cd /Users/a/Desktop/Usb2api/sub2api/backend
    local token=$(go run ./cmd/jwtgen/main.go -email "$email" 2>/dev/null | grep "JWT=" | cut -d'=' -f2)

    if [ -z "$token" ]; then
        log_fail "Failed to get token for $email"
        return
    fi

    local response=$(curl -s "$API_BASE/user/agent-contact" -H "Authorization: Bearer $token")
    local has_agent=$(echo "$response" | grep -o '"has_agent":[^,}]*' | cut -d':' -f2)
    local agent_email=$(echo "$response" | grep -o '"email":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [ "$expected_agent" = "none" ]; then
        if [ "$has_agent" = "false" ]; then
            log_pass
        else
            log_fail "Expected no agent, got has_agent=$has_agent"
        fi
    else
        if [ "$has_agent" = "true" ] && [ "$agent_email" = "$expected_agent" ]; then
            log_pass
        else
            log_fail "Expected agent $expected_agent, got has_agent=$has_agent, email=$agent_email"
        fi
    fi
}

# Test set agent status
test_set_agent() {
    local user_id=$1
    local is_agent=$2
    local parent_id=$3

    log_test "Setting agent status for user $user_id: is_agent=$is_agent, parent=$parent_id"

    local body="{\"is_agent\": $is_agent"
    if [ -n "$parent_id" ] && [ "$parent_id" != "null" ]; then
        body="$body, \"parent_agent_id\": $parent_id"
    fi
    body="$body}"

    local response=$(curl -s -X PATCH "$API_BASE/admin/agents/$user_id/status" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$body")

    local code=$(echo "$response" | grep -o '"code":[0-9]*' | cut -d':' -f2)

    if [ "$code" = "0" ]; then
        log_pass
    else
        log_fail "API returned code $code"
    fi
}

# Main test execution
echo "=========================================="
echo "Agent System Comprehensive Test"
echo "=========================================="

get_admin_token

echo ""
echo "=== Phase 1: Agent Contact API Tests (18 users) ==="

# Test all 18 users
test_agent_contact 1 "admin@example.com" "none"
test_agent_contact 10 "user1@test.com" "admin@example.com"
test_agent_contact 11 "user2@test.com" "admin@example.com"
test_agent_contact 12 "agent1@test.com" "none"
test_agent_contact 13 "user3@test.com" "admin@example.com"
test_agent_contact 14 "user4@test.com" "admin@example.com"
test_agent_contact 15 "123123@qq.com" "admin@example.com"
test_agent_contact 16 "5238410499@qq.com" "admin@example.com"
test_agent_contact 17 "test01@example.com" "admin@example.com"
test_agent_contact 18 "test02@example.com" "admin@example.com"
test_agent_contact 19 "test03@example.com" "admin@example.com"
test_agent_contact 20 "test04@example.com" "user1@test.com"
test_agent_contact 21 "test05@example.com" "user1@test.com"
test_agent_contact 22 "test06@example.com" "user1@test.com"
test_agent_contact 23 "test07@example.com" "agent1@test.com"
test_agent_contact 24 "test08@example.com" "agent1@test.com"
test_agent_contact 25 "test09@example.com" "agent1@test.com"
test_agent_contact 26 "test10@example.com" "agent1@test.com"

echo ""
echo "=== Phase 2: Set Agent Status Tests ==="

# Test setting various users as agents
test_set_agent 17 true 1
test_set_agent 18 true 1
test_set_agent 19 true 10

echo ""
echo "=== Phase 3: Verify New Agent Hierarchy ==="

test_agent_contact 17 "test01@example.com" "admin@example.com"
test_agent_contact 18 "test02@example.com" "admin@example.com"
test_agent_contact 19 "test03@example.com" "user1@test.com"

echo ""
echo "=== Phase 4: Revoke Agent Status Tests ==="

test_set_agent 17 false null
test_set_agent 18 false null
test_set_agent 19 false null

echo ""
echo "=== Phase 5: Multiple Set/Revoke Cycles ==="

for i in {1..5}; do
    test_set_agent 17 true 1
    test_set_agent 17 false null
done

echo ""
echo "=== Phase 6: Agent Contact After Revoke ==="

test_agent_contact 17 "test01@example.com" "admin@example.com"
test_agent_contact 18 "test02@example.com" "admin@example.com"
test_agent_contact 19 "test03@example.com" "admin@example.com"

echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo "Total Tests: $TEST_COUNT"
echo -e "Passed: ${GREEN}$PASS_COUNT${NC}"
echo -e "Failed: ${RED}$FAIL_COUNT${NC}"
echo "=========================================="
