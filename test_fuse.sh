#!/bin/bash

echo "🚀 Testing Takakrypt Transparent Encryption Agent with FUSE"

# Clean up any existing mounts
umount /tmp/test-mount 2>/dev/null || true

# Build the agent
echo "📦 Building agent..."
go build -o agent ./cmd/agent

# Run agent in background for 5 seconds
echo "🔧 Starting agent..."
./agent --config ./ --log-level info &
AGENT_PID=$!

# Wait a moment for startup
sleep 2

# Check if mount is active
echo "🔍 Checking mount status..."
if mount | grep -q "/tmp/test-mount"; then
    echo "✅ FUSE mount is active!"
    
    # Test basic operations
    echo "📁 Testing file operations..."
    ls -la /tmp/test-mount/
    
    echo "📝 Creating test file..."
    echo "Hello from Takakrypt!" > /tmp/test-mount/test.txt 2>/dev/null && echo "✅ File created" || echo "❌ File creation failed"
    
    echo "📖 Reading test file..."
    cat /tmp/test-mount/test.txt 2>/dev/null && echo "✅ File read" || echo "❌ File read failed"
    
    echo "🗂️ Listing files..."
    ls -la /tmp/test-mount/
else
    echo "❌ FUSE mount not found"
fi

# Kill agent
echo "⏹️ Stopping agent..."
kill $AGENT_PID 2>/dev/null

# Clean up
sleep 1
umount /tmp/test-mount 2>/dev/null || true

echo "✨ Test completed!"