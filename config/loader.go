package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"sync"
)

var (
	instance Config
	once     sync.Once
)

// Load loads configuration from files
func Load(configPaths ...string) (Config, error) {
	var err error
	once.Do(func() {
		cfg := &config{}

		// Load configurations from files such YAML, ENV
		for _, configPath := range configPaths {
			if err = cleanenv.ReadConfig(configPath, cfg); err != nil {
				err = fmt.Errorf("failed to read config file %s: %w", configPath, err)
				return
			}
		}

		// Load secrets from environment variables
		if err = cleanenv.ReadEnv(cfg); err != nil {
			err = fmt.Errorf("failed to read environment variables: %w", err)
			return
		}

		instance = cfg
	})

	if err != nil {
		return nil, err
	}

	return instance, nil
}

func MustLoad(configPath string) Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	return cfg
}

func Reset() {
	instance = nil
	once = sync.Once{}
}

func MustGet() Config {
	if instance == nil {
		panic("config not loaded, call Load() first")
	}
	return instance
}
