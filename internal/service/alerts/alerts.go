package alerts

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
	"telemetry-system/internal/domain"
)

type AlertService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewAlertService(db *gorm.DB, logger *zap.Logger) *AlertService {
	return &AlertService{
		db:     db,
		logger: logger,
	}
}

func (s *AlertService) GetActiveRulesForSensor(sensorID string) []domain.Alert {
	var alerts []domain.Alert
	err := s.db.Where("sensor_id = ? AND is_active = ?", sensorID, true).Find(&alerts).Error
	if err != nil {
		s.logger.Error("Failed to get alert rules", zap.String("sensor_id", sensorID), zap.Error(err))
		return []domain.Alert{}
	}
	return alerts
}

func (s *AlertService) CreateAlert(alert *domain.Alert) error {
	return s.db.Create(alert).Error
}

func (s *AlertService) UpdateAlert(alert *domain.Alert) error {
	return s.db.Save(alert).Error
}

func (s *AlertService) DeleteAlert(id string) error {
	return s.db.Delete(&domain.Alert{}, "id = ?", id).Error
}

func (s *AlertService) GetAlertByID(id string) (*domain.Alert, error) {
	var alert domain.Alert
	err := s.db.First(&alert, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func (s *AlertService) GetAllAlerts() ([]domain.Alert, error) {
	var alerts []domain.Alert
	err := s.db.Find(&alerts).Error
	return alerts, err
}