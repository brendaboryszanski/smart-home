# Smart Home Voice Assistant

A voice-controlled smart home assistant for Raspberry Pi that integrates with Home Assistant using natural language processing.

## Features

- **Multiple audio sources**: HTTP endpoint, file watcher, or USB microphone
- **Speech-to-text**: OpenAI Whisper API (optional - not needed for Alexa)
- **Natural language understanding**: Claude or Gemini API for intent parsing
- **Device control**: Home Assistant integration (works with Tuya, and 2000+ other integrations)
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

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Home Assistant instance with your devices configured
- API key for Anthropic (Claude) OR Google (Gemini)

### 1. Setup Home Assistant

See [docs/HOME_ASSISTANT.md](docs/HOME_ASSISTANT.md) for complete setup guide.

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
curl -X POST http://localhost:8080/text -d "turn on living room light"
```

## Documentation

| Guide | Description |
|-------|-------------|
| [Home Assistant Setup](docs/HOME_ASSISTANT.md) | Installing and configuring Home Assistant |
| [Raspberry Pi Setup](docs/RASPBERRY_PI.md) | Deploying on Raspberry Pi |
| [Alexa Integration](alexa/SETUP.md) | Setting up the Alexa skill |

## Configuration

### Minimal config (Alexa + Home Assistant)

```yaml
audio:
  source: http
  http_addr: ":8080"

# LLM - choose ONE
anthropic:
  api_key: "your-anthropic-key"

# OR use Gemini:
# gemini:
#   api_key: "your-gemini-key"

homeassistant:
  url: "http://YOUR_IP:8123"
  token: "your-token"
```

See [config.example.yaml](config.example.yaml) for all options.

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

## Development

```bash
# Run locally
go run ./cmd/assistant -config config.yaml

# Run tests
go test ./...

# Build
go build -o bin/assistant ./cmd/assistant
```

## Project Structure

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
├── docs/                   # Documentation
├── config/                 # Configuration loader
└── tests/                  # Integration tests
```
