package cfg

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type RedisConfig struct {
	Host string
	Port string
}

type AirAsiaClientConfig struct {
	BaseURL string
}

type BatikAirClientConfig struct {
	BaseURL string
}

type GarudaIndonesiaClientConfig struct {
	BaseURL string
}

type LionAirClientConfig struct {
	BaseURL string
}

type Config struct {
	AppEnv               string
	AppPort              string
	RedisConfig          RedisConfig
	AirAsiaClientConfig  AirAsiaClientConfig
	BatikAirClientConfig BatikAirClientConfig
	GarudaClientConfig   GarudaIndonesiaClientConfig
	LionAirClientConfig  LionAirClientConfig
	CacheTTLSeconds      int
}

func Load() (*Config, error) {
	var errs []error

	// Ignore read .env if it not exist. (read from docker-compose)
	_ = godotenv.Load()

	appEnv := mustEnv("APP_ENV", &errs)
	appPort := mustEnv("APP_PORT", &errs)
	redisHost := mustEnv("REDIS_HOST", &errs)
	redistPort := mustEnv("REDIS_PORT", &errs)

	airAsiaClientBaseUrl := mustEnv("AIRASIA_CLIENT_BASE_URL", &errs)
	batikAirClientBaseUrl := mustEnv("BATIKAIR_CLIENT_BASE_URL", &errs)
	garudaClientBaseUrl := mustEnv("GARUDA_CLIENT_BASE_URL", &errs)
	lionAirClientBaseUrl := mustEnv("LIONAIR_CLIENT_BASE_URL", &errs)

	cacheTTLInSeconds := mustEnv("CACHE_TTL_SECONDS", &errs)
	cacheTTLSecondsInt, err := strconv.Atoi(cacheTTLInSeconds)

	if err != nil {
		errs = append(errs, errors.New("conversion failed env: "+"CACHE_TTL_SECONDS"))
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return &Config{
		AppEnv:  appEnv,
		AppPort: appPort,
		RedisConfig: RedisConfig{
			Host: redisHost,
			Port: redistPort,
		},
		AirAsiaClientConfig: AirAsiaClientConfig{
			BaseURL: airAsiaClientBaseUrl,
		},
		BatikAirClientConfig: BatikAirClientConfig{
			BaseURL: batikAirClientBaseUrl,
		},
		GarudaClientConfig: GarudaIndonesiaClientConfig{
			BaseURL: garudaClientBaseUrl,
		},
		LionAirClientConfig: LionAirClientConfig{
			BaseURL: lionAirClientBaseUrl,
		},
		CacheTTLSeconds: cacheTTLSecondsInt,
	}, nil
}

func mustEnv(key string, errs *[]error) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		*errs = append(*errs, errors.New("missing env: "+key))
	}
	return value
}
