# Home Assistant Setup

This guide covers setting up Home Assistant to work with the smart home assistant.

## What is Home Assistant?

Home Assistant is an open-source home automation platform. It supports thousands of devices and services, including Tuya, Xiaomi, Philips Hue, and many more.

This app uses Home Assistant as the backend to control your smart home devices.

## Installation Options

### Option 1: Docker (Recommended for testing)

```bash
# Create config directory
mkdir -p ~/homeassistant/config

# Run Home Assistant
docker run -d \
  --name homeassistant \
  --restart=unless-stopped \
  -v ~/homeassistant/config:/config \
  -p 8123:8123 \
  ghcr.io/home-assistant/home-assistant:stable
```

Access: http://localhost:8123

### Option 2: Docker on Raspberry Pi

```bash
# Create config directory
mkdir -p ~/homeassistant/config

# Run with host networking (better device discovery)
docker run -d \
  --name homeassistant \
  --restart=unless-stopped \
  -v ~/homeassistant/config:/config \
  --network=host \
  ghcr.io/home-assistant/home-assistant:stable
```

Access: http://YOUR_RPI_IP:8123

### Option 3: Home Assistant OS (Dedicated device)

For a dedicated Home Assistant device, install Home Assistant OS:
https://www.home-assistant.io/installation/

## Initial Setup

1. Open Home Assistant in your browser
2. Create your admin account
3. Set your home location and timezone
4. Home Assistant will auto-discover some devices on your network

## Adding Tuya Devices

The official Tuya integration uses your Tuya/SmartLife app credentials (not the developer portal).

1. Go to **Settings** -> **Devices & Services**
2. Click **+ Add Integration**
3. Search for **"Tuya"**
4. Select your country and enter your Tuya/SmartLife app credentials
5. Your devices and scenes will be imported automatically

## Installing HACS (Optional)

HACS (Home Assistant Community Store) gives you access to additional integrations not included by default.

```bash
# Run this command to install HACS
docker exec homeassistant bash -c "wget -O - https://get.hacs.xyz | bash -"

# Restart Home Assistant
docker restart homeassistant
```

Then in Home Assistant:
1. Go to **Settings** -> **Devices & Services**
2. Click **+ Add Integration**
3. Search for **"HACS"**
4. Follow the GitHub authentication steps

## Generating Access Token

The smart home app needs a long-lived access token to communicate with Home Assistant.

1. Click your username in the bottom left corner
2. Go to the **Security** tab
3. Scroll down to **Long-Lived Access Tokens**
4. Click **Create Token**
5. Give it a name (e.g., "smart-home-app")
6. **Copy the token immediately** - it's only shown once!

## Configuration

In your `config.yaml`:

```yaml
homeassistant:
  url: "http://YOUR_HOME_ASSISTANT_IP:8123"
  token: "your-long-lived-access-token"
  sync_interval: "5m"
```

## Migrating to Another Device

The Home Assistant config is stored in the `/config` directory (the Docker volume). To migrate:

### From Mac to Raspberry Pi

```bash
# On Mac: copy config to RPi
scp -r ~/homeassistant/config pi@YOUR_RPI_IP:~/homeassistant/

# On RPi: start Home Assistant
docker run -d \
  --name homeassistant \
  --restart=unless-stopped \
  -v ~/homeassistant/config:/config \
  --network=host \
  ghcr.io/home-assistant/home-assistant:stable
```

All integrations, users, automations, and settings will be preserved.

### Important Notes

- The token you generated will still work after migration
- Device IPs might change - some integrations may need reconfiguration
- If using mDNS names (like `homeassistant.local`), ensure your network supports it

## Useful Home Assistant Features

### Scenes

Scenes let you set multiple devices to specific states with one command. You can create them in:
- **Settings** -> **Automations & Scenes** -> **Scenes**

Example: "Movie Night" scene that dims lights and turns on TV.

### Automations

Automate actions based on triggers:
- **Settings** -> **Automations & Scenes** -> **Automations**

Example: Turn on porch light at sunset.

### Areas

Organize devices by room:
- **Settings** -> **Areas & Zones**

This helps the AI understand commands like "turn off the kitchen lights".

## Troubleshooting

### Can't connect to Home Assistant

```bash
# Check if container is running
docker ps | grep homeassistant

# View logs
docker logs homeassistant

# Test API
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8123/api/states | head
```

### Token not working

- Tokens don't expire, but they can be revoked
- Check if the token still exists in your profile
- Create a new token if needed

### Devices not showing

- Check if the integration is configured correctly
- Try reloading the integration: **Settings** -> **Devices & Services** -> (your integration) -> **Reload**
- Check the Home Assistant logs for errors

### Container using too much memory

```bash
# Check memory usage
docker stats homeassistant

# Restart to free memory
docker restart homeassistant
```
