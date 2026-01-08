.PHONY: build run test lint docker docker-rpi clean

# Build
build:
	go build -o bin/assistant ./cmd/assistant

# Run locally
run:
	go run ./cmd/assistant -config config.yaml

# Test
test:
	go test -v ./...

# Lint
lint:
	golangci-lint run

# Docker (for development/testing on Mac/Linux)
docker:
	docker compose build

docker-up:
	docker compose up

docker-down:
	docker compose down

# Docker for Raspberry Pi (with audio support)
docker-rpi:
	docker compose -f docker-compose.rpi.yml build

docker-rpi-up:
	docker compose -f docker-compose.rpi.yml up

# Test audio via HTTP (from your phone or curl)
# Record on phone, then: curl -X POST -H "Content-Type: audio/wav" --data-binary @audio.wav http://localhost:8080/audio
test-audio:
	@echo "Send audio via:"
	@echo "  curl -X POST --data-binary @your-audio.wav http://localhost:8080/audio"
	@echo ""
	@echo "Or check health:"
	@echo "  curl http://localhost:8080/health"

# Clean
clean:
	rm -rf bin/
	docker compose down --rmi local

# Setup config from example
setup:
	@if [ ! -f config.yaml ]; then \
		cp config.example.yaml config.yaml; \
		echo "Created config.yaml from example. Please edit with your API keys."; \
	else \
		echo "config.yaml already exists"; \
	fi

