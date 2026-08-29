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

	"github.com/klinova/kinara-os/outbreak-service/db"
	"github.com/klinova/kinara-os/outbreak-service/middleware"
	"github.com/klinova/kinara-os/outbreak-service/models"
)

var reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{Name: "outbreak_requests_total"}, []string{"method", "path", "status"})

type Handler struct{ queries *db.Queries }

func New(q *db.Queries) *Handler { return &Handler{queries: q} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/outbreak").Subrouter()
	api.HandleFunc("/responses", h.createResponse).Methods(http.MethodPost)
	api.HandleFunc("/responses", h.listResponses).Methods(http.MethodGet)
	api.HandleFunc("/responses/{id}", h.getResponse).Methods(http.MethodGet)
	api.HandleFunc("/responses/{id}/status", h.updateStatus).Methods(http.MethodPut)
	api.HandleFunc("/responses/{id}/actions", h.addAction).Methods(http.MethodPost)
}

func (h *Handler) createResponse(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("admin", "doctor", "epidemiologist") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var req models.CreateResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id := uuid.New()
	now := time.Now().UTC()
	resp := models.OutbreakResponse{
		ID:              id,
		ResponseRef:     "OR-" + strings.ToUpper(id.String()[:8]),
		AlertRef:        req.AlertRef,
		DiseaseName:     req.DiseaseName,
		Country:         req.Country,
		Region:          req.Region,
		Status:          models.ResponseActive,
		LeadCoordinator: req.LeadCoordinator,
		TeamSize:        req.TeamSize,
		CasesTargeted:   req.CasesTargeted,
		Population:      req.Population,
		TenantID:        claims.TenantID,
		StartedAt:       now,
		UpdatedAt:       now,
	}
	if err := h.queries.CreateResponse(r.Context(), resp); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "create failed"})
		return
	}
	reqTotal.WithLabelValues("POST", "/outbreak/responses", "201").Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": resp})
}

func (h *Handler) listResponses(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")
	items, err := h.queries.ListResponses(r.Context(), country, 50)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if items == nil {
		items = []models.OutbreakResponse{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

func (h *Handler) getResponse(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	resp, err := h.queries.GetResponse(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": resp})
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req struct {
		Status models.ResponseStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	now := time.Now().UTC()
	var resolvedAt *time.Time
	if req.Status == models.ResponseResolved {
		resolvedAt = &now
	}
	if err := h.queries.UpdateStatus(r.Context(), id, req.Status, resolvedAt, now); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "id": id})
}

func (h *Handler) addAction(w http.ResponseWriter, r *http.Request) {
	respID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req models.AddActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	action := models.ResponseAction{
		ID:          uuid.New(),
		ResponseID:  respID,
		ActionType:  req.ActionType,
		Description: req.Description,
		AssignedTo:  req.AssignedTo,
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.queries.AddAction(r.Context(), action); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "add failed"})
		return
	}
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": action})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
