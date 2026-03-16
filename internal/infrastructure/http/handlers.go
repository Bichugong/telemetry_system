package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"telemetry-system/internal/domain"
	"telemetry-system/internal/infrastructure/influx"
	"telemetry-system/internal/service/ingestor"
)

type Handler struct {
	db            *gorm.DB
	influxClient  *influx.InfluxClient
	ingestor      *ingestor.Ingestor
	jwtSecret     string
	logger        *zap.Logger
}

func NewHandler(db *gorm.DB, influxClient *influx.InfluxClient, 
	ingestor *ingestor.Ingestor, jwtSecret string, logger *zap.Logger) *Handler {
	return &Handler{
		db:           db,
		influxClient: influxClient,
		ingestor:     ingestor,
		jwtSecret:    jwtSecret,
		logger:       logger,
	}
}

func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	response := domain.HealthResponse{
		Status:     "healthy",
		Timestamp:  time.Now(),
		Components: make(map[string]string),
		Stats:      h.ingestor.GetStats(),
	}

	if err := h.influxClient.Health(r.Context()); err != nil {
		response.Components["influxdb"] = "unhealthy"
		response.Status = "unhealthy"
	} else {
		response.Components["influxdb"] = "healthy"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var user domain.User
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := domain.LoginResponse{
		Token:     tokenString,
		ExpiresAt: time.Now().Add(time.Hour * 24),
		User:      user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetSensorsHandler(w http.ResponseWriter, r *http.Request) {
	var sensors []domain.Sensor
	if err := h.db.Preload("Gateway").Find(&sensors).Error; err != nil {
		http.Error(w, "Failed to get sensors", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sensors)
}

func (h *Handler) CreateSensorHandler(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateSensorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sensor := domain.Sensor{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		GatewayID: req.GatewayID,
		Type:      req.Type,
		Unit:      req.Unit,
		MinValue:  req.MinValue,
		MaxValue:  req.MaxValue,
	}

	if err := h.db.Create(&sensor).Error; err != nil {
		http.Error(w, "Failed to create sensor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sensor)
}

func (h *Handler) GetAlertsHandler(w http.ResponseWriter, r *http.Request) {
	var alerts []domain.Alert
	if err := h.db.Preload("Sensor").Find(&alerts).Error; err != nil {
		http.Error(w, "Failed to get alerts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func (h *Handler) CreateAlertHandler(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	alert := domain.Alert{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		SensorID:  req.SensorID,
		Threshold: req.Threshold,
		Condition: string(req.Condition),
		IsActive:  true,
	}

	if err := h.db.Create(&alert).Error; err != nil {
		http.Error(w, "Failed to create alert", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(alert)
}