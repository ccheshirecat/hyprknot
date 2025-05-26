#!/bin/bash

# HyprKnot API Test Script
# This script demonstrates the API functionality

set -e

API_BASE="http://localhost:8080"
API_KEY="test-api-key-12345"

echo "🚀 HyprKnot API Test Script"
echo "================================"

# Function to make API calls
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    
    if [ -n "$data" ]; then
        curl -s -X "$method" \
             -H "X-API-Key: $API_KEY" \
             -H "Content-Type: application/json" \
             -d "$data" \
             "$API_BASE$endpoint" | jq .
    else
        curl -s -X "$method" \
             -H "X-API-Key: $API_KEY" \
             "$API_BASE$endpoint" | jq .
    fi
}

# Test health check (no auth required)
echo "📊 Testing health check..."
curl -s "$API_BASE/health" | jq .
echo

# Test API documentation
echo "📚 Getting API documentation..."
api_call "GET" "/api/v1/docs"
echo

# Test zones listing
echo "🌐 Listing zones..."
api_call "GET" "/api/v1/zones"
echo

# Example: Create an A record
echo "➕ Example: Creating A record..."
api_call "POST" "/api/v1/zones/example.com/records" '{
  "name": "test-vm",
  "type": "A",
  "ttl": 300,
  "data": "192.168.1.100"
}'
echo

# Example: Create PTR record
echo "🔄 Example: Creating PTR record..."
api_call "POST" "/api/v1/zones/1.168.192.in-addr.arpa/records" '{
  "name": "100.1.168.192.in-addr.arpa",
  "type": "PTR", 
  "ttl": 300,
  "data": "test-vm.example.com."
}'
echo

# Example: List records in zone
echo "📋 Example: Listing records in zone..."
api_call "GET" "/api/v1/zones/example.com/records"
echo

# Example: Get specific record
echo "🔍 Example: Getting specific record..."
api_call "GET" "/api/v1/zones/example.com/records/test-vm/A"
echo

# Example: Update record
echo "✏️  Example: Updating record..."
api_call "PUT" "/api/v1/zones/example.com/records/test-vm/A" '{
  "ttl": 600,
  "data": "192.168.1.101"
}'
echo

# Example: Delete record
echo "🗑️  Example: Deleting record..."
api_call "DELETE" "/api/v1/zones/example.com/records/test-vm/A"
echo

# Example: Reload zone
echo "🔄 Example: Reloading zone..."
api_call "POST" "/api/v1/zones/example.com/reload"
echo

echo "✅ API test complete!"
echo
echo "💡 Tips:"
echo "   - Modify config.yaml to set your API keys"
echo "   - Configure allowed_zones to restrict access"
echo "   - Check logs with: journalctl -u hyprknot -f"
echo "   - API docs available at: $API_BASE/api/v1/docs"
