package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"telemetry-system/internal/config"
	"telemetry-system/internal/domain"
	"telemetry-system/internal/infrastructure/http"
	"telemetry-system/internal/infrastructure/influx"
	"telemetry-system/internal/infrastructure/mqtt"
	"telemetry-system/internal/logger"
	"telemetry-system/internal/service/alerts"
	"telemetry-system/internal/service/ingestor"
	"telemetry-system/internal/service/storage"
)

func main() {
	// 1. Загрузка конфигурации
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Инициализация логгера
	appLogger, err := logger.New(cfg.Logger.Level, cfg.Logger.Format)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer appLogger.Sync()

	appLogger.Info("Starting telemetry system backend...")

	// 3. SQLite для метаданных
	db, err := gorm.Open(sqlite.Open("metadata.db"), &gorm.Config{})
	if err != nil {
		appLogger.Error("Failed to connect to SQLite", zap.Error(err))
		os.Exit(1)
	}

	if err := db.AutoMigrate(&domain.User{}, &domain.Gateway{}, &domain.Sensor{}, &domain.Alert{}); err != nil {
		appLogger.Error("Failed to migrate database", zap.Error(err))
		os.Exit(1)
	}
	appLogger.Info("SQLite metadata database initialized")

	// 4. InfluxDB для телеметрии
	influxClient, err := influx.NewInfluxClient(&cfg.InfluxDB, appLogger.Logger)
	if err != nil {
		appLogger.Error("Failed to initialize InfluxDB client", zap.Error(err))
		os.Exit(1)
	}
	defer influxClient.Close()

	appLogger.Info("InfluxDB client connected")

	// 5. Сервисы
	storageService := storage.NewStorageService(influxClient, appLogger.Logger)
	alertService := alerts.NewAlertService(db, appLogger.Logger)
	ingestorService := ingestor.NewIngestor(storageService, alertService, appLogger.Logger)

	// 6. MQTT клиент
	mqttClient, err := mqtt.NewMQTTClient(&cfg.MQTT, appLogger.Logger)
	if err != nil {
		appLogger.Error("Failed to initialize MQTT client", zap.Error(err))
		os.Exit(1)
	}

	// 7. HTTP сервер
	httpServer := http.NewServer(
		cfg.Server.Host,
		cfg.Server.Port,
		db,
		influxClient,
		ingestorService,
		cfg.JWT.Secret,
		appLogger.Logger,
	)

	// 8. Контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	
	mqttClient.SetMessageHandler(func(data *domain.Measurement) {
		// Обработка происходит внутри client.go (отправка в канал)
	})


	if err := mqttClient.Connect(); err != nil {
		appLogger.Error("Failed to connect to MQTT broker", zap.Error(err))
		os.Exit(1)
	}

	appLogger.Info("MQTT broker connected", zap.String("broker", cfg.MQTT.Broker))

	// 9. Запускаем ingestor
	go ingestorService.Process(ctx, mqttClient.GetMessageChannel())

	// 10. Запускаем HTTP сервер
	go func() {
		appLogger.Info("HTTP API server starting",
			zap.String("address", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)))
		if err := httpServer.Start(); err != nil {
			appLogger.Error("HTTP server failed", zap.Error(err))
		}
	}()

	appLogger.Info("Telemetry system backend started successfully")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down telemetry system backend...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("HTTP server shutdown error", zap.Error(err))
	}

	mqttClient.Disconnect()
	storageService.Flush()

	appLogger.Info("Telemetry system backend stopped")
}