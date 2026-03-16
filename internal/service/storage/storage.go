package storage

import (
	"go.uber.org/zap"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"telemetry-system/internal/domain"
	"telemetry-system/internal/infrastructure/influx"
)

type StorageService struct {
	influxClient *influx.InfluxClient
	logger       *zap.Logger
}

func NewStorageService(influxClient *influx.InfluxClient, logger *zap.Logger) *StorageService {
	return &StorageService{
		influxClient: influxClient,
		logger:       logger,
	}
}

func (s *StorageService) WriteMeasurement(m *domain.Measurement) error {
	point := influxdb2.NewPoint(
		m.InfluxMeasurementName(),
		m.ToInfluxTags(),
		m.ToInfluxFields(),
		m.Timestamp,
	)

	s.influxClient.WriteAPI().WritePoint(point)
	s.logger.Debug("Measurement written",
		zap.String("sensor_id", m.SensorID),
		zap.Float64("value", m.Value))

	return nil
}

func (s *StorageService) WriteAlertLog(log domain.AlertLog) error {
	point := influxdb2.NewPoint(
		log.InfluxMeasurementName(),
		log.ToInfluxTags(),
		log.ToInfluxFields(),
		log.Timestamp,
	)

	s.influxClient.WriteAPI().WritePoint(point)
	s.logger.Warn("Alert log written",
		zap.String("alert_id", log.AlertID),
		zap.Float64("value", log.Value))

	return nil
}

func (s *StorageService) Flush() {
	s.influxClient.WriteAPI().Flush()
}

func (s *StorageService) Close() {
	s.influxClient.Close()
}