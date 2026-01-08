#!/bin/bash
# Complete setup on Raspberry Pi
# Usage: curl -sSL https://raw.githubusercontent.com/.../scripts/setup-rpi.sh | bash

set -e

echo "=== Smart Home Assistant - Raspberry Pi Setup ==="
echo ""

# Update system
echo "Updating system..."
sudo apt-get update
sudo apt-get upgrade -y

# Install Docker
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com | sh
    sudo usermod -aG docker $USER
    echo "Docker installed. You need to restart session or run: newgrp docker"
fi

# Install Docker Compose
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "Installing Docker Compose..."
    sudo apt-get install -y docker-compose-plugin
fi

# Create project directory
PROJECT_DIR="$HOME/smart-home"
mkdir -p $PROJECT_DIR
cd $PROJECT_DIR

echo ""
echo "=== Downloading project ==="
# If you have the repo on GitHub:
# git clone https://github.com/your-username/smart-home.git .

echo ""
echo "=== Configuration ==="
if [ ! -f config.yaml ]; then
    cat > config.yaml << 'EOF'
audio:
  source: http
  http_addr: ":8080"

openai:
  api_key: "${OPENAI_API_KEY}"
  language: "en"

anthropic:
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-sonnet-4-20250514"

tuya:
  client_id: "${TUYA_CLIENT_ID}"
  secret: "${TUYA_SECRET}"
  region: "us"
  sync_interval: "5m"

pushover:
  enabled: false

log:
  level: "info"
  format: "text"
EOF
    echo "Created config.yaml - edit with your API keys"
fi

# Create .env
if [ ! -f .env ]; then
    cat > .env << 'EOF'
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
TUYA_CLIENT_ID=your-client-id
TUYA_SECRET=your-secret
EOF
    echo "Created .env - edit with your credentials"
fi

echo ""
echo "=== Next steps ==="
echo ""
echo "1. Edit credentials:"
echo "   nano $PROJECT_DIR/.env"
echo ""
echo "2. Setup Cloudflare Tunnel:"
echo "   ./scripts/setup-tunnel.sh home.yourdomain.com"
echo ""
echo "3. Start the service:"
echo "   docker compose up -d"
echo ""
echo "4. View logs:"
echo "   docker compose logs -f"
