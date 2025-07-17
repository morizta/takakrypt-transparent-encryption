#!/bin/bash

echo "🚀 Deploy Takakrypt Agent"
echo "========================"

if [ "$EUID" -ne 0 ]; then
    echo "❌ Run as root: sudo ./deploy-agent.sh"
    exit 1
fi

AGENT_DIR="/opt/takakrypt"
CONFIG_DIR="$AGENT_DIR/config"

echo "📁 Creating directories..."
mkdir -p $AGENT_DIR $CONFIG_DIR /var/log/takakrypt

echo "📋 Copying config..."
cp ubuntu-config/*.json $CONFIG_DIR/

echo "🔨 Building agent..."
if [ -d "/tmp/takakrypt-src" ]; then
    cd /tmp/takakrypt-src
elif [ -d "../" ]; then
    cd ../
else
    echo "❌ Source not found. Copy source to /tmp/takakrypt-src first:"
    echo "scp -r . user@vm:/tmp/takakrypt-src"
    exit 1
fi

# Build from wherever main.go is
if [ -f "cmd/agent/main.go" ]; then
    go build -o $AGENT_DIR/takakrypt-agent ./cmd/agent
elif [ -f "main.go" ]; then
    go build -o $AGENT_DIR/takakrypt-agent .
else
    echo "❌ Cannot find main.go"
    exit 1
fi

chmod +x $AGENT_DIR/takakrypt-agent

echo "🔧 Creating service..."
cat > /etc/systemd/system/takakrypt.service << EOF
[Unit]
Description=Takakrypt Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=$AGENT_DIR/takakrypt-agent --config $CONFIG_DIR --log-level info
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable takakrypt

echo "📝 Creating start script..."
cat > $AGENT_DIR/start.sh << 'EOF'
#!/bin/bash
echo "🚀 Starting agent..."
systemctl start takakrypt
sleep 3
systemctl status takakrypt --no-pager
echo ""
mount | grep data || echo "Mounts will appear after startup"
EOF
chmod +x $AGENT_DIR/start.sh

echo "✅ Deploy complete!"
echo ""
echo "Start: sudo $AGENT_DIR/start.sh"
echo "Logs:  journalctl -u takakrypt -f"