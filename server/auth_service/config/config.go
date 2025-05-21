package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	StoragePath     string        `env:"STORAGE_PATH" env-required:"true"`
	HTTPServer      string        `env:"HTTP_SERVER_ADDRESS" env-required:"true"`
	AccessTokenTTL  time.Duration `env:"ACCESS_TOKEN_TTL" env-required:"true"`
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" env-required:"true"`
	PublicKeyPath   string        `env:"PUBLIC_KEY_PATH" env-required:"true"`
	PrivateKeyPath  string        `env:"PRIVATE_KEY_PATH" env-required:"true"`
	QuotaServiceURL string        `env:"QUOTA_SERVICE_URL" env-required:"true"`
}

func MustLoad() *Config {
	path := getConfigPath()

	if path == "" {
		fmt.Println("CONFIG_PATH is not set, using default .env")
		path = ".env"
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file does not exists" + path)
	}

	err := godotenv.Load(path)
	if err != nil {
		panic(fmt.Sprintf("No .env file found at %s, relying on environment variables", path))
	}

	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic(fmt.Sprintf("Failed to load environment variables: %v", err))
	}

	return &cfg
}

func getConfigPath() string {
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		return envPath
	}

	var res string
	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	return res
}
