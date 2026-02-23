package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/goccy/go-yaml"
)

// OutputKind mirrors the TypeScript OutputKindSchema.
type OutputKind string

const (
	KindAudio OutputKind = "audio"
	KindVideo OutputKind = "video"
	KindAuto  OutputKind = "auto"
)

func (k OutputKind) IsValid() bool {
	switch k {
	case KindAudio, KindVideo, KindAuto:
		return true
	}
	return false
}

// AudioQuality is either a VBR integer (0–10, as string) or a CBR bitrate like "192K".
type AudioQuality string

func (q AudioQuality) IsValid() bool {
	if q == "" {
		return false
	}
	// CBR bitrate: e.g. "128K", "320K"
	if matched, _ := regexp.MatchString(`(?i)^[0-9]+K$`, string(q)); matched {
		return true
	}
	// VBR integer 0–10
	for _, c := range q {
		if c < '0' || c > '9' {
			return false
		}
	}
	n := 0
	fmt.Sscanf(string(q), "%d", &n)
	return n >= 0 && n <= 10
}

// Config is the global configuration, mirroring the TypeScript ConfigSchema.
type Config struct {
	Kind         OutputKind   `yaml:"kind"`
	Format       string       `yaml:"format"`
	AudioQuality AudioQuality `yaml:"audio_quality"`
	VideoQuality string       `yaml:"video_quality"`
	OutputFolder string       `yaml:"output_folder"`
}

// EntryConfig is a per-entry override of Config (all fields optional).
type EntryConfig struct {
	Kind         *OutputKind   `yaml:"kind,omitempty"`
	Format       *string       `yaml:"format,omitempty"`
	AudioQuality *AudioQuality `yaml:"audio_quality,omitempty"`
	VideoQuality *string       `yaml:"video_quality,omitempty"`
	OutputFolder *string       `yaml:"output_folder,omitempty"`
}

func defaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	return Config{
		Kind:         KindAuto,
		Format:       "mp3",
		AudioQuality: "5",
		VideoQuality: "best",
		OutputFolder: filepath.Join(homeDir, "Downloads", "mldy"),
	}
}

func loadConfig() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return defaultConfig(), nil
	}

	configPath := filepath.Join(homeDir, ".config", "mldy", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		cfg := defaultConfig()
		if saveErr := saveConfig(cfg); saveErr != nil {
			return Config{}, saveErr
		}
		return cfg, nil
	}

	cfg := defaultConfig() // start from defaults so missing keys are filled in
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig(), nil
	}

	// Validate and fall back to defaults on bad values.
	if !cfg.Kind.IsValid() {
		cfg.Kind = KindAuto
	}
	if !cfg.AudioQuality.IsValid() {
		cfg.AudioQuality = "5"
	}

	return cfg, nil
}

func saveConfig(cfg Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".config", "mldy")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func (c Config) MergeWith(entry EntryConfig) Config {
	merged := c
	if entry.Kind != nil {
		merged.Kind = *entry.Kind
	}
	if entry.Format != nil {
		merged.Format = *entry.Format
	}
	if entry.AudioQuality != nil {
		merged.AudioQuality = *entry.AudioQuality
	}
	if entry.VideoQuality != nil {
		merged.VideoQuality = *entry.VideoQuality
	}
	if entry.OutputFolder != nil {
		merged.OutputFolder = *entry.OutputFolder
	}
	return merged
}
