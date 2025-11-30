package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config captures all runtime configuration for the service.
type Config struct {
	HTTP      HTTPConfig
	Postgres  PostgresConfig
	Redis     RedisConfig
	Scheduler SchedulerConfig
	Webhook   WebhookConfig
	Server    ServerConfig
}

// HTTPConfig holds HTTP server related configuration.
type HTTPConfig struct {
	Port string
}

// PostgresConfig holds database connection settings.
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DSN returns the formatted connection string for pgx.
func (p PostgresConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", p.Host, p.Port, p.User, p.Password, p.DBName, p.SSLMode)
}

// RedisConfig holds redis connection settings.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// SchedulerConfig holds scheduling settings.
type SchedulerConfig struct {
	Interval   time.Duration
	FetchLimit int
}

// WebhookConfig stores outbound webhook details.
type WebhookConfig struct {
	URL     string
	AuthKey string
}

// ServerConfig stores general server runtime configuration.
type ServerConfig struct {
	ShutdownTimeout time.Duration
}

// Load builds configuration by reading environment variables with sane defaults.
func Load() (*Config, error) {
	pgPort, err := getInt("POSTGRES_PORT", 5432)
	if err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_PORT: %w", err)
	}

	redisDB, err := getInt("REDIS_DB", 0)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	fetchLimit, err := getInt("SCHEDULER_FETCH_LIMIT", 2)
	if err != nil {
		return nil, fmt.Errorf("invalid SCHEDULER_FETCH_LIMIT: %w", err)
	}

	intervalStr := getString("SCHEDULER_INTERVAL", "2m")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SCHEDULER_INTERVAL: %w", err)
	}
	if interval < 2*time.Minute {
		interval = 2 * time.Minute
	}

	shutdownTimeoutStr := getString("SERVER_SHUTDOWN_TIMEOUT", "10s")
	shutdownTimeout, err := time.ParseDuration(shutdownTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_SHUTDOWN_TIMEOUT: %w", err)
	}

	cfg := &Config{
		HTTP: HTTPConfig{
			Port: getString("HTTP_PORT", "8083"),
		},
		Postgres: PostgresConfig{
			Host:     getString("POSTGRES_HOST", "postgres"),
			Port:     pgPort,
			User:     getString("POSTGRES_USER", "appuser"),
			Password: getString("POSTGRES_PASSWORD", "appsecret"),
			DBName:   getString("POSTGRES_DB", "automessaging"),
			SSLMode:  getString("POSTGRES_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     getString("REDIS_ADDR", "redis:6379"),
			Password: getString("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Scheduler: SchedulerConfig{
			Interval:   interval,
			FetchLimit: fetchLimit,
		},
		Webhook: WebhookConfig{
			URL:     getString("WEBHOOK_URL", ""),
			AuthKey: getString("WEBHOOK_AUTH_KEY", "INS.me1x9uMcyYGlhKKQVPoc.bO3j9aZwRTOcA2Ywo"),
		},
		Server: ServerConfig{
			ShutdownTimeout: shutdownTimeout,
		},
	}

	return cfg, nil
}

func getString(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func getInt(key string, def int) (int, error) {
	if val := os.Getenv(key); val != "" {
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	}
	return def, nil
}
