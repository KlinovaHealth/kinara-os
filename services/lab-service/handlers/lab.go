package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/klinova/kinara-os/lab-service/db"
	"github.com/klinova/kinara-os/lab-service/middleware"
	"github.com/klinova/kinara-os/lab-service/models"
)

// Store is the interface the handler requires. *db.Queries satisfies it.
type Store interface {
	CreateOrder(ctx context.Context, o models.LabOrder) error
	GetOrder(ctx context.Context, id uuid.UUID) (*models.LabOrder, error)
	UploadResult(ctx context.Context, r models.LabResult) error
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string, completedAt time.Time) error
	ListPatientResults(ctx context.Context, patientID uuid.UUID) ([]models.LabResultWithOrder, error)
	GetOrderStatus(ctx context.Context, id uuid.UUID) (string, error)
	GetTestCatalog(ctx context.Context, testCode string) (*models.TestCatalogEntry, error)
	InsertAudit(ctx context.Context, orderID, actorID, action string) error
}

var (
	labOrdersTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lab_orders_total",
		Help: "Total number of lab orders created.",
	})
	labResultsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lab_results_total",
		Help: "Total number of lab results uploaded.",
	})
)

type Handler struct{ store Store }

// New constructs a Handler backed by a real *db.Queries.
func New(q *db.Queries) *Handler { return &Handler{store: q} }

// NewWithStore constructs a Handler backed by any Store (useful for testing).
func NewWithStore(s Store) *Handler { return &Handler{store: s} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/lab").Subrouter()
	api.HandleFunc("/orders", h.createOrder).Methods(http.MethodPost)
	api.HandleFunc("/orders/{id}", h.getOrder).Methods(http.MethodGet)
	api.HandleFunc("/orders/{id}/result", h.uploadResult).Methods(http.MethodPost)
	api.HandleFunc("/orders/{id}/status", h.getOrderStatus).Methods(http.MethodGet)
	api.HandleFunc("/patient/{patient_id}/results", h.listPatientResults).Methods(http.MethodGet)
	api.HandleFunc("/catalog/{test_code}", h.getTestCatalog).Methods(http.MethodGet)
}

// interpretFlag classifies a numeric result against the normal range.
func interpretFlag(value, low, high float64) string {
	if value < low*0.5 || value > high*2.0 {
		return "critical"
	}
	if value < low || value > high {
		return "abnormal"
	}
	return "normal"
}

// POST /api/v1/lab/orders
func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("doctor", "nurse", "admin") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.Priority == "" {
		req.Priority = "routine"
	}

	id := uuid.New()
	o := models.LabOrder{
		ID:        id,
		OrderRef:  "LAB-" + strings.ToUpper(id.String()[:8]),
		PatientID: req.PatientID,
		OrderedBy: req.OrderedBy,
		ClinicID:  req.ClinicID,
		TestCode:  req.TestCode,
		TestName:  req.TestName,
		Priority:  req.Priority,
		Status:    models.OrderPending,
		Notes:     req.Notes,
		TenantID:  claims.TenantID.String(),
		OrderedAt: time.Now().UTC(),
	}

	if err := h.store.CreateOrder(r.Context(), o); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}

	// Audit: INSERT immediately, never query first.
	_ = h.store.InsertAudit(r.Context(), o.ID.String(), claims.UserID.String(), "create_order")

	labOrdersTotal.Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": o})
}

// GET /api/v1/lab/orders/{id}
func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	o, err := h.store.GetOrder(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": o})
}

// POST /api/v1/lab/orders/{id}/result
func (h *Handler) uploadResult(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("lab_tech", "doctor", "admin") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	orderID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid order id"})
		return
	}

	_, err = h.store.GetOrder(r.Context(), orderID)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}

	var req models.UploadResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	flag := interpretFlag(req.ResultValue, req.NormalLow, req.NormalHigh)

	now := time.Now().UTC()
	result := models.LabResult{
		ID:          uuid.New(),
		OrderID:     orderID,
		ResultValue: req.ResultValue,
		Unit:        req.Unit,
		NormalLow:   req.NormalLow,
		NormalHigh:  req.NormalHigh,
		Flag:        flag,
		Notes:       req.Notes,
		RecordedBy:  claims.UserID,
		RecordedAt:  now,
	}

	if err := h.store.UploadResult(r.Context(), result); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}

	// Update order status to completed.
	_ = h.store.UpdateOrderStatus(r.Context(), orderID, "completed", now)

	// Audit: INSERT immediately, never query first.
	_ = h.store.InsertAudit(r.Context(), orderID.String(), claims.UserID.String(), "upload_result")

	labResultsTotal.Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": result})
}

// GET /api/v1/lab/patient/{patient_id}/results
func (h *Handler) listPatientResults(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	patientID, err := uuid.Parse(mux.Vars(r)["patient_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid patient_id"})
		return
	}

	results, err := h.store.ListPatientResults(r.Context(), patientID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if results == nil {
		results = []models.LabResultWithOrder{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": results})
}

// GET /api/v1/lab/orders/{id}/status
func (h *Handler) getOrderStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	status, err := h.store.GetOrderStatus(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]string{"status": status}})
}

// GET /api/v1/lab/catalog/{test_code}
func (h *Handler) getTestCatalog(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	testCode := mux.Vars(r)["test_code"]
	entry, err := h.store.GetTestCatalog(r.Context(), testCode)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "test code not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": entry})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
