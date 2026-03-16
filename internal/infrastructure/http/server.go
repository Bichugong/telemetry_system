package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"telemetry-system/internal/infrastructure/influx"
	"telemetry-system/internal/service/ingestor"
)

type Server struct {
	server  *http.Server
	handler *Handler
	logger  *zap.Logger
}

func NewServer(host string, port int, db *gorm.DB, 
	influxClient *influx.InfluxClient, ingestor *ingestor.Ingestor,
	jwtSecret string, logger *zap.Logger) *Server {

	handler := NewHandler(db, influxClient, ingestor, jwtSecret, logger)
	router := mux.NewRouter()

	// Public routes
	router.HandleFunc("/health", handler.HealthHandler).Methods("GET")
	router.HandleFunc("/api/login", handler.LoginHandler).Methods("POST")

	// Protected routes
	api := router.PathPrefix("/api").Subrouter()
	api.Use(AuthMiddleware(jwtSecret))

	// Sensors
	api.HandleFunc("/sensors", handler.GetSensorsHandler).Methods("GET")
	api.HandleFunc("/sensors", createSensorHandlerWithRole(handler, "engineer", "admin")).Methods("POST")

	// Alerts
	api.HandleFunc("/alerts", handler.GetAlertsHandler).Methods("GET")
	api.HandleFunc("/alerts", createAlertHandlerWithRole(handler, "engineer", "admin")).Methods("POST")

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		server:  srv,
		handler: handler,
		logger:  logger,
	}
}

func createSensorHandlerWithRole(h *Handler, allowedRoles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value(userRoleKey)
		if role == nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		
		roleStr := role.(string)
		for _, allowed := range allowedRoles {
			if roleStr == allowed {
				h.CreateSensorHandler(w, r)
				return
			}
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}

func createAlertHandlerWithRole(h *Handler, allowedRoles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value(userRoleKey)
		if role == nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		
		roleStr := role.(string)
		for _, allowed := range allowedRoles {
			if roleStr == allowed {
				h.CreateAlertHandler(w, r)
				return
			}
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}

func (s *Server) Start() error {
	s.logger.Info("HTTP server starting", zap.String("addr", s.server.Addr))
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}