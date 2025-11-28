package cfg

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

type AirAsiaClientConfig struct {
	BaseURL string
}

type Config struct {
	AppEnv              string
	RedisConfig         RedisConfig
	AirAsiaClientConfig AirAsiaClientConfig
}

func Load() (*Config, error) {
	var errs []error

	err := godotenv.Load()
	if err != nil {
		return nil, errors.New("failed load cfg: " + err.Error())
	}

	appEnv := mustEnv("APP_ENV", &errs)
	redisHost := mustEnv("REDIS_HOST", &errs)
	redistPort := mustEnv("REDIS_PORT", &errs)
	redisPassword := mustEnv("REDIS_PASSWORD", &errs)

	airAsiaClientBaseUrl := mustEnv("AIRASIA_CLIENT_BASE_URL", &errs)

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return &Config{
		AppEnv: appEnv,
		RedisConfig: RedisConfig{
			Host:     redisHost,
			Port:     redistPort,
			Password: redisPassword,
		},
		AirAsiaClientConfig: AirAsiaClientConfig{
			BaseURL: airAsiaClientBaseUrl,
		},
	}, nil
}

// mustEnv appends error into slice instead of returning.
func mustEnv(key string, errs *[]error) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		*errs = append(*errs, errors.New("missing env: "+key))
	}
	return value
}
