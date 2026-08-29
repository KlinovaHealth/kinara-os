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

	"github.com/klinova/kinara-os/input-service/db"
	"github.com/klinova/kinara-os/input-service/middleware"
	"github.com/klinova/kinara-os/input-service/models"
)

var reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{Name: "input_requests_total"}, []string{"method", "path", "status"})

type Handler struct{ queries *db.Queries }

func New(q *db.Queries) *Handler { return &Handler{queries: q} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/inputs").Subrouter()
	api.HandleFunc("/purchases", h.createPurchase).Methods(http.MethodPost)
	api.HandleFunc("/purchases/farmer/{farmer_id}", h.listByFarmer).Methods(http.MethodGet)
	api.HandleFunc("/purchases/{id}/usage", h.recordUsage).Methods(http.MethodPost)
}

func (h *Handler) createPurchase(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	_ = claims
	var req models.CreatePurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id := uuid.New()
	p := models.InputPurchase{
		ID:          id,
		PurchaseRef: "INP-" + strings.ToUpper(id.String()[:8]),
		FarmerID:    req.FarmerID,
		CoopID:      req.CoopID,
		InputType:   req.InputType,
		InputName:   req.InputName,
		Quantity:    req.Quantity,
		Unit:        req.Unit,
		CostXOF:     req.CostXOF,
		Supplier:    req.Supplier,
		PurchasedAt: req.PurchasedAt,
		TenantID:    claims.TenantID,
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.queries.CreatePurchase(r.Context(), p); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}
	reqTotal.WithLabelValues("POST", "/inputs/purchases", "201").Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": p})
}

func (h *Handler) listByFarmer(w http.ResponseWriter, r *http.Request) {
	farmerID, err := uuid.Parse(mux.Vars(r)["farmer_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid farmer_id"})
		return
	}
	items, err := h.queries.ListByFarmer(r.Context(), farmerID, 50)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.InputPurchase{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

func (h *Handler) recordUsage(w http.ResponseWriter, r *http.Request) {
	purchaseID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req models.RecordUsageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	u := models.InputUsage{
		ID:         uuid.New(),
		PurchaseID: purchaseID,
		FieldID:    req.FieldID,
		Quantity:   req.Quantity,
		UsedAt:     req.UsedAt,
		Notes:      req.Notes,
	}
	if err := h.queries.RecordUsage(r.Context(), u); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "record failed"})
		return
	}
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": u})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
