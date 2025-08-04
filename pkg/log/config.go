package log

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	Level       string `yaml:"level"`
	Format      string `yaml:"format"`
	Environment string `yaml:"environment"`
	ServiceName string `yaml:"service_name"`
	Version     string `yaml:"version"`

	OutputPath string `yaml:"output_path"`

	FileMaxSizeInMB  int  `yaml:"file_max_size_mb"`
	FileMaxAgeInDays int  `yaml:"file_max_age_days"`
	FileMaxBackups   int  `yaml:"file_max_backups"`
	CompressRotated  bool `yaml:"compress_rotated"`

	DisableCaller     bool            `yaml:"disable_caller"`
	DisableStacktrace bool            `yaml:"disable_stacktrace"`
	SamplingConfig    *SamplingConfig `yaml:"sampling"`

	InitialFields map[string]interface{} `yaml:"initial_fields"`
}

type SamplingConfig struct {
	Initial    int           `yaml:"initial"`
	Thereafter int           `yaml:"thereafter"`
	Tick       time.Duration `yaml:"tick"`
}

func (c *Config) Validate() error {
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLevels[strings.ToLower(c.Level)] {
		return fmt.Errorf("invalid log level: %s, must be one of: debug, info, warn, error, fatal", c.Level)
	}

	validFormats := map[string]bool{"json": true, "console": true}
	if !validFormats[strings.ToLower(c.Format)] {
		return fmt.Errorf("invalid log format: %s, must be 'json' or 'console'", c.Format)
	}

	if c.FileMaxSizeInMB <= 0 {
		return fmt.Errorf("file_max_size_mb must be greater than 0")
	}
	if c.FileMaxAgeInDays <= 0 {
		return fmt.Errorf("file_max_age_days must be greater than 0")
	}
	if c.FileMaxBackups < 0 {
		return fmt.Errorf("file_max_backups must be greater than or equal to 0")
	}

	if c.SamplingConfig != nil {
		if c.SamplingConfig.Initial <= 0 {
			return fmt.Errorf("sampling initial must be greater than 0")
		}
		if c.SamplingConfig.Thereafter <= 0 {
			return fmt.Errorf("sampling thereafter must be greater than 0")
		}
	}

	return nil
}

func DefaultConfig() Config {
	return Config{
		Level:             "info",
		Format:            "json",
		Environment:       "development",
		ServiceName:       "go-clean-arch",
		Version:           "1.0.0",
		OutputPath:        "stdout",
		FileMaxSizeInMB:   100,
		FileMaxAgeInDays:  30,
		FileMaxBackups:    10,
		CompressRotated:   true,
		DisableCaller:     false,
		DisableStacktrace: false,
		InitialFields:     make(map[string]interface{}),
	}
}

func DevelopmentConfig() Config {
	config := DefaultConfig()
	config.Level = "debug"
	config.Format = "console"
	config.Environment = "development"
	config.DisableCaller = false
	config.DisableStacktrace = false
	return config
}

func ProductionConfig(serviceName, version string) Config {
	config := DefaultConfig()
	config.Level = "info"
	config.Format = "json"
	config.Environment = "production"
	config.ServiceName = serviceName
	config.Version = version
	config.DisableCaller = true
	config.DisableStacktrace = true

	config.SamplingConfig = &SamplingConfig{
		Initial:    100,
		Thereafter: 100,
	}

	return config
}
