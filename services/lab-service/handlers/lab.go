package handlers

import (
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

var reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{Name: "lab_requests_total"}, []string{"method", "path", "status"})

type Handler struct{ queries *db.Queries }

func New(q *db.Queries) *Handler { return &Handler{queries: q} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/lab").Subrouter()
	api.HandleFunc("/orders", h.createOrder).Methods(http.MethodPost)
	api.HandleFunc("/orders/{id}", h.getOrder).Methods(http.MethodGet)
	api.HandleFunc("/orders/{id}/result", h.recordResult).Methods(http.MethodPost)
	api.HandleFunc("/patient/{patient_id}/orders", h.listOrders).Methods(http.MethodGet)
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("admin", "nurse", "doctor") {
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
		TenantID:  claims.TenantID,
		OrderedAt: time.Now().UTC(),
	}
	if err := h.queries.CreateOrder(r.Context(), o); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}
	reqTotal.WithLabelValues("POST", "/lab/orders", "201").Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": o})
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	o, err := h.queries.GetOrder(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": o})
}

func (h *Handler) recordResult(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	_ = claims
	orderID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid order id"})
		return
	}
	order, err := h.queries.GetOrder(r.Context(), orderID)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}
	var req models.RecordResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	now := time.Now().UTC()
	result := models.LabResult{
		ID:             uuid.New(),
		OrderID:        orderID,
		PatientID:      order.PatientID,
		TestCode:       order.TestCode,
		ResultValue:    req.ResultValue,
		Unit:           req.Unit,
		ReferenceRange: req.ReferenceRange,
		Flag:           req.Flag,
		AnalyzedBy:     req.AnalyzedBy,
		ResultAt:       now,
		TenantID:       order.TenantID,
	}
	if err := h.queries.InsertResult(r.Context(), result); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}
	_ = h.queries.MarkCompleted(r.Context(), orderID, now)
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": result})
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	patientID, err := uuid.Parse(mux.Vars(r)["patient_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid patient_id"})
		return
	}
	orders, err := h.queries.ListByPatient(r.Context(), patientID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if orders == nil {
		orders = []models.LabOrder{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": orders})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
