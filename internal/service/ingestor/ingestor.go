package ingestor

import (
	"context"
	"go.uber.org/zap"
	"telemetry-system/internal/domain"
	"telemetry-system/internal/service/alerts"
	"telemetry-system/internal/service/storage"
	"time"
)

type Ingestor struct {
	storage  *storage.StorageService
	alertSvc *alerts.AlertService
	logger   *zap.Logger
	stats    IngestorStats
}

type IngestorStats struct {
	TotalReceived   int64
	TotalValid      int64
	TotalInvalid    int64
	AlertsTriggered int64
	LastUpdateTime  time.Time
}

func NewIngestor(storage *storage.StorageService, alertSvc *alerts.AlertService, logger *zap.Logger) *Ingestor {
	return &Ingestor{
		storage:  storage,
		alertSvc: alertSvc,
		logger:   logger,
		stats:    IngestorStats{},
	}
}

func (i *Ingestor) Process(ctx context.Context, messageCh <-chan *domain.Measurement) {
	i.logger.Info("Ingestor started processing messages")

	for {
		select {
		case <-ctx.Done():
			i.logger.Info("Ingestor stopped gracefully")
			return
		case data := <-messageCh:
			i.stats.TotalReceived++
			i.stats.LastUpdateTime = time.Now()

			if !i.validateMeasurement(data) {
				i.stats.TotalInvalid++
				i.logger.Warn("Measurement validation failed",
					zap.String("sensor_id", data.SensorID))
				continue
			}

			if err := i.storage.WriteMeasurement(data); err != nil {
				i.logger.Error("Failed to write measurement",
					zap.String("sensor_id", data.SensorID),
					zap.Error(err))
				continue
			}

			i.stats.TotalValid++
			i.checkAlerts(data)
		}
	}
}

func (i *Ingestor) validateMeasurement(m *domain.Measurement) bool {
	if m.SensorID == "" || m.Timestamp.IsZero() {
		return false
	}
	if m.Value != m.Value {
		return false
	}
	return true
}

func (i *Ingestor) checkAlerts(m *domain.Measurement) {
	rules := i.alertSvc.GetActiveRulesForSensor(m.SensorID)

	for _, rule := range rules {
		violated := false
		if rule.Condition == "gt" && m.Value > rule.Threshold {
			violated = true
		} else if rule.Condition == "lt" && m.Value < rule.Threshold {
			violated = true
		}

		if violated {
			logEntry := domain.AlertLog{
				AlertID:      rule.ID,
				SensorID:     m.SensorID,
				GatewayID:    m.GatewayID,
				Timestamp:    time.Now(),
				Value:        m.Value,
				Acknowledged: false,
				Message:      "Threshold exceeded",
			}

			if err := i.storage.WriteAlertLog(logEntry); err != nil {
				i.logger.Error("Failed to write alert log", zap.Error(err))
				continue
			}

			i.stats.AlertsTriggered++
			i.logger.Warn("Alert triggered",
				zap.String("alert_id", rule.ID),
				zap.Float64("value", m.Value))
		}
	}
}

func (i *Ingestor) GetStats() IngestorStats {
	return i.stats
}