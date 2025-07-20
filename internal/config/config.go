package config

import (
	"fmt"
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" env-default:"local"`
	Log         LogConfig
	JWT         JWTConfig
	Server      ServerConfig
	Postgres    PostgresConfig
}

type LogConfig struct {
	Level   int    `env:"LOG_LEVEL" env-default:"0"`
	Handler string `env:"LOG_HANDLER" env-default:"text"`
}

type JWTConfig struct {
	JWTSecret     string        `env:"JWT_SECRET" env-required:"true"`
	TokenLiveTime time.Duration `env:"TOKEN_LIVE_TIME" env-default:"24h"`
	BCryptCost    int           `env:"BCRYPT_COST" env-default:"12"`
}

type ServerConfig struct {
	ServerPort        string        `env:"SERVER_PORT" env-default:"8080"`
	MetricsPort       string        `env:"METRICS_PORT" env-default:"9000"`
	ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout      time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" env-default:"5s"`
}

type PostgresConfig struct {
	Host            string        `env:"POSTGRES_HOST" env-required:"true"`
	Port            string        `env:"POSTGRES_PORT" env-required:"true"`
	User            string        `env:"POSTGRES_USER" env-required:"true"`
	Password        string        `env:"POSTGRES_PASSWORD" env-required:"true"`
	Name            string        `env:"POSTGRES_DB" env-required:"true"`
	SSLMode         string        `env:"POSTGRES_SSL_MODE" env-default:"disable"`
	MaxOpenConns    int           `env:"POSTGRES_MAX_OPEN_CONNS" env-default:"10"`
	MaxIdleConns    int           `env:"POSTGRES_MAX_IDLE_CONNS" env-default:"5"`
	ConnMaxLifetime time.Duration `env:"POSTGRES_CONN_MAX_LIFETIME" env-default:"1h"`
	Url             string
}

func MustLoad() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	var cfg Config

	err = cleanenv.ReadEnv(&cfg)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cfg.Postgres.Url = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.Name,
		cfg.Postgres.SSLMode,
	)

	return &cfg
}

type JWTConfigForTest struct {
	JWTSecret     string
	BCryptCost    int
	TokenLiveTime time.Duration
}

type ConfigForTest struct {
	JWT JWTConfigForTest
}

func NewConfigForTest() *ConfigForTest {
	return &ConfigForTest{
		JWT: JWTConfigForTest{
			JWTSecret:     "test-secret",
			BCryptCost:    10,
			TokenLiveTime: 24 * time.Hour,
		},
	}
}
