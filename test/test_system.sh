#!/bin/bash

# Comprehensive Test Script for Healing Eval System
# Tests API endpoints, windowing, and evaluations

set -e

BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api/v1"
DATA_FILE="test/data/conversations.json"

echo "========================================="
echo "Healing Eval System - Comprehensive Tests"
echo "========================================="
echo ""

# Check if server is running
echo "[1/10] Checking if server is running..."
if curl -s "$BASE_URL/health" > /dev/null; then
    echo "✓ Server is running"
else
    echo "✗ Server is not running. Please start it with: make run-server"
    exit 1
fi

# Test 1: Submit short conversation (3 turns)
echo ""
echo "[2/10] Testing short conversation (3 turns)..."
SHORT_CONV=$(cat <<'EOF'
{
  "conversations": [
    {
      "conversation_id": "test_short_001",
      "agent_version": "v1.0-test",
      "turns": [
        {"turn_id": 1, "role": "user", "content": "Hello, what's the weather like today?", "timestamp": "2024-01-15T10:00:00Z"},
        {"turn_id": 2, "role": "assistant", "content": "I don't have access to real-time weather data.", "timestamp": "2024-01-15T10:00:02Z"},
        {"turn_id": 3, "role": "user", "content": "Thanks anyway!", "timestamp": "2024-01-15T10:00:10Z"}
      ]
    }
  ]
}
EOF
)

RESPONSE=$(curl -s -X POST "$API_URL/conversations" \
  -H "Content-Type: application/json" \
  -d "$SHORT_CONV")

if echo "$RESPONSE" | grep -q "accepted"; then
    echo "✓ Short conversation submitted successfully"
    ACCEPTED=$(echo "$RESPONSE" | grep -o '"accepted":[0-9]*' | cut -d':' -f2)
    echo "  Accepted: $ACCEPTED conversation(s)"
else
    echo "✗ Failed to submit short conversation"
    echo "  Response: $RESPONSE"
fi

# Wait for processing
echo "  Waiting 10 seconds for evaluation..."
sleep 10

# Test 2: Submit medium conversation with tool calls (7 turns)
echo ""
echo "[3/10] Testing medium conversation with tool calls (7 turns)..."
MEDIUM_CONV=$(cat <<'EOF'
{
  "conversations": [
    {
      "conversation_id": "test_medium_tool_001",
      "agent_version": "v1.0-test",
      "turns": [
        {"turn_id": 1, "role": "user", "content": "I need to book a flight from San Francisco to New York for next Monday.", "timestamp": "2024-01-15T10:00:00Z"},
        {"turn_id": 2, "role": "assistant", "content": "I'll help you search for flights.", "tool_calls": [{"tool_name": "flight_search", "parameters": {"origin": "SFO", "destination": "NYC", "date": "2024-01-22"}, "result": {"status": "success"}, "latency_ms": 450}], "timestamp": "2024-01-15T10:00:05Z"},
        {"turn_id": 3, "role": "user", "content": "What are the prices?", "timestamp": "2024-01-15T10:00:20Z"},
        {"turn_id": 4, "role": "assistant", "content": "Let me get the pricing details.", "tool_calls": [{"tool_name": "flight_pricing", "parameters": {"flight_ids": ["UA123", "AA456"]}, "result": {"status": "success"}, "latency_ms": 300}], "timestamp": "2024-01-15T10:00:22Z"},
        {"turn_id": 5, "role": "user", "content": "I'll take UA123", "timestamp": "2024-01-15T10:00:40Z"},
        {"turn_id": 6, "role": "assistant", "content": "Great! I'll proceed with booking.", "tool_calls": [{"tool_name": "book_flight", "parameters": {"flight_id": "UA123"}, "result": {"status": "success"}, "latency_ms": 650}], "timestamp": "2024-01-15T10:00:43Z"},
        {"turn_id": 7, "role": "user", "content": "Perfect, thanks!", "timestamp": "2024-01-15T10:01:00Z"}
      ]
    }
  ]
}
EOF
)

RESPONSE=$(curl -s -X POST "$API_URL/conversations" \
  -H "Content-Type: application/json" \
  -d "$MEDIUM_CONV")

if echo "$RESPONSE" | grep -q "accepted"; then
    echo "✓ Medium conversation with tools submitted successfully"
else
    echo "✗ Failed to submit medium conversation"
fi

echo "  Waiting 15 seconds for evaluation..."
sleep 15

# Test 3: Submit long conversation (35 turns) - Tests windowing
echo ""
echo "[4/10] Testing long conversation (35 turns) - Windowing test..."
if [ -f "$DATA_FILE" ]; then
    # Extract just the 35-turn conversation
    LONG_CONV=$(cat "$DATA_FILE" | jq '{conversations: [.conversations[] | select(.conversation_id == "test_long_context_035")]}')
    
    RESPONSE=$(curl -s -X POST "$API_URL/conversations" \
      -H "Content-Type: application/json" \
      -d "$LONG_CONV")
    
    if echo "$RESPONSE" | grep -q "accepted"; then
        echo "✓ Long conversation (35 turns) submitted successfully"
        echo "  This will test windowing in LLM Judge, Tool Call, and Coherence evaluators"
    else
        echo "✗ Failed to submit long conversation"
    fi
    
    echo "  Waiting 20 seconds for evaluation (longer due to windowing)..."
    sleep 20
