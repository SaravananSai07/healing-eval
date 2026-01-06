#!/bin/bash
#
# Test Ollama Connection
# Usage: ./test_ollama.sh [ollama_url] [model]
# Example: ./test_ollama.sh http://your-vps-ip:11434 llama3.1:8b
#

OLLAMA_URL="${1:-http://localhost:11434}"
MODEL="${2:-llama3.1:8b}"

echo "================================================"
echo "🧪 Testing Ollama Connection"
echo "================================================"
echo "URL: $OLLAMA_URL"
echo "Model: $MODEL"
echo ""

# Test 1: Check if Ollama is reachable
echo "Test 1: Checking if Ollama is reachable..."
if curl -s -f "$OLLAMA_URL/api/tags" > /dev/null; then
    echo "✅ Ollama is reachable"
else
    echo "❌ Cannot reach Ollama at $OLLAMA_URL"
    echo "   Make sure Ollama is running and accessible"
    exit 1
fi

# Test 2: List available models
echo ""
echo "Test 2: Listing available models..."
MODELS=$(curl -s "$OLLAMA_URL/api/tags" | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
if [ -n "$MODELS" ]; then
    echo "✅ Available models:"
    echo "$MODELS" | while read -r model; do
        echo "   - $model"
    done
else
    echo "❌ No models found"
    exit 1
fi

# Test 3: Check if requested model is available
echo ""
echo "Test 3: Checking if $MODEL is available..."
if echo "$MODELS" | grep -q "^$MODEL$"; then
    echo "✅ Model $MODEL is available"
else
    echo "⚠️  Model $MODEL not found. Pull it first:"
    echo "   ollama pull $MODEL"
    exit 1
fi

# Test 4: Try a simple completion
echo ""
echo "Test 4: Testing completion with model $MODEL..."
RESPONSE=$(curl -s "$OLLAMA_URL/api/chat" \
    -H "Content-Type: application/json" \
    -d "{
        \"model\": \"$MODEL\",
        \"messages\": [{\"role\": \"user\", \"content\": \"Say 'test passed' and nothing else.\"}],
        \"stream\": false
    }")

if echo "$RESPONSE" | grep -q '"message"'; then
    CONTENT=$(echo "$RESPONSE" | grep -o '"content":"[^"]*"' | head -1 | cut -d'"' -f4)
    echo "✅ Completion successful"
    echo "   Response: $CONTENT"
else
    echo "❌ Completion failed"
    echo "   Response: $RESPONSE"
    exit 1
fi

# Test 5: Check response time
echo ""
echo "Test 5: Measuring response time..."
START=$(date +%s%N)
curl -s "$OLLAMA_URL/api/chat" \
    -H "Content-Type: application/json" \
    -d "{
        \"model\": \"$MODEL\",
        \"messages\": [{\"role\": \"user\", \"content\": \"Hi\"}],
        \"stream\": false
    }" > /dev/null
END=$(date +%s%N)
ELAPSED=$(( (END - START) / 1000000 ))
echo "✅ Response time: ${ELAPSED}ms"

if [ $ELAPSED -lt 5000 ]; then
    echo "   ⚡ Fast response!"
elif [ $ELAPSED -lt 15000 ]; then
    echo "   ✅ Good response time"
else
    echo "   ⚠️  Slow response (CPU inference is normal)"
fi

echo ""
echo "================================================"
echo "✅ All tests passed!"
echo "================================================"
echo ""
echo "Your Ollama instance is working correctly."
echo ""
echo "Add these to Render environment variables:"
echo "  OLLAMA_BASE_URL=$OLLAMA_URL"
echo "  OLLAMA_MODEL=$MODEL"
echo "  LLM_DEFAULT_PROVIDER=ollama"
echo ""

