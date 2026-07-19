package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

const (
	defaultHost       = "0.0.0.0"
	defaultPort       = 8080
	defaultConfigPath = "/config"
	defaultCachePath  = "/cache"
	defaultMediaPath  = "/media"
	defaultWebPath    = "/app/web"
)

type Config struct {
	Host       string
	Port       int
	ConfigPath string
	CachePath  string
	MediaPath  string
	WebPath    string
}

func FromEnv() (Config, error) {
	port, err := intFromEnv("FLEX_PORT", defaultPort)
	if err != nil {
		return Config{}, err
	}
	if port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("FLEX_PORT must be between 1 and 65535")
	}

	return Config{
		Host:       stringFromEnv("FLEX_HOST", defaultHost),
		Port:       port,
		ConfigPath: stringFromEnv("FLEX_CONFIG_DIR", defaultConfigPath),
		CachePath:  stringFromEnv("FLEX_CACHE_DIR", defaultCachePath),
		MediaPath:  stringFromEnv("FLEX_MEDIA_DIR", defaultMediaPath),
		WebPath:    stringFromEnv("FLEX_WEB_DIR", defaultWebPath),
	}, nil
}

func (config Config) Address() string {
	return net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
}

func stringFromEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intFromEnv(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}
