package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type RedisConfig struct {
	NoncesTTL     time.Duration `env:"REDIS_NONCES_TTL" env-required:"true"`
	SessionKeyTTL time.Duration `env:"REDIS_SESSION_KEY_TTL" env-required:"true"`
	ServerAddr    string        `env:"REDIS_SERVER_ADDRESS" env-required:"true"`
}

type ServKeysConfig struct {
	DirKeysPath   string `env:"KEY_DIR_PATH" env-required:"true"`
	RSAPubPath    string `env:"RSA_PUB_PATH" env-required:"true"`
	RSAPrivPath   string `env:"RSA_PRIV_PATH" env-required:"true"`
	ECDSAPubPath  string `env:"ECDSA_PUB_PATH" env-required:"true"`
	ECDSAPrivPath string `env:"ECDSA_PRIV_PATH" env-required:"true"`
}

type HTTPServConfig struct {
	ServerAddr string `env:"HTTP_SERVER_ADDRESS" env-required:"true"`
}

type HandShakeLimiter struct {
	RPC   float64       `env:"LIMITER_RPC" env-required:"true"`
	Burst int           `env:"LIMITER_BURST" env-required:"true"`
	TTL   time.Duration `env:"LIMITER_EXP_TTL" env-required:"true"`
}

type Config struct {
	Redis     RedisConfig
	ServKeys  ServKeysConfig
	HTTPServ  HTTPServConfig
	HSLimiter HandShakeLimiter
}

func MustLoad() *Config {
	path := getConfigPath()

	if path == "" {
		panic("config path is empty")
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
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	return res
}
