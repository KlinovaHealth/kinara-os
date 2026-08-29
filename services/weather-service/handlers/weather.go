package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/weather-service/db"
	"github.com/klinova/kinara-os/weather-service/middleware"
	"github.com/klinova/kinara-os/weather-service/models"
)

// Store abstracts the database layer for testability.
type Store interface {
	CreateForecast(ctx context.Context, f models.WeatherForecast) error
	GetForecast(ctx context.Context, id uuid.UUID) (*models.WeatherForecast, error)
	ListForecasts(ctx context.Context, p db.ListForecastsParams) ([]models.WeatherForecast, error)

	CreateAlert(ctx context.Context, a models.WeatherAlert) error
	GetAlert(ctx context.Context, id uuid.UUID) (*models.WeatherAlert, error)
	ListActiveAlerts(ctx context.Context, country, region string) ([]models.WeatherAlert, error)
	DeactivateAlert(ctx context.Context, id uuid.UUID, now time.Time) error

	CreateAdvisory(ctx context.Context, a models.PestAdvisory) error
	GetAdvisory(ctx context.Context, id uuid.UUID) (*models.PestAdvisory, error)
	ListAdvisories(ctx context.Context, country, region, cropType string, riskLevel *models.RiskLevel) ([]models.PestAdvisory, error)

	CreateObservation(ctx context.Context, o models.WeatherObservation) error
	ListObservations(ctx context.Context, country, region string, from time.Time, limit int) ([]models.WeatherObservation, error)

	InsertAuditLog(ctx context.Context, log models.WeatherAuditLog) error
}

type WeatherHandler struct {
	s Store
}

func NewWeatherHandler(q *db.Queries) *WeatherHandler         { return &WeatherHandler{s: q} }
func NewWeatherHandlerWithStore(s Store) *WeatherHandler       { return &WeatherHandler{s: s} }

func (h *WeatherHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/forecasts", h.createForecast).Methods(http.MethodPost)
	r.HandleFunc("/forecasts", h.listForecasts).Methods(http.MethodGet)
	r.HandleFunc("/forecasts/{id}", h.getForecast).Methods(http.MethodGet)

	r.HandleFunc("/alerts", h.createAlert).Methods(http.MethodPost)
	r.HandleFunc("/alerts", h.listAlerts).Methods(http.MethodGet)
	r.HandleFunc("/alerts/{id}", h.getAlert).Methods(http.MethodGet)
	r.HandleFunc("/alerts/{id}/deactivate", h.deactivateAlert).Methods(http.MethodPut)

	r.HandleFunc("/advisories", h.createAdvisory).Methods(http.MethodPost)
	r.HandleFunc("/advisories", h.listAdvisories).Methods(http.MethodGet)
	r.HandleFunc("/advisories/{id}", h.getAdvisory).Methods(http.MethodGet)

	r.HandleFunc("/observations", h.submitObservation).Methods(http.MethodPost)
	r.HandleFunc("/observations", h.listObservations).Methods(http.MethodGet)
}

// ─── Forecasts ────────────────────────────────────────────────────────────────

