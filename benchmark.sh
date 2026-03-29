#!/bin/bash

# Performance Benchmark Script for ReadPanda
# Compares Node.js and Go backend performance

echo "=========================================="
echo "ReadPanda Performance Benchmark"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if both backends are running
echo "🔍 Checking backend availability..."
echo ""

NODE_RUNNING=false
GO_RUNNING=false

# Check Node.js backend (port 3000)
if curl -s http://localhost:3000/api/v1/users > /dev/null 2>&1; then
    NODE_RUNNING=true
    echo -e "${GREEN}✓${NC} Node.js backend detected on port 3000"
else
    echo -e "${YELLOW}⚠${NC} Node.js backend not running on port 3000"
fi

# Check Go backend (port 3001 - or configure as needed)
if curl -s http://localhost:3001/api/v1/users > /dev/null 2>&1; then
    GO_RUNNING=true
    echo -e "${GREEN}✓${NC} Go backend detected on port 3001"
else
    echo -e "${YELLOW}⚠${NC} Go backend not running on port 3001"
    echo "   To test both, start Go backend on different port:"
    echo "   cd packages/api-go && PORT=3001 go run cmd/server/main.go"
fi

echo ""
echo "=========================================="

# Function to measure response time
measure_response_time() {
    local url=$1
    local name=$2
    
    echo -e "\n${BLUE}Testing:${NC} $name"
    echo "URL: $url"
    
    # Make 5 requests and get average
    local total=0
    local count=5
    
    for i in $(seq 1 $count); do
        # Use curl to measure time
        time=$(curl -o /dev/null -s -w '%{time_total}\n' "$url")
        total=$(echo "$total + $time" | bc)
        echo "  Request $i: ${time}s"
    done
    
    # Calculate average
    avg=$(echo "scale=4; $total / $count" | bc)
    echo -e "${GREEN}Average:${NC} ${avg}s"
    
    echo "$avg"
}

# Function to measure memory usage
measure_memory() {
    local port=$1
    local name=$2
    
    # Get process ID
    local pid=$(lsof -t -i:$port 2>/dev/null | head -1)
    
    if [ -z "$pid" ]; then
        echo "  Could not find process"
        return
    fi
    
    # Get memory usage (works on Linux)
    if command -v ps > /dev/null; then
        local mem=$(ps -p $pid -o rss= 2>/dev/null)
        if [ -n "$mem" ]; then
            local mem_mb=$(echo "scale=2; $mem / 1024" | bc)
            echo -e "${BLUE}$name Memory:${NC} ${mem_mb}MB"
        fi
    fi
}

echo ""
echo "=========================================="
echo "📊 Performance Tests"
echo "=========================================="

if [ "$NODE_RUNNING" = true ]; then
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Node.js Backend (port 3000)"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Measure memory
    measure_memory 3000 "Node.js"
    
    # Test endpoint
    NODE_TIME=$(measure_response_time "http://localhost:3000/api/v1/users" "Get Users Endpoint")
fi

if [ "$GO_RUNNING" = true ]; then
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Go Backend (port 3001)"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Measure memory
    measure_memory 3001 "Go"
    
    # Test endpoint
    GO_TIME=$(measure_response_time "http://localhost:3001/api/v1/users" "Get Users Endpoint")
fi

echo ""
echo "=========================================="
echo "📈 Comparison Summary"
echo "=========================================="

if [ "$NODE_RUNNING" = true ] && [ "$GO_RUNNING" = true ]; then
    echo ""
    echo "Response Time Comparison:"
    echo "  Node.js: ${NODE_TIME}s"
    echo "  Go:      ${GO_TIME}s"
    
    # Calculate speedup
    if command -v bc > /dev/null; then
        speedup=$(echo "scale=2; $NODE_TIME / $GO_TIME" | bc)
        echo ""
        echo -e "${GREEN}Go is ${speedup}x faster!${NC}"
    fi
    
    echo ""
    echo "💡 Tips for accurate benchmarking:"
    echo "   1. Run both backends with same database"
    echo "   2. Warm up servers with a few requests first"
    echo "   3. Use 'ab' or 'wrk' for load testing"
    echo "   4. Test under load (concurrent requests)"
    
elif [ "$NODE_RUNNING" = true ]; then
    echo ""
    echo "Only Node.js backend is running."
    echo "Start Go backend to compare: cd packages/api-go && PORT=3001 go run cmd/server/main.go"
    
elif [ "$GO_RUNNING" = true ]; then
    echo ""
    echo "Only Go backend is running."
    echo "Start Node.js backend to compare: cd packages/api && npm start"
    
else
    echo ""
    echo "No backends are running!"
    echo ""
    echo "Start backends:"
    echo "  Node.js: cd packages/api && npm start"
    echo "  Go:      cd packages/api-go && go run cmd/server/main.go"
fi

echo ""
echo "=========================================="
echo "For detailed load testing, use Apache Bench:"
echo "  ab -n 1000 -c 100 http://localhost:3000/api/v1/users"
echo ""
echo "Or use wrk for more advanced testing:"
echo "  wrk -t4 -c100 -d30s http://localhost:3000/api/v1/users"
echo "=========================================="
echo ""
