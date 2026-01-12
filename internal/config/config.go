package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	LLM      LLMConfig
	Worker   WorkerConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	URL             string
}

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	URL      string
	Host     string
	Port     int
	Password string
	DB       int
}

// LLMConfig holds LLM provider configuration.
type LLMConfig struct {
	OpenAIAPIKey        string
	AnthropicAPIKey     string
	OllamaBaseURL       string
	OllamaModel         string
	OpenRouterAPIKey    string
	OpenRouterModel     string
	OpenRouterReasoning bool
	DefaultProvider     string // "openai", "anthropic", "ollama", or "openrouter"
	Timeout             time.Duration
}

// WorkerConfig holds worker configuration.
type WorkerConfig struct {
	Concurrency   int
	BatchSize     int
	StreamName    string
	ConsumerGroup string
	ConsumerName  string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvAsInt("SERVER_PORT", getEnvAsInt("PORT", 8080)),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Database:        getEnv("DB_NAME", "healing_eval"),
			MaxConns:        getEnvAsInt("DB_MAX_CONNS", 25),
			MinConns:        getEnvAsInt("DB_MIN_CONNS", 5),
			MaxConnLifetime: getEnvAsDuration("DB_MAX_CONN_LIFETIME", time.Hour),
		},
		Redis: loadRedisConfig(),
		LLM: LLMConfig{
			OpenAIAPIKey:        getEnv("OPENAI_API_KEY", ""),
			AnthropicAPIKey:     getEnv("ANTHROPIC_API_KEY", ""),
			OllamaBaseURL:       getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
			OllamaModel:         getEnv("OLLAMA_MODEL", "llama3.1:8b"),
			OpenRouterAPIKey:    getEnv("OPENROUTER_API_KEY", ""),
			OpenRouterModel:     getEnv("OPENROUTER_MODEL", "nvidia/nemotron-3-nano-30b-a3b:free"),
			OpenRouterReasoning: getEnvAsBool("OPENROUTER_ENABLE_REASONING", false),
			DefaultProvider:     getEnv("LLM_DEFAULT_PROVIDER", "ollama"),
			Timeout:             getEnvAsDuration("LLM_TIMEOUT", 120*time.Second),
		},
		Worker: WorkerConfig{
			Concurrency:   getEnvAsInt("WORKER_CONCURRENCY", 10),
			BatchSize:     getEnvAsInt("WORKER_BATCH_SIZE", 10),
			StreamName:    getEnv("WORKER_STREAM_NAME", "conversations"),
			ConsumerGroup: getEnv("WORKER_CONSUMER_GROUP", "eval-workers"),
			ConsumerName:  getEnv("WORKER_CONSUMER_NAME", "worker-1"),
		},
	}

	return cfg, nil
}

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	if c.URL != "" {
		return c.URL
	}
	return "postgres://" + c.User + ":" + c.Password + "@" + c.Host + ":" + strconv.Itoa(c.Port) + "/" + c.Database + "?sslmode=disable"
}

// Addr returns the Redis address.
func (c *RedisConfig) Addr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func (c *RedisConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("Redis host is empty. Set REDIS_URL or REDIS_HOST environment variable")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid Redis port: %d", c.Port)
	}
	return nil
}

func loadRedisConfig() RedisConfig {
	redisURL := getEnv("REDIS_URL", "")
	if redisURL != "" {
		return parseRedisURL(redisURL)
	}

	return RedisConfig{
		Host:     getEnv("REDISHOST", getEnv("REDIS_HOST", "")),
		Port:     getEnvAsInt("REDISPORT", getEnvAsInt("REDIS_PORT", 6379)),
		Password: getEnv("REDISPASSWORD", getEnv("REDIS_PASSWORD", "")),
		DB:       getEnvAsInt("REDIS_DB", 0),
	}
}

func parseRedisURL(redisURL string) RedisConfig {
	cfg := RedisConfig{
		URL:  redisURL,
		Port: 6379,
		DB:   0,
	}

	if !strings.HasPrefix(redisURL, "redis://") && !strings.HasPrefix(redisURL, "rediss://") {
		redisURL = "redis://" + redisURL
		cfg.URL = redisURL
	}

	u, err := url.Parse(redisURL)
	if err != nil {
		return cfg
	}

	if u.User != nil {
		cfg.Password, _ = u.User.Password()
	}

	cfg.Host = u.Hostname()
	if u.Port() != "" {
		if port, err := strconv.Atoi(u.Port()); err == nil {
			cfg.Port = port
		}
	}

	if u.Path != "" {
		dbStr := strings.TrimPrefix(u.Path, "/")
		if dbStr != "" {
			if db, err := strconv.Atoi(dbStr); err == nil {
				cfg.DB = db
			}
		}
	}

	return cfg
}

// Addr returns the server address.
func (c *ServerConfig) Addr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
