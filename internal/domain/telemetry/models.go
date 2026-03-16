package domain

import "time"

type TelemetryData struct {
    DeviceID   string    `json:"device_id"`
    SensorType string    `json:"sensor_type"`
    Value      float64   `json:"value"`
    Unit       string    `json:"unit"`
    Timestamp  time.Time `json:"timestamp"`
}

type User struct {
    ID           string `json:"id" gorm:"primaryKey"` 
    Username     string `json:"username"`
    PasswordHash string `json:"-"` 
    Role         string `json:"role"`
}

type Sensor struct {
    ID        string  `json:"id" gorm:"primaryKey"` 
    GatewayID string  `json:"gateway_id"`
    Type      string  `json:"type"`
    Unit      string  `json:"unit"`
    MinValue  float64 `json:"min_value"`
    MaxValue  float64 `json:"max_value"`
}

type AlertRule struct {
    ID        string  `json:"id" gorm:"primaryKey"`
    SensorID  string  `json:"sensor_id"`
    Threshold float64 `json:"threshold"`
    Condition string  `json:"condition"` 
    IsActive  bool    `json:"is_active"`
}

type AlertLog struct {
    AlertID     string    `json:"alert_id"`
    Timestamp   time.Time `json:"timestamp"`
    Value       float64   `json:"value"`
    Acknowledged bool     `json:"acknowledged"`
}