func (h *WeatherHandler) createForecast(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	var req models.CreateForecastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Country == "" || req.ForecastDate == "" || req.Condition == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "country, forecast_date, and condition are required")
		return
	}
	forecastDate, err := time.Parse(time.RFC3339, req.ForecastDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DATE", "forecast_date must be RFC3339")
		return
	}
	if req.ForecastType == "" {
		req.ForecastType = models.ForecastDaily
	}
	if req.DataSource == "" {
		req.DataSource = "manual"
	}
	validHours := req.ValidHours
	if validHours <= 0 {
		validHours = 24
	}
	now := time.Now().UTC()
	f := models.WeatherForecast{
		ID:            uuid.New(),
		Country:       req.Country,
		Region:        req.Region,
		District:      req.District,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		ForecastType:  req.ForecastType,
		ForecastDate:  forecastDate.UTC(),
		Condition:     req.Condition,
		TempMinC:      req.TempMinC,
		TempMaxC:      req.TempMaxC,
		TempAvgC:      (req.TempMinC + req.TempMaxC) / 2,
		HumidityPct:   req.HumidityPct,
		WindSpeedKmh:  req.WindSpeedKmh,
		WindDirection: req.WindDirection,
		RainfallMm:    req.RainfallMm,
		RainfallProb:  req.RainfallProb,
		UVIndex:       req.UVIndex,
		DataSource:    req.DataSource,
		ValidUntil:    forecastDate.UTC().Add(time.Duration(validHours) * time.Hour),
		CreatedAt:     now,
	}
	if err := h.s.CreateForecast(r.Context(), f); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, f.ID, claims.UserID, "create_forecast", "forecast")
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: f})
}

func (h *WeatherHandler) listForecasts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := queryInt(q.Get("page"), 1)
	limit := queryInt(q.Get("limit"), 20)
	if limit > 200 {
		limit = 200
	}
	p := db.ListForecastsParams{Country: q.Get("country"), Region: q.Get("region"), Page: page, Limit: limit}
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			tUTC := t.UTC()
			p.DateFrom = &tUTC
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			tUTC := t.UTC()
			p.DateTo = &tUTC
		}
	}
	if v := q.Get("type"); v != "" {
		ft := models.ForecastType(v)
		p.Type = &ft
	}
	forecasts, err := h.s.ListForecasts(r.Context(), p)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: forecasts})
}

func (h *WeatherHandler) getForecast(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid forecast id")
		return
	}
	f, err := h.s.GetForecast(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "forecast not found")
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: f})
}

// ─── Alerts ───────────────────────────────────────────────────────────────────

func (h *WeatherHandler) createAlert(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	var req models.CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.AlertType == "" || req.Country == "" || req.Title == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "alert_type, country, and title are required")
		return
	}
	if req.Severity == "" {
		req.Severity = models.SeverityWarning
	}
	now := time.Now().UTC()
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			tUTC := t.UTC()
			expiresAt = &tUTC
		}
	}
	if req.AffectedCrops == nil {
		req.AffectedCrops = []string{}
	}
	alert := models.WeatherAlert{
		ID:            uuid.New(),
		AlertType:     req.AlertType,
		Severity:      req.Severity,
		Country:       req.Country,
		Region:        req.Region,
		District:      req.District,
		Title:         req.Title,
		Description:   req.Description,
		Instructions:  req.Instructions,
		AffectedCrops: req.AffectedCrops,
		IssuedAt:      now,
		ExpiresAt:     expiresAt,
		Active:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := h.s.CreateAlert(r.Context(), alert); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, alert.ID, claims.UserID, "create_alert", "alert")
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: alert})
}

func (h *WeatherHandler) listAlerts(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")
	region := r.URL.Query().Get("region")
	alerts, err := h.s.ListActiveAlerts(r.Context(), country, region)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: alerts})
}

func (h *WeatherHandler) getAlert(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid alert id")
		return
	}
	a, err := h.s.GetAlert(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "alert not found")
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: a})
}

func (h *WeatherHandler) deactivateAlert(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid alert id")
		return
	}
	now := time.Now().UTC()
	if err := h.s.DeactivateAlert(r.Context(), id, now); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, id, claims.UserID, "deactivate_alert", "alert")
	updated, _ := h.s.GetAlert(r.Context(), id)
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: updated})
}

// ─── Pest/Disease advisories ──────────────────────────────────────────────────

