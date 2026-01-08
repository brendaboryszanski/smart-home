#!/bin/bash
# Setup Cloudflare Tunnel on Raspberry Pi
# Usage: ./setup-tunnel.sh home.yourdomain.com

set -e

HOSTNAME=${1:-""}
TUNNEL_NAME="smart-home"

if [ -z "$HOSTNAME" ]; then
    echo "Usage: $0 <hostname>"
    echo "Example: $0 home.mydomain.com"
    exit 1
fi

echo "=== Installing Cloudflare Tunnel ==="

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    aarch64|arm64)
        CLOUDFLARED_URL="https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64"
        ;;
    armv7l|armhf)
        CLOUDFLARED_URL="https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm"
        ;;
    x86_64)
        CLOUDFLARED_URL="https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Download cloudflared
echo "Downloading cloudflared for $ARCH..."
sudo curl -L "$CLOUDFLARED_URL" -o /usr/local/bin/cloudflared
sudo chmod +x /usr/local/bin/cloudflared

# Verify installation
cloudflared --version

echo ""
echo "=== Authenticating with Cloudflare ==="
echo "A browser will open for authentication (or copy the link)"
echo ""
cloudflared tunnel login

echo ""
echo "=== Creating tunnel ==="
cloudflared tunnel create $TUNNEL_NAME

# Get tunnel ID
TUNNEL_ID=$(cloudflared tunnel list | grep $TUNNEL_NAME | awk '{print $1}')
echo "Tunnel created with ID: $TUNNEL_ID"

echo ""
echo "=== Configuring DNS ==="
cloudflared tunnel route dns $TUNNEL_NAME $HOSTNAME

# Create config directory
mkdir -p ~/.cloudflared

# Create configuration file
cat > ~/.cloudflared/config.yml << EOF
tunnel: $TUNNEL_ID
credentials-file: $HOME/.cloudflared/$TUNNEL_ID.json

ingress:
  - hostname: $HOSTNAME
    service: http://localhost:8080
  - service: http_status:404
EOF

echo ""
echo "=== Configuration created ==="
cat ~/.cloudflared/config.yml

echo ""
echo "=== Installing as service ==="
sudo cloudflared service install
sudo systemctl enable cloudflared
sudo systemctl start cloudflared

echo ""
echo "=== Done! ==="
echo ""
echo "Your Smart Home is available at: https://$HOSTNAME"
echo ""
echo "Check status:"
echo "  sudo systemctl status cloudflared"
echo ""
echo "View logs:"
echo "  sudo journalctl -u cloudflared -f"
echo ""
echo "Test:"
echo "  curl https://$HOSTNAME/health"
