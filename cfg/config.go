package cfg

import (
	"errors"
	"os"
	"strconv"

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
	RedisConfig          RedisConfig
	AirAsiaClientConfig  AirAsiaClientConfig
	BatikAirClientConfig BatikAirClientConfig
	GarudaClientConfig   GarudaIndonesiaClientConfig
	LionAirClientConfig  LionAirClientConfig
	CacheTTLMinutes      int
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
	batikAirClientBaseUrl := mustEnv("BATIKAIR_CLIENT_BASE_URL", &errs)
	garudaClientBaseUrl := mustEnv("GARUDA_CLIENT_BASE_URL", &errs)
	lionAirClientBaseUrl := mustEnv("LIONAIR_CLIENT_BASE_URL", &errs)

	cacheTTLMinutes := mustEnv("CACHE_TTL_MINUTES", &errs)
	cacheTTLMinutesInt, err := strconv.Atoi(cacheTTLMinutes)

	if err != nil {
		errs = append(errs, errors.New("conversion failed env: "+"CACHE_TTL_MINUTES"))
	}

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
		BatikAirClientConfig: BatikAirClientConfig{
			BaseURL: batikAirClientBaseUrl,
		},
		GarudaClientConfig: GarudaIndonesiaClientConfig{
			BaseURL: garudaClientBaseUrl,
		},
		LionAirClientConfig: LionAirClientConfig{
			BaseURL: lionAirClientBaseUrl,
		},
		CacheTTLMinutes: cacheTTLMinutesInt,
	}, nil
}

func mustEnv(key string, errs *[]error) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		*errs = append(*errs, errors.New("missing env: "+key))
	}
	return value
}
