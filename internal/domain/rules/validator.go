package rules

import (
	"telemetry-system/internal/domain"
	"fmt"
)

type Validator struct {
	ranges map[string]ValueRange
}

type ValueRange struct {
	Min float64
	Max float64
}

func NewValidator() *Validator {
	return &Validator{
		ranges: map[string]ValueRange{
			"temperature": {Min: -50, Max: 150},
			"pressure":    {Min: 0, Max: 100},
			"current":     {Min: 4, Max: 20},
			"voltage":     {Min: 0, Max: 10},
			"humidity":    {Min: 0, Max: 100},
		},
	}
}

func (v *Validator) Validate(data *domain.Measurement) domain.ValidationResult {
	if data == nil {
		return domain.ValidationResult{
			IsValid: false,
			Error:   "data is nil",
		}
	}

	if data.SensorID == "" {
		return domain.ValidationResult{
			IsValid: false,
			Error:   "sensor_id is required",
		}
	}

	if data.Timestamp.IsZero() {
		return domain.ValidationResult{
			IsValid: false,
			Error:   "timestamp is required",
		}
	}

	// Проверка на NaN
	if data.Value != data.Value {
		return domain.ValidationResult{
			IsValid: false,
			Error:   "value is NaN",
		}
	}

	rangeConfig, exists := v.ranges[data.SensorID]
	if exists {
		if data.Value < rangeConfig.Min || data.Value > rangeConfig.Max {
			return domain.ValidationResult{
				IsValid: false,
				Error:   fmt.Sprintf("value %.2f out of range [%.2f, %.2f]", 
					data.Value, rangeConfig.Min, rangeConfig.Max),
			}
		}
	}

	return domain.ValidationResult{
		IsValid: true,
		Error:   "",
	}
}

func (v *Validator) AddRange(sensorType string, min, max float64) {
	v.ranges[sensorType] = ValueRange{Min: min, Max: max}
}