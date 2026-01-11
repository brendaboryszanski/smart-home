# Raspberry Pi Setup

This guide covers deploying the smart home assistant on a Raspberry Pi.

## Prerequisites

- Raspberry Pi 3/4/5 with Raspberry Pi OS
- MicroSD card (16GB+ recommended)
- Network connection (WiFi or Ethernet)

## 1. Install Docker

```bash
# SSH into your Raspberry Pi
ssh pi@raspberrypi.local

# Install Docker
curl -fsSL https://get.docker.com | sh

# Add your user to docker group
sudo usermod -aG docker $USER

# Logout and login again for group changes to take effect
exit
ssh pi@raspberrypi.local

# Verify Docker is working
docker --version
```

## 2. Install Docker Compose

```bash
sudo apt-get update
sudo apt-get install -y docker-compose-plugin

# Or install standalone docker-compose
sudo apt-get install -y docker-compose
```

## 3. Clone the Project

```bash
git clone https://github.com/your-username/smart-home.git
cd smart-home
```

## 4. Configure

```bash
cp config.example.yaml config.yaml
nano config.yaml
```

Update with your values:
- Home Assistant URL (use the RPi's IP or `localhost` if HA runs on same device)
- Home Assistant token
- Anthropic or Gemini API key
- Alexa auth token (if using Alexa)

## 5. Run with Docker Compose

```bash
# Using the Raspberry Pi optimized compose file
docker-compose -f docker-compose.rpi.yml up -d

# Or standard compose file
docker-compose up -d
```

## 6. View Logs

```bash
docker-compose logs -f
```

## 7. Auto-start on Boot

Docker containers with `restart: unless-stopped` will automatically start on boot.

To manually manage:

```bash
# Stop
docker-compose down

# Start
docker-compose up -d

# Restart
docker-compose restart
```

## Running Home Assistant on the Same Raspberry Pi

If you want to run both Home Assistant and this app on the same RPi:

```bash
# Create Home Assistant config directory
mkdir -p ~/homeassistant/config

# Run Home Assistant
docker run -d \
  --name homeassistant \
  --restart=unless-stopped \
  -v ~/homeassistant/config:/config \
  --network=host \
  ghcr.io/home-assistant/home-assistant:stable

# Wait for Home Assistant to start (first boot takes a few minutes)
# Then access http://YOUR_RPI_IP:8123 to complete setup
```

In your `config.yaml`, use:
```yaml
homeassistant:
  url: "http://localhost:8123"
  token: "your-token"
```

## Exposing to the Internet (for Alexa)

To receive commands from Alexa, you need to expose your RPi to the internet.

### Cloudflare Tunnel (Recommended - Free, No Domain Required)

1. **Create a tunnel in Cloudflare Dashboard**:
   - Go to https://dash.cloudflare.com
   - Create a free account
   - Go to **Zero Trust** → **Networks** → **Tunnels**
   - Click **Create a tunnel** → Select **Cloudflared** → Name it `smart-home`
   - Copy the **tunnel token**
   - Add a public hostname:
     - Subdomain: `smart-home`
     - Domain: `cfargotunnel.com` (free)
     - Service: `HTTP` → `localhost:8080`

2. **Install and run on RPi**:
```bash
# Install cloudflared
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64 -o cloudflared
chmod +x cloudflared
sudo mv cloudflared /usr/local/bin/

# Install as service with your tunnel token
sudo cloudflared service install YOUR_TUNNEL_TOKEN

# Enable and start
sudo systemctl enable cloudflared
sudo systemctl start cloudflared
```

Your free URL will be: `https://smart-home-XXXXX.cfargotunnel.com`

See [alexa/SETUP.md](../alexa/SETUP.md) for complete Alexa skill setup instructions.

## Troubleshooting

### Container won't start

```bash
# Check logs
docker-compose logs

# Check if port is in use
sudo netstat -tlnp | grep 8080
```

### Can't connect to Home Assistant

```bash
# Test connection from RPi
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8123/api/states | head

# Check if Home Assistant is running
docker ps | grep homeassistant
```

### Out of memory

Raspberry Pi 3 has limited RAM. If you run into memory issues:

```bash
# Check memory usage
free -h

# Add swap space
sudo dphys-swapfile swapoff
sudo nano /etc/dphys-swapfile  # Change CONF_SWAPSIZE=2048
sudo dphys-swapfile setup
sudo dphys-swapfile swapon
```

## Updating

```bash
cd smart-home
git pull
docker-compose down
docker-compose up -d --build
```
