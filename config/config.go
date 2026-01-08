package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Audio     AudioConfig     `yaml:"audio"`
	OpenAI    OpenAIConfig    `yaml:"openai"`
	Anthropic AnthropicConfig `yaml:"anthropic"`
	Tuya      TuyaConfig      `yaml:"tuya"`
	Pushover  PushoverConfig  `yaml:"pushover"`
	Log       LogConfig       `yaml:"log"`
}

type AudioConfig struct {
	Source     string `yaml:"source"`
	HTTPAddr   string `yaml:"http_addr"`
	FileDir    string `yaml:"file_dir"`
	WakeWord   string `yaml:"wake_word"`
	SampleRate int    `yaml:"sample_rate"`
	AuthToken  string `yaml:"auth_token"`
}

type OpenAIConfig struct {
	APIKey   string `yaml:"api_key"`
	Language string `yaml:"language"`
}

type AnthropicConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type TuyaConfig struct {
	ClientID     string `yaml:"client_id"`
	Secret       string `yaml:"secret"`
	Region       string `yaml:"region"`
	SyncInterval string `yaml:"sync_interval"`
}

type PushoverConfig struct {
	Token   string `yaml:"token"`
	UserKey string `yaml:"user_key"`
	Enabled bool   `yaml:"enabled"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.setDefaults()

	return &cfg, nil
}

func (c *Config) setDefaults() {
	if c.Audio.Source == "" {
		c.Audio.Source = "http"
	}
	if c.Audio.HTTPAddr == "" {
		c.Audio.HTTPAddr = ":8080"
	}
	if c.Audio.FileDir == "" {
		c.Audio.FileDir = "./audio"
	}
	if c.Audio.SampleRate == 0 {
		c.Audio.SampleRate = 16000
	}
	if c.OpenAI.Language == "" {
		c.OpenAI.Language = "es"
	}
	if c.Anthropic.Model == "" {
		c.Anthropic.Model = "claude-sonnet-4-20250514"
	}
	if c.Tuya.Region == "" {
		c.Tuya.Region = "us"
	}
	if c.Tuya.SyncInterval == "" {
		c.Tuya.SyncInterval = "5m"
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "text"
	}
}

