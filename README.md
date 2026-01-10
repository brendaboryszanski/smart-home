# Smart Home Voice Assistant

A voice-controlled smart home assistant for Raspberry Pi that integrates with Home Assistant using natural language processing.

## Features

- **Multiple audio sources**: HTTP endpoint, file watcher, or USB microphone
- **Speech-to-text**: OpenAI Whisper API (optional - not needed for Alexa)
- **Natural language understanding**: Claude or Gemini API for intent parsing
- **Device control**: Home Assistant integration (works with Tuya Local, and many other integrations)
- **Alexa integration**: Custom skill support for voice commands
- **Notifications**: Optional Pushover push notifications

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Audio Input │────▶│   Whisper   │────▶│Claude/Gemini│────▶│    Home     │
│ (Alexa/Mic) │     │    (STT)    │     │    (NLU)    │     │  Assistant  │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

> **Note**: When using Alexa, Whisper is not needed since Alexa sends text directly.

### Project Structure

```
smart-home/
├── cmd/assistant/          # Application entrypoint
├── internal/
│   ├── domain/             # Business entities
│   ├── application/        # Use cases and interfaces
│   └── infra/              # External service implementations
│       ├── audio/          # Audio sources (HTTP, file, microphone)
│       ├── openai/         # Whisper client
│       ├── anthropic/      # Claude client
│       ├── gemini/         # Gemini client
│       ├── homeassistant/  # Home Assistant client
│       └── pushover/       # Push notifications
├── alexa/                  # Alexa skill configuration
├── config/                 # Configuration loader
└── tests/                  # Integration tests
```

## Quick Start

### Prerequisites

- Go 1.23+
- Docker and Docker Compose
- Home Assistant instance with your devices configured
- API key for Anthropic (Claude) OR Google (Gemini)
- **Optional:** OpenAI API key (only needed for microphone/audio file input)

### 1. Setup Home Assistant

