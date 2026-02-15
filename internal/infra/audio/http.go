package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"smart-home/internal/domain"
)

type HTTPSource struct {
	addr        string
	server      *http.Server
	audioChan   chan []byte
	logger      *slog.Logger
	mu          sync.Mutex
	running     bool
	mux         *http.ServeMux
	closeOnce   sync.Once
	rateLimiter *RateLimiter
	authToken   string
}

func NewHTTPSource(addr string, authToken string, logger *slog.Logger) *HTTPSource {
	h := &HTTPSource{
		addr:        addr,
		audioChan:   make(chan []byte, 10),
		logger:      logger,
		mux:         http.NewServeMux(),
		rateLimiter: NewRateLimiter(30, time.Minute), // 30 requests per minute per IP
		authToken:   authToken,
	}
	// Apply rate limiting to command endpoints
	h.mux.HandleFunc("POST /audio", h.rateLimiter.Middleware(h.handleAudio))
	h.mux.HandleFunc("POST /text", h.rateLimiter.Middleware(h.handleText))
	h.mux.HandleFunc("POST /alexa", h.rateLimiter.Middleware(h.handleAlexa))
	// No rate limiting on health check
	h.mux.HandleFunc("GET /health", h.handleHealth)
	return h
}

func (h *HTTPSource) Name() string {
	return "http"
}

func (h *HTTPSource) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return nil
	}

	h.server = &http.Server{
		Addr:         h.addr,
		Handler:      h.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		h.logger.Info("HTTP audio server starting", "addr", h.addr)
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.Error("HTTP server error", "error", err)
		}
	}()

	h.running = true
	return nil
}

func (h *HTTPSource) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return nil
	}

	if h.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.server.Shutdown(ctx); err != nil {
			h.logger.Warn("graceful shutdown failed, forcing close", "error", err)
			if err := h.server.Close(); err != nil {
				return fmt.Errorf("closing server: %w", err)
			}
		}
	}

	h.closeOnce.Do(func() {
		close(h.audioChan)
	})
	h.running = false
	return nil
}

func (h *HTTPSource) NextCommand(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case audio, ok := <-h.audioChan:
		if !ok {
			return nil, fmt.Errorf("audio channel closed")
		}
		return audio, nil
	}
}

func (h *HTTPSource) Handler() http.Handler {
	return h.mux
}

func (h *HTTPSource) InjectAudio(data []byte) {
	select {
	case h.audioChan <- data:
	default:
	}
}

func (h *HTTPSource) handleAudio(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
	if err != nil {
		h.logger.Error("reading audio body", "error", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(data) == 0 {
		http.Error(w, "empty audio", http.StatusBadRequest)
		return
	}

	select {
	case h.audioChan <- data:
		h.logger.Info("received audio via HTTP", "bytes", len(data))
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"status":"received","bytes":%d}`, len(data))
	default:
		http.Error(w, "queue full, try again", http.StatusServiceUnavailable)
	}
}

func (h *HTTPSource) handleText(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1024))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	text := string(data)
	if text == "" {
		http.Error(w, "empty text", http.StatusBadRequest)
		return
	}

	marker := []byte(domain.TextCommandPrefix + text)

	select {
	case h.audioChan <- marker:
		h.logger.Info("received text command via HTTP", "text", text)
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"status":"received","text":"%s"}`, text)
	default:
		http.Error(w, "queue full, try again", http.StatusServiceUnavailable)
	}
}

type alexaRequest struct {
	Version string `json:"version"`
	Request struct {
		Type   string `json:"type"`
		Intent struct {
			Name  string `json:"name"`
			Slots map[string]struct {
				Value string `json:"value"`
			} `json:"slots"`
		} `json:"intent"`
	} `json:"request"`
}

func alexaResponse(text string, endSession bool) []byte {
	resp := fmt.Sprintf(`{
		"version": "1.0",
		"response": {
			"outputSpeech": {
				"type": "PlainText",
				"text": %q
			},
			"shouldEndSession": %t
		}
	}`, text, endSession)
	return []byte(resp)
}

func (h *HTTPSource) handleAlexa(w http.ResponseWriter, r *http.Request) {
	if h.authToken != "" {
		token := r.Header.Get("X-Auth-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if token != h.authToken {
			h.logger.Warn("unauthorized alexa request", "remote_addr", r.RemoteAddr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(alexaResponse("No autorizado", true))
			return
		}
	}

	data, err := io.ReadAll(io.LimitReader(r.Body, 8192))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(alexaResponse("Error al leer la solicitud", true))
		return
	}
	defer r.Body.Close()

	var alexaReq alexaRequest
	if err := json.Unmarshal(data, &alexaReq); err != nil {
		h.logger.Error("parsing alexa request", "error", err, "body", string(data))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(alexaResponse("No pude entender la solicitud", true))
		return
	}

	h.logger.Info("received alexa request", "type", alexaReq.Request.Type, "intent", alexaReq.Request.Intent.Name)

	switch alexaReq.Request.Type {
	case "LaunchRequest":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(alexaResponse("Hola, decime qué querés hacer", false))
		return

	case "SessionEndedRequest":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(alexaResponse("Chau", true))
		return

	case "IntentRequest":
		// handled below

	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(alexaResponse("No entendí el pedido", true))
		return
	}

	intentName := alexaReq.Request.Intent.Name
	if intentName == "AMAZON.HelpIntent" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(alexaResponse("Podés decirme cosas como: encendé la luz de la cocina, o activá la escena película", false))
		return
	}
	if intentName == "AMAZON.StopIntent" || intentName == "AMAZON.CancelIntent" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(alexaResponse("Chau", true))
		return
	}

	commandSlot, ok := alexaReq.Request.Intent.Slots["command"]
	if !ok || commandSlot.Value == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(alexaResponse("No entendí el comando, probá de nuevo", false))
		return
	}

	text := commandSlot.Value
	marker := []byte(domain.TextCommandPrefix + text)

	select {
	case h.audioChan <- marker:
		h.logger.Info("received command from Alexa", "text", text)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(alexaResponse(fmt.Sprintf("Ejecutando: %s", text), true))
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(alexaResponse("Estoy ocupado, probá en un momento", true))
	}
}

func (h *HTTPSource) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	running := h.running
	queueSize := len(h.audioChan)
	h.mu.Unlock()

	status := "ok"
	statusCode := http.StatusOK

	if !running {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, `{"status":"%s","running":%t,"queue_size":%d}`, status, running, queueSize)
}

func IsTextCommand(data []byte) (string, bool) {
	if len(data) > len(domain.TextCommandPrefix) && string(data[:len(domain.TextCommandPrefix)]) == domain.TextCommandPrefix {
		return string(data[len(domain.TextCommandPrefix):]), true
	}
	return "", false
}