func (h *WeatherHandler) createAdvisory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	var req models.CreateAdvisoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.PestName == "" || req.Country == "" || req.Description == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "pest_name, country, and description are required")
		return
	}
	if req.PestType == "" {
		req.PestType = "pest"
	}
	if req.RiskLevel == "" {
		req.RiskLevel = models.RiskModerate
	}
	if req.AffectedCrops == nil {
		req.AffectedCrops = []string{}
	}
	now := time.Now().UTC()
	validFrom := now
	if req.ValidFrom != "" {
		if t, err := time.Parse(time.RFC3339, req.ValidFrom); err == nil {
			validFrom = t.UTC()
		}
	}
	var validUntil *time.Time
	if req.ValidUntil != nil {
		if t, err := time.Parse(time.RFC3339, *req.ValidUntil); err == nil {
			tUTC := t.UTC()
			validUntil = &tUTC
		}
	}
	advisory := models.PestAdvisory{
		ID:            uuid.New(),
		PestName:      req.PestName,
		PestType:      req.PestType,
		AffectedCrops: req.AffectedCrops,
		Country:       req.Country,
		Region:        req.Region,
		RiskLevel:     req.RiskLevel,
		Description:   req.Description,
		Symptoms:      req.Symptoms,
		Prevention:    req.Prevention,
		Treatment:     req.Treatment,
		ReportedCases: 0,
		ValidFrom:     validFrom,
		ValidUntil:    validUntil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := h.s.CreateAdvisory(r.Context(), advisory); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, advisory.ID, claims.UserID, "create_advisory", "pest_advisory")
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: advisory})
}

func (h *WeatherHandler) listAdvisories(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	country := q.Get("country")
	region := q.Get("region")
	cropType := q.Get("crop_type")
	var riskLevel *models.RiskLevel
	if v := q.Get("risk_level"); v != "" {
		rl := models.RiskLevel(v)
		riskLevel = &rl
	}
	advisories, err := h.s.ListAdvisories(r.Context(), country, region, cropType, riskLevel)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: advisories})
}

func (h *WeatherHandler) getAdvisory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid advisory id")
		return
	}
	a, err := h.s.GetAdvisory(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "advisory not found")
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: a})
}

// ─── Observations ─────────────────────────────────────────────────────────────

func (h *WeatherHandler) submitObservation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	var req models.SubmitObservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Country == "" || req.Condition == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "country and condition are required")
		return
	}
	now := time.Now().UTC()
	observedAt := now
	if req.ObservedAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ObservedAt); err == nil {
			observedAt = t.UTC()
		}
	}
	reporterID, _ := uuid.Parse(claims.UserID)
	obs := models.WeatherObservation{
		ID:           uuid.New(),
		ReporterID:   reporterID,
		Country:      req.Country,
		Region:       req.Region,
		District:     req.District,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		ObservedAt:   observedAt,
		TempC:        req.TempC,
		RainfallMm:   req.RainfallMm,
		HumidityPct:  req.HumidityPct,
		WindSpeedKmh: req.WindSpeedKmh,
		Condition:    req.Condition,
		Notes:        req.Notes,
		CreatedAt:    now,
	}
	if err := h.s.CreateObservation(r.Context(), obs); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: obs})
}

func (h *WeatherHandler) listObservations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	country := q.Get("country")
	region := q.Get("region")
	days := queryInt(q.Get("days"), 7)
	limit := queryInt(q.Get("limit"), 100)
	if limit > 500 {
		limit = 500
	}
	from := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	obs, err := h.s.ListObservations(r.Context(), country, region, from, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: obs})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (h *WeatherHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid, _ := uuid.Parse(userID)
	eid := entityID
	log := models.WeatherAuditLog{
		ID:        uuid.New(),
		EntityID:  &eid,
		UserID:    uid,
		Action:    action,
		Resource:  resource,
		IPAddress: r.RemoteAddr,
		CreatedAt: time.Now().UTC(),
	}
	h.s.InsertAuditLog(r.Context(), log)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: code, Message: msg},
	})
}

func queryInt(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil && v > 0 {
		return v
	}
	return def
}

// splitCSV splits a comma-separated query param (kept for potential future use).
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}
