package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           string         `gorm:"primaryKey;type:text;size:36" json:"id"`
	Username     string         `gorm:"unique;type:text;size:50" json:"username"`
	PasswordHash string         `gorm:"type:text;size:255" json:"-"`
	Role         string         `gorm:"type:text;size:20" json:"role"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}

type Gateway struct {
	ID        string         `gorm:"primaryKey;type:text;size:36" json:"id"`
	Name      string         `gorm:"type:text;size:100" json:"name"`
	Location  string         `gorm:"type:text;size:255" json:"location"`
	LastSeen  time.Time      `json:"last_seen"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Sensors   []Sensor       `gorm:"foreignKey:GatewayID" json:"sensors,omitempty"`
}

func (Gateway) TableName() string {
	return "gateways"
}

type Sensor struct {
	ID        string         `gorm:"primaryKey;type:text;size:36" json:"id"`
	GatewayID string         `gorm:"type:text;size:36;index" json:"gateway_id"`
	Type      string         `gorm:"type:text;size:50" json:"type"`
	Unit      string         `gorm:"type:text;size:20" json:"unit"`
	MinValue  float64        `gorm:"type:real" json:"min_value"`
	MaxValue  float64        `gorm:"type:real" json:"max_value"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Gateway   Gateway        `gorm:"foreignKey:GatewayID" json:"gateway,omitempty"`
	Alerts    []Alert        `gorm:"foreignKey:SensorID" json:"alerts,omitempty"`
}

func (Sensor) TableName() string {
	return "sensors"
}

type Alert struct {
	ID        string         `gorm:"primaryKey;type:text;size:36" json:"id"`
	SensorID  string         `gorm:"type:text;size:36;index" json:"sensor_id"`
	Threshold float64        `gorm:"type:real" json:"threshold"`
	Condition string         `gorm:"type:text;size:10" json:"condition"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedBy string         `gorm:"type:text;size:36" json:"created_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Sensor    Sensor         `gorm:"foreignKey:SensorID" json:"sensor,omitempty"`
}

func (Alert) TableName() string {
	return "alerts"
}

type Measurement struct {
	SensorID  string                 `json:"sensor_id"`
	GatewayID string                 `json:"gateway_id"`
	Value     float64                `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (Measurement) InfluxMeasurementName() string {
	return "telemetry"
}

func (m *Measurement) ToInfluxTags() map[string]string {
	return map[string]string{
		"sensor_id":  m.SensorID,
		"gateway_id": m.GatewayID,
	}
}

func (m *Measurement) ToInfluxFields() map[string]interface{} {
	return map[string]interface{}{
		"value": m.Value,
	}
}

type AlertLog struct {
	AlertID      string    `json:"alert_id"`
	SensorID     string    `json:"sensor_id"`
	GatewayID    string    `json:"gateway_id"`
	Timestamp    time.Time `json:"timestamp"`
	Value        float64   `json:"value"`
	Acknowledged bool      `json:"acknowledged"`
	Message      string    `json:"message"`
}

func (AlertLog) InfluxMeasurementName() string {
	return "alert_log"
}

func (a *AlertLog) ToInfluxTags() map[string]string {
	return map[string]string{
		"alert_id":   a.AlertID,
		"sensor_id":  a.SensorID,
		"gateway_id": a.GatewayID,
	}
}

func (a *AlertLog) ToInfluxFields() map[string]interface{} {
	return map[string]interface{}{
		"value":        a.Value,
		"acknowledged": a.Acknowledged,
		"message":      a.Message,
	}
}

type UserRole string

const (
	RoleOperator UserRole = "operator"
	RoleEngineer UserRole = "engineer"
	RoleAdmin    UserRole = "admin"
)

func (r UserRole) IsValid() bool {
	switch r {
	case RoleOperator, RoleEngineer, RoleAdmin:
		return true
	}
	return false
}

type AlertCondition string

const (
	ConditionGreaterThan AlertCondition = "gt"
	ConditionLessThan    AlertCondition = "lt"
)

func (c AlertCondition) IsValid() bool {
	switch c {
	case ConditionGreaterThan, ConditionLessThan:
		return true
	}
	return false
}

// DTO для API
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

type CreateUserRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Password string   `json:"password" binding:"required,min=6"`
	Role     UserRole `json:"role" binding:"required,oneof=operator engineer admin"`
}

type CreateGatewayRequest struct {
	Name     string `json:"name" binding:"required,max=100"`
	Location string `json:"location" binding:"max=255"`
}

type CreateSensorRequest struct {
	GatewayID string  `json:"gateway_id" binding:"required"`
	Type      string  `json:"type" binding:"required"`
	Unit      string  `json:"unit" binding:"required"`
	MinValue  float64 `json:"min_value"`
	MaxValue  float64 `json:"max_value"`
}

type CreateAlertRequest struct {
	SensorID  string         `json:"sensor_id" binding:"required"`
	Threshold float64        `json:"threshold" binding:"required"`
	Condition AlertCondition `json:"condition" binding:"required,oneof=gt lt"`
}

type HealthResponse struct {
	Status     string            `json:"status"`
	Timestamp  time.Time         `json:"timestamp"`
	Components map[string]string `json:"components"`
	Stats      interface{}       `json:"stats,omitempty"`
}

type IngestorStats struct {
	TotalReceived   int64     `json:"total_received"`
	TotalValid      int64     `json:"total_valid"`
	TotalInvalid    int64     `json:"total_invalid"`
	AlertsTriggered int64     `json:"alerts_triggered"`
	LastUpdateTime  time.Time `json:"last_update_time"`
}

type ValidationResult struct {
	IsValid bool
	Error   string
}