See [Home Assistant Setup](#home-assistant-setup) section below.

### 2. Clone and configure

```bash
git clone https://github.com/your-username/smart-home.git
cd smart-home

cp config.example.yaml config.yaml
# Edit config.yaml with your Home Assistant URL and token
```

### 3. Run with Docker

```bash
docker-compose up --build
```

### 4. Send a command

```bash
# Via HTTP text command
curl -X POST http://localhost:8080/alexa -d "turn on living room light"

# Or via /text endpoint
curl -X POST http://localhost:8080/text -d "prende la luz del living"
```

---

## Home Assistant Setup

This app uses Home Assistant as the smart home backend. Home Assistant supports hundreds of integrations including Tuya, Xiaomi, Philips Hue, and more.

### Option 1: Docker (for testing on Mac/PC)

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

Access http://localhost:8123 and create your account.

### Option 2: Raspberry Pi (production)

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Logout and login again, then:
mkdir -p ~/homeassistant/config

docker run -d \
  --name homeassistant \
  --restart=unless-stopped \
  -v ~/homeassistant/config:/config \
  --network=host \
  ghcr.io/home-assistant/home-assistant:stable
```

Access http://YOUR_RPI_IP:8123

### Install HACS (optional, for extra integrations)

```bash
docker exec homeassistant bash -c "wget -O - https://get.hacs.xyz | bash -"
docker restart homeassistant
```

Then in Home Assistant: Settings → Devices & Services → Add Integration → Search "HACS"

### Add your Tuya devices

1. Go to **Settings → Devices & Services → Add Integration**
2. Search for **"Tuya"** (the official integration)
3. Login with your Tuya/SmartLife app credentials
4. Your devices and scenes will be imported automatically

### Generate Access Token

1. In Home Assistant, click your username (bottom left)
2. Go to **Security** tab
3. Under **Long-Lived Access Tokens**, click **Create Token**
4. Give it a name (e.g., "smart-home-app")
5. Copy the token (only shown once!)

### Migrate config to Raspberry Pi

The Home Assistant config is stored in the `/config` volume. To migrate from Mac to Raspberry Pi:

```bash
# On Mac: copy config to RPi
scp -r ~/homeassistant/config pi@YOUR_RPI_IP:~/homeassistant/

# On RPi: start Home Assistant pointing to that config
docker run -d \
  --name homeassistant \
  --restart=unless-stopped \
  -v ~/homeassistant/config:/config \
  --network=host \
  ghcr.io/home-assistant/home-assistant:stable
```

All your integrations, users, and settings will be preserved.

---

## Configuration

### Minimal config (Alexa + Home Assistant)

```yaml
audio:
  source: http
  http_addr: ":8080"
  auth_token: "your-alexa-auth-token"  # Generate: openssl rand -hex 32

# LLM - choose ONE
anthropic:
  api_key: "your-anthropic-key"
  model: "claude-sonnet-4-20250514"

# OR use Gemini instead:
# gemini:
#   api_key: "your-gemini-key"
#   model: "gemini-2.0-flash"

homeassistant:
  url: "http://YOUR_HOMEASSISTANT_IP:8123"
  token: "your-long-lived-access-token"
  sync_interval: "5m"

log:
  level: "info"
```

### Full config with all options

```yaml
audio:
  source: http              # http, file, or microphone
  http_addr: ":8080"
  auth_token: ""            # Optional auth for /alexa endpoint
  file_dir: "./audio"       # Only for file source
  wake_word: "home"         # Only for microphone source
  sample_rate: 16000        # Only for microphone source

# LLM for intent parsing - configure ONE
anthropic:
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-sonnet-4-20250514"

# gemini:
#   api_key: "${GEMINI_API_KEY}"
#   model: "gemini-2.0-flash"

# Speech-to-text - only needed for audio input (not needed for Alexa)
# openai:
#   api_key: "${OPENAI_API_KEY}"
#   language: "es"

homeassistant:
  url: "http://192.168.1.X:8123"
  token: "${HOMEASSISTANT_TOKEN}"
  sync_interval: "5m"

pushover:
  enabled: false
  token: "${PUSHOVER_TOKEN}"
  user_key: "${PUSHOVER_USER_KEY}"

log:
  level: "info"   # debug, info, warn, error
  format: "text"  # text, json
```

---

## Audio Sources

| Source | Config | Description |
|--------|--------|-------------|
| `http` | `audio.source: http` | REST endpoint for audio/text (default) |
| `file` | `audio.source: file` | Watch directory for audio files |
| `microphone` | `audio.source: microphone` | USB microphone with wake word |

### HTTP Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/audio` | POST | Send audio file (WAV, MP3, M4A) |
| `/text` | POST | Send text command directly |
| `/alexa` | POST | Alexa skill webhook |
| `/health` | GET | Health check |

---

## Alexa Integration

See [alexa/SETUP.md](alexa/SETUP.md) for complete setup instructions.

**Example usage:**
> "Alexa, amor prende la luz del living"
> "Alexa, amor apaga todo"

---

## Deployment on Raspberry Pi

### 1. Install Docker

```bash
ssh pi@raspberrypi.local
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Logout and login again
```

### 2. Clone and configure

```bash
git clone https://github.com/your-username/smart-home.git
cd smart-home
cp config.example.yaml config.yaml
# Edit config.yaml
```

### 3. Run

```bash
docker-compose -f docker-compose.rpi.yml up -d
```

### 4. Expose to internet (for Alexa)

Use Cloudflare Tunnel to securely expose your RPi:

```bash
./scripts/setup-tunnel.sh home.yourdomain.com
```

---

## Development

### Run locally

```bash
go run ./cmd/assistant -config config.yaml
```

### Run tests

```bash
go test ./...
```

### Build

```bash
# Standard build
go build -o bin/assistant ./cmd/assistant

# Build with microphone support (requires PortAudio)
go build -tags portaudio -o bin/assistant ./cmd/assistant
```
