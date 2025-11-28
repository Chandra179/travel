package main

import (
	"log"
	"net/http"
	"time"
	"travel/cfg"
	"travel/internal/flight"
	"travel/pkg/cache"
	"travel/pkg/flightclient"
	"travel/pkg/logger"

	_ "travel/api" // swagger docs

	"github.com/gin-gonic/gin"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	// ============
	// config
	// ============
	config, errCfg := cfg.Load()
	if errCfg != nil {
		log.Fatal(errCfg)
	}

	// ============
	// logger
	// ============
	zlogger := logger.NewZeroLog(config.AppEnv)

	// ============
	// Cache
	// ============
	redisAddr := config.RedisConfig.Host + ":" + config.RedisConfig.Port
	_ = cache.NewRedisCache(redisAddr)

	// ============
	// External Service
	// ============
	httpClient := &http.Client{ // httpClient can be reused or seperate client per external service=
		Timeout: 15 * time.Second,
	}
	airAsiaClient := flightclient.NewAirAsiaClient(httpClient,
		config.AirAsiaClientConfig.BaseURL, zlogger)
	flightClient := flightclient.NewFlightClient(airAsiaClient, zlogger)

	// ============
	// Inernal Service
	// ============
	flightSvc := flight.NewFlightService(flightClient)
	flightHandler := flight.NewFlightHandler(flightSvc)

	// ============
	// HTTP
	// ============
	r := gin.Default()

	flightHandler.RegisterRoutes(r)
	initSwagger(r)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initSwagger(r *gin.Engine) {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/docs", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
    <script id="api-reference" data-url="/swagger/doc.json"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`
		c.String(200, html)
	})
}
