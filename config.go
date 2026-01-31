package main

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Format       string `yaml:"format"`
	AudioQuality string `yaml:"audio_quality"`
	VideoQuality string `yaml:"video_quality"`
	OutputFolder string `yaml:"output_folder"`
}

type EntryConfig struct {
	Format       *string `yaml:"format,omitempty"`
	AudioQuality *string `yaml:"audio_quality,omitempty"`
	VideoQuality *string `yaml:"video_quality,omitempty"`
	OutputFolder *string `yaml:"output_folder,omitempty"`
}

func defaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	return Config{
		Format:       "mp3",
		AudioQuality: "best",
		VideoQuality: "best",
		OutputFolder: filepath.Join(homeDir, "Downloads", "mldy-cli"),
	}
}

func loadConfig() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return defaultConfig(), nil
	}

	configPath := filepath.Join(homeDir, ".config", "mldy-cli", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		// if config doesn't exist create it with defaults
		cfg := defaultConfig()
		err := saveConfig(cfg)
		if err != nil {
			return Config{}, err
		}
		return cfg, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig(), nil
	}

	return cfg, nil
}

func saveConfig(cfg Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".config", "mldy-cli")
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