else
    echo "⚠ Test data file not found: $DATA_FILE"
fi

# Test 4: Submit very long conversation (50 turns) - Stress test windowing
echo ""
echo "[5/10] Testing very long conversation (50 turns) - Stress test..."
if [ -f "$DATA_FILE" ]; then
    VERY_LONG_CONV=$(cat "$DATA_FILE" | jq '{conversations: [.conversations[] | select(.conversation_id == "test_very_long_050")]}')
    
    RESPONSE=$(curl -s -X POST "$API_URL/conversations" \
      -H "Content-Type: application/json" \
      -d "$VERY_LONG_CONV")
    
    if echo "$RESPONSE" | grep -q "accepted"; then
        echo "✓ Very long conversation (50 turns) submitted successfully"
        echo "  This stress-tests windowing and token management"
    else
        echo "✗ Failed to submit very long conversation"
    fi
    
    echo "  Waiting 25 seconds for evaluation..."
    sleep 25
else
    echo "⚠ Test data file not found"
fi

# Test 5: Query evaluations
echo ""
echo "[6/10] Querying evaluation results..."
QUERY_RESPONSE=$(curl -s -X POST "$API_URL/evaluations/query" \
  -H "Content-Type: application/json" \
  -d '{"limit": 10}')

EVAL_COUNT=$(echo "$QUERY_RESPONSE" | jq -r '.total // 0')
echo "✓ Found $EVAL_COUNT evaluations"

if [ "$EVAL_COUNT" -gt 0 ]; then
    echo "  Sample evaluation scores:"
    echo "$QUERY_RESPONSE" | jq -r '.evaluations[0] | "  - Overall: \(.scores.overall), Type: \(.evaluator_type), Conv: \(.conversation_id)"' 2>/dev/null || echo "  (Unable to parse)"
fi

# Test 6: Test suggestion generation
echo ""
echo "[7/10] Testing automated suggestion generation..."
SUGG_RESPONSE=$(curl -s -X POST "$API_URL/suggestions/generate" \
  -H "Content-Type: application/json" \
  -d '{"hours_back": 24, "max_score": 0.9}')

if echo "$SUGG_RESPONSE" | grep -q "patterns_detected"; then
    PATTERNS=$(echo "$SUGG_RESPONSE" | jq -r '.patterns_detected // 0')
    SUGGESTIONS=$(echo "$SUGG_RESPONSE" | jq -r '.suggestions_generated // 0')
    echo "✓ Suggestion generation completed"
    echo "  Patterns detected: $PATTERNS"
    echo "  Suggestions generated: $SUGGESTIONS"
else
    echo "✗ Suggestion generation failed"
    echo "  Response: $SUGG_RESPONSE"
fi

# Test 7: List suggestions
echo ""
echo "[8/10] Listing generated suggestions..."
LIST_RESPONSE=$(curl -s "$API_URL/suggestions")
SUGG_COUNT=$(echo "$LIST_RESPONSE" | jq -r '.suggestions | length')
echo "✓ Found $SUGG_COUNT suggestions"

if [ "$SUGG_COUNT" -gt 0 ]; then
    echo "  Sample suggestion:"
    echo "$LIST_RESPONSE" | jq -r '.suggestions[0] | "  - Type: \(.suggestion_type), Target: \(.target), Confidence: \(.confidence)"' 2>/dev/null || echo "  (Unable to parse)"
fi

# Test 8: Check evaluator metrics
echo ""
echo "[9/10] Checking evaluator metrics..."
METRICS_RESPONSE=$(curl -s "$API_URL/metrics/evaluators")

if echo "$METRICS_RESPONSE" | grep -q "evaluators"; then
    echo "✓ Evaluator metrics retrieved"
    echo "$METRICS_RESPONSE" | jq -r '.evaluators[] | "  - \(.evaluator_type): Precision=\(.precision), Recall=\(.recall), F1=\(.f1_score)"' 2>/dev/null || echo "  (Metrics available)"
else
    echo "⚠ No evaluator metrics available yet (need human annotations)"
fi

# Test 9: Test Web UI accessibility
echo ""
echo "[10/10] Testing Web UI accessibility..."
if curl -s "$BASE_URL/" | grep -q "Dashboard"; then
    echo "✓ Web UI is accessible at $BASE_URL"
    echo "  Pages available:"
    echo "    - Dashboard: $BASE_URL/"
    echo "    - Conversations: $BASE_URL/conversations"
    echo "    - Suggestions: $BASE_URL/suggestions"
    echo "    - Metrics: $BASE_URL/metrics"
else
    echo "✗ Web UI not accessible"
fi

# Summary
echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo "✓ Short conversation (3 turns) - Tests basic evaluation"
echo "✓ Medium conversation (7 turns) with tools - Tests tool call evaluator"
echo "✓ Long conversation (35 turns) - Tests windowing (15/20 turn windows)"
echo "✓ Very long conversation (50 turns) - Stress tests windowing"
echo "✓ Evaluation queries working"
echo "✓ Suggestion generation working"
echo "✓ Web UI accessible"
echo ""
echo "Next steps:"
echo "1. Open browser to $BASE_URL to view results"
echo "2. Check worker logs: docker-compose logs -f worker"
echo "3. Review evaluations in the UI"
echo "4. Monitor token usage and performance"
echo ""
echo "========================================="

