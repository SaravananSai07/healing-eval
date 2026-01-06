#!/bin/bash
#
# Ollama VPS Setup Script
# Run this on your free VPS (Alavps, VPS Mart, etc.)
# Usage: ./setup_ollama_vps.sh [model_name]
# Example: ./setup_ollama_vps.sh llama3.1:8b
#

set -e

MODEL="${1:-llama3.1:8b}"

echo "================================================"
echo "🚀 Ollama VPS Setup Script"
echo "================================================"
echo "Model: $MODEL"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "⚠️  This script needs sudo access. Rerun with sudo or as root."
    exit 1
fi

# Install Ollama
echo "📦 Installing Ollama..."
if command -v ollama &> /dev/null; then
    echo "✅ Ollama already installed"
else
    curl -fsSL https://ollama.com/install.sh | sh
    echo "✅ Ollama installed"
fi

# Pull the model
echo ""
echo "📥 Pulling model: $MODEL (this may take a few minutes)..."
ollama pull "$MODEL"
echo "✅ Model pulled"

# Configure Ollama for external access
echo ""
echo "⚙️  Configuring Ollama for external connections..."
systemctl stop ollama 2>/dev/null || true

mkdir -p /etc/systemd/system/ollama.service.d
cat << 'EOF' > /etc/systemd/system/ollama.service.d/override.conf
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
EOF

systemctl daemon-reload
systemctl start ollama
systemctl enable ollama

echo "✅ Ollama configured"

# Configure firewall
echo ""
echo "🔥 Configuring firewall..."
if command -v ufw &> /dev/null; then
    ufw allow 11434/tcp
    ufw --force enable
    echo "✅ Firewall configured (port 11434 open)"
else
    echo "⚠️  UFW not found. Manually open port 11434 in your firewall."
fi

# Test Ollama
echo ""
echo "🧪 Testing Ollama..."
sleep 3
if curl -s http://localhost:11434/api/tags > /dev/null; then
    echo "✅ Ollama is responding"
else
    echo "❌ Ollama test failed. Check logs: journalctl -u ollama -f"
    exit 1
fi

# Get public IP
PUBLIC_IP=$(curl -s ifconfig.me || curl -s icanhazip.com || echo "UNKNOWN")

echo ""
echo "================================================"
echo "✅ Setup Complete!"
echo "================================================"
echo ""
echo "Your Ollama is running at:"
echo "  http://$PUBLIC_IP:11434"
echo ""
echo "Add these to your Render environment variables:"
echo "  OLLAMA_BASE_URL=http://$PUBLIC_IP:11434"
echo "  OLLAMA_MODEL=$MODEL"
echo "  LLM_DEFAULT_PROVIDER=ollama"
echo ""
echo "Test it:"
echo "  curl http://$PUBLIC_IP:11434/api/tags"
echo ""
echo "⚠️  SECURITY WARNING:"
echo "  This Ollama instance is publicly accessible!"
echo "  For production, use Tailscale or a VPN."
echo ""
echo "View logs:"
echo "  sudo journalctl -u ollama -f"
echo ""
echo "Stop Ollama:"
echo "  sudo systemctl stop ollama"
echo ""
echo "Available models:"
ollama list
echo ""
echo "================================================"

