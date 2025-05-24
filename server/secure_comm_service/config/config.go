package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type JWTConfig struct {
	PublicKeyPath string `env:"JWT_PUBLIC_KEY_PATH" env-required:"true"`
}

type MinIoConfig struct {
	Port         string        `env:"MINIO_PORT" env-required:"true"`
	RootUser     string        `env:"MINIO_ROOT_USER" env-required:"true"`
	RootPassword string        `env:"MINIO_ROOT_PASSWORD" env-required:"true"`
	UseSSL       bool          `env:"MINIO_USE_SSL" env-required:"true"`
	UrlTTL       time.Duration `env:"MINIO_URL_LIFETIME" env-required:"true"`
}

type RedisConfig struct {
	HandshakeNoncesTTL time.Duration `env:"REDIS_HANDSHAKE_NONCES_TTL" env-required:"true"`
	SessionNoncesTTL   time.Duration `env:"REDIS_SESSION_NONCES_TTL" env-required:"true"`
	SessionKeyTTL      time.Duration `env:"REDIS_SESSION_KEY_TTL" env-required:"true"`
	ClientPubKeysTTL   time.Duration `env:"REDIS_CLIENT_PUB_KEYS_TTL" env-required:"true"`
	MinioUrlTTL        time.Duration `env:"REDIS_MINIO_URL_TTL" env-required:"true"`
	ServerAddr         string        `env:"REDIS_SERVER_ADDRESS" env-required:"true"`
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
	RPC   float64       `env:"HANDSHAKE_LIMITER_RPC" env-required:"true"`
	Burst int           `env:"HANDSHAKE_LIMITER_BURST" env-required:"true"`
	TTL   time.Duration `env:"HANDSHAKE_LIMITER_EXP_TTL" env-required:"true"`
}

type SessionLimiter struct {
	RPC   float64       `env:"SESSION_LIMITER_RPC" env-required:"true"`
	Burst int           `env:"SESSION_LIMITER_BURST" env-required:"true"`
	TTL   time.Duration `env:"SESSION_LIMITER_EXP_TTL" env-required:"true"`
}

type Config struct {
	JWT        JWTConfig
	Minio      MinIoConfig
	Redis      RedisConfig
	ServKeys   ServKeysConfig
	HTTPServ   HTTPServConfig
	HSLimiter  HandShakeLimiter
	SesLimiter SessionLimiter
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
