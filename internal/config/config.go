package config

import (
	"log"
	"sync"

	"github.com/joho/godotenv"

	"github.com/caarlos0/env/v11"
)

var (
	once sync.Once
	cfg  Config
)

type (
	Config struct {
		App                AppConfig   `envPrefix:"APP_"`
		DB                 DB          `envPrefix:"DB_"`
		JWT                JWTConfig   `envPrefix:"JWT_"`
		MODEL_URL          string      `env:"MODEL_URL"`
		CLASSIFICATION_URL string      `env:"CLASSIFICATION_URL" envDefault:"http://localhost:8000/classification/classify_image"`
		MinIO              MinIOConfig `envPrefix:"MINIO_"`
	}

	AppConfig struct {
		Port string `env:"PORT" envDefault:"8000"`
	}

	DB struct {
		Host     string `env:"HOST"`
		Port     string `env:"PORT"`
		User     string `env:"USER"`
		Password string `env:"PASSWORD"`
		Name     string `env:"NAME"`
		URL      string `env:"-"`
	}

	JWTConfig struct {
		Secret string `env:"SECRET"`
	}

	MinIOConfig struct {
		Endpoint       string `env:"ENDPOINT"`
		PublicEndpoint string `env:"PUBLIC_ENDPOINT"`
		AccessKey      string `env:"ACCESS_KEY"`
		SecretKey      string `env:"SECRET_KEY"`
		Bucket         string `env:"BUCKET"`
		UseSSL         bool   `env:"USE_SSL"`
	}
)

// Inst returns thse singleton instance of the Config object.
// It loads the configuration from environment variables, using a .env file if present.
func Inst() *Config {
	once.Do(func() {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file", err)
		}
		err = env.Parse(&cfg)
		if err != nil {
			log.Fatalf("Failed to parse environment variables: %v", err)
		}
		cfg.DB.URL = "postgres://" + cfg.DB.User + ":" + cfg.DB.Password + "@" + cfg.DB.Host + ":" + cfg.DB.Port + "/" + cfg.DB.Name + "?sslmode=disable"
	})
	return &cfg
}
