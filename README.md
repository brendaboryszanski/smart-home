# Smart Home Voice Assistant

A voice-controlled smart home assistant for Raspberry Pi that integrates with Tuya devices using natural language processing.

## Features

- 🎤 **Multiple audio sources**: HTTP endpoint, file watcher, or USB microphone
- 🗣️ **Speech-to-text**: OpenAI Whisper API
- 🧠 **Natural language understanding**: Claude API for intent parsing
- 💡 **Device control**: Tuya Cloud API integration
- 📱 **Alexa integration**: Custom skill support for voice commands
- 🔔 **Notifications**: Optional Pushover push notifications

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Audio Input │────▶│   Whisper   │────▶│   Claude    │────▶│    Tuya     │
│ (Alexa/Mic) │     │    (STT)    │     │    (NLU)    │     │   (IoT)     │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

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
│       ├── tuya/           # Tuya API client
│       └── pushover/       # Push notifications
├── alexa/                  # Alexa skill configuration
├── scripts/                # Setup scripts for Raspberry Pi
├── config/                 # Configuration loader
└── tests/                  # Integration tests
```

## Quick Start

### Prerequisites

- Go 1.23+
- Docker and Docker Compose (recommended for deployment)
- API keys for:
  - OpenAI (Whisper)
  - Anthropic (Claude)
  - Tuya Cloud
- **Optional:** PortAudio library (only needed if using `microphone` audio source)
  - macOS: `brew install portaudio`
  - Linux: `sudo apt-get install portaudio19-dev`
  - Not needed for HTTP or file-based audio sources

### 1. Clone and configure

```bash
git clone https://github.com/your-username/smart-home.git
cd smart-home

cp config.example.yaml config.yaml
# Edit config.yaml with your API keys
```

### 2. Run with Docker

```bash
docker compose up --build
```

### 3. Send a command

```bash
# Via HTTP (from phone or curl)
curl -X POST http://localhost:8080/alexa -d "turn on living room light"

# Or send audio file
curl -X POST --data-binary @audio.wav http://localhost:8080/audio
```

## Audio Sources

The assistant supports multiple audio input methods:

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

## Alexa Integration

See [alexa/SETUP.md](alexa/SETUP.md) for complete step-by-step instructions including:
- Setting up Cloudflare Tunnel to expose your RPI
- Configuring authentication token (required)
- Creating the Alexa skill
- AWS Lambda function setup

**Two architecture options:**
1. **Alexa → Lambda → RPI** (recommended, includes Alexa signature verification)
2. **Alexa → RPI direct** (simpler but less secure, skips Lambda)

**Example usage:**
> "Alexa, amor prende la luz del living"
> "Alexa, amor apaga todo"

## Configuration

```yaml
audio:
  source: http              # http, file, or microphone
  http_addr: ":8080"
  
openai:
  api_key: "${OPENAI_API_KEY}"
  language: "en"            # or "es" for Spanish

anthropic:
  api_key: "${ANTHROPIC_API_KEY}"
  model: "claude-sonnet-4-20250514"

tuya:
  client_id: "${TUYA_CLIENT_ID}"
  secret: "${TUYA_SECRET}"
  region: "us"              # us, eu, cn, in
  sync_interval: "5m"

pushover:
  enabled: false
  token: "${PUSHOVER_TOKEN}"
  user_key: "${PUSHOVER_USER_KEY}"
```

## Deployment on Raspberry Pi

### Option 1: Using setup script

```bash
ssh pi@raspberrypi.local
curl -sSL https://raw.githubusercontent.com/.../scripts/setup-rpi.sh | bash
```

### Option 2: Manual setup

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Clone and run
git clone https://github.com/your-username/smart-home.git
cd smart-home
cp config.example.yaml config.yaml
docker compose -f docker-compose.rpi.yml up -d
```

### Cloudflare Tunnel (expose to internet securely)

```bash
./scripts/setup-tunnel.sh home.yourdomain.com
```

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
# Standard build (HTTP and file sources only)
go build -o bin/assistant ./cmd/assistant

# Build with microphone support (requires PortAudio)
go build -tags portaudio -o bin/assistant ./cmd/assistant
```