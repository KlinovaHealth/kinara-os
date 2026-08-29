package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/logistics-analytics-service/db"
	"github.com/klinova/kinara-os/logistics-analytics-service/middleware"
	"github.com/klinova/kinara-os/logistics-analytics-service/models"
)

type Store interface {
	RecordMetric(ctx context.Context, m models.LogisticsMetric) error
	ListMetrics(ctx context.Context, p db.ListMetricsParams) ([]models.LogisticsMetric, error)
	CreateForecast(ctx context.Context, f models.DemandForecast) error
	GetForecast(ctx context.Context, id uuid.UUID) (*models.DemandForecast, error)
	ListForecasts(ctx context.Context, country string) ([]models.DemandForecast, error)
	InsertAuditLog(ctx context.Context, l models.AnalyticsAuditLog) error
}

type AnalyticsHandler struct{ s Store }

func NewAnalyticsHandler(q *db.Queries) *AnalyticsHandler       { return &AnalyticsHandler{s:q} }
func NewAnalyticsHandlerWithStore(s Store) *AnalyticsHandler     { return &AnalyticsHandler{s:s} }

func (h *AnalyticsHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/metrics", h.recordMetric).Methods(http.MethodPost)
	r.HandleFunc("/metrics", h.listMetrics).Methods(http.MethodGet)
	r.HandleFunc("/forecasts", h.createForecast).Methods(http.MethodPost)
	r.HandleFunc("/forecasts/{id}", h.getForecast).Methods(http.MethodGet)
	r.HandleFunc("/forecasts/country/{country}", h.listForecasts).Methods(http.MethodGet)
}

func (h *AnalyticsHandler) recordMetric(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.RecordMetricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.Country=="" || req.Period=="" { writeError(w,400,"VALIDATION_ERROR","country and period required"); return }
	pStart,err := time.Parse(time.RFC3339,req.PeriodStart); if err != nil { writeError(w,400,"INVALID_DATE","period_start must be RFC3339"); return }
	pEnd,err := time.Parse(time.RFC3339,req.PeriodEnd); if err != nil { writeError(w,400,"INVALID_DATE","period_end must be RFC3339"); return }
	if req.Currency=="" { req.Currency="USD" }
	now := time.Now().UTC()
	m := models.LogisticsMetric{ID:uuid.New(), Period:req.Period, PeriodStart:pStart.UTC(), PeriodEnd:pEnd.UTC(),
		Country:req.Country, TotalTrips:req.TotalTrips, TotalDistanceKm:req.TotalDistanceKm,
		TotalDeliveries:req.TotalDeliveries, SuccessfulDeliveries:req.SuccessfulDeliveries,
		OnTimeDeliveries:req.OnTimeDeliveries, AvgCostPerKm:req.AvgCostPerKm, TotalRevenue:req.TotalRevenue,
		Currency:req.Currency, BottleneckRoute:req.BottleneckRoute, BottleneckWarehouse:req.BottleneckWarehouse, CreatedAt:now}
	if err := h.s.RecordMetric(r.Context(), m); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,201,models.APIResponse{Success:true, Data:m})
}

func (h *AnalyticsHandler) listMetrics(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := db.ListMetricsParams{Page:queryInt(q.Get("page"),1), Limit:queryInt(q.Get("limit"),30)}
	if v:=q.Get("country"); v!="" { p.Country=&v }
	if v:=q.Get("period"); v!="" { per:=models.MetricPeriod(v); p.Period=&per }
	metrics,err := h.s.ListMetrics(r.Context(), p)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:metrics})
}

func (h *AnalyticsHandler) createForecast(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreateForecastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.Country=="" || req.Route=="" || req.ForecastDate=="" { writeError(w,400,"VALIDATION_ERROR","country, route, forecast_date required"); return }
	fDate,err := time.Parse(time.RFC3339,req.ForecastDate); if err != nil { writeError(w,400,"INVALID_DATE","forecast_date must be RFC3339"); return }
	now := time.Now().UTC()
	f := models.DemandForecast{ID:uuid.New(), Country:req.Country, Route:req.Route, ForecastDate:fDate.UTC(),
		PredictedVolume:req.PredictedVolume, PredictedTrips:req.PredictedTrips, ConfidencePct:req.ConfidencePct,
		Notes:req.Notes, CreatedAt:now}
	if err := h.s.CreateForecast(r.Context(), f); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,201,models.APIResponse{Success:true, Data:f})
}

func (h *AnalyticsHandler) getForecast(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid forecast id"); return }
	f,err := h.s.GetForecast(r.Context(), id)
	if err != nil { writeError(w,404,"NOT_FOUND","forecast not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:f})
}

func (h *AnalyticsHandler) listForecasts(w http.ResponseWriter, r *http.Request) {
	country := mux.Vars(r)["country"]
	forecasts,err := h.s.ListForecasts(r.Context(), country)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:forecasts})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type","application/json"); w.WriteHeader(status); json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w,status,models.APIResponse{Success:false,Error:&models.APIError{Code:code,Message:msg}})
}
func queryInt(s string, def int) int {
	if v,err:=strconv.Atoi(s); err==nil && v>0 { return v }; return def
}
