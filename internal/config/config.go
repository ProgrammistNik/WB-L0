package config

import (
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

// A Config represents all configuration of service
type Config struct {
	Server         ServerConfig         `yaml:"server"`
	Database       DatabaseConfig       `yaml:"database"`
	Kafka          KafkaConfig          `yaml:"kafka"`
	Cache          CacheConfig          `yaml:"cache"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
}

// A ServerConfig contains configurations for HTTP server
type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// A DatabaseConfig contains settings for Postgres
type DatabaseConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	User               string
	Password           string
	Database           string
	SSLMode            string        `yaml:"ssl_mode"`
	MaxOpenConnections int           `yaml:"max_open_connections"`
	MinOpenConnections int           `yaml:"min_open_connections"`
	MinIdleConnections int           `yaml:"min_idle_connections"`
	HealthCheckPeriod  time.Duration `yaml:"health_check_period"`
	Retry              RetryConfig   `yaml:"retry"`
}

// A KafkaConfig contains settings for Kafka
type KafkaConfig struct {
	Topic               string   `yaml:"topic"`
	GroupID             string   `yaml:"group_id"`
	Listeners           string   `yaml:"listeners"`
	AdvertisedListeners []string `yaml:"advertised_listeners"`
}

// A CacheConfig represents settings for cache
type CacheConfig struct {
	Capacity int `yaml:"capacity"`
}

// A RetryConfig represents retry configurations
type RetryConfig struct {
	MaxAttempts int `yaml:"max_attempts"`
}

// A CircuitBreakerConfig represents circuit breaker configurations
type CircuitBreakerConfig struct {
	MaxFailers       int           `yaml:"max_failers"`
	Timeout          time.Duration `yaml:"timeout"`
	HalfOpenMaxCalls int           `yaml:"half_open_max_calls"`
}

// LoadConfig loads data into Config structure from a file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal([]byte(data), &config); err != nil {
		return nil, err
	}
	config.loadEnv()
	return &config, nil
}

// loadEnv loads data into Config structure from the environmental variables
func (c *Config) loadEnv() {
	err := godotenv.Load("deployments/.env")
	if err != nil {
		return
	}
	// Database env variables
	c.Database.User = os.Getenv("POSTGRES_USER")
	c.Database.Password = os.Getenv("POSTGRES_PASSWORD")
	c.Database.Database = os.Getenv("POSTGRES_DB")

}

func (c *Config) GetServerAddress() string {
	return fmt.Sprintf(":%d", c.Server.Port)
}

// Validate checks if the most important fields are properly filled
func (c *Config) Validate() error {
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d: ", c.Database.Port)

	}
	if c.Database.Host == "" {
		return fmt.Errorf("database host is requested")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d: ", c.Server.Port)
	}
	if c.Cache.Capacity <= 0 {
		return errors.New("cache capacity must be positive")
	}

	return nil
}