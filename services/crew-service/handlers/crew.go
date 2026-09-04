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

	"github.com/klinova/kinara-os/crew-service/db"
	"github.com/klinova/kinara-os/crew-service/middleware"
	"github.com/klinova/kinara-os/crew-service/models"
)

var reqTotal = promauto.NewCounterVec(prometheus.CounterOpts{Name: "crew_requests_total"}, []string{"method", "path", "status"})

type Handler struct{ queries *db.Queries }

func New(q *db.Queries) *Handler { return &Handler{queries: q} }

func (h *Handler) Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1/crew").Subrouter()
	api.HandleFunc("", h.register).Methods(http.MethodPost)
	api.HandleFunc("/{id}", h.getCrew).Methods(http.MethodGet)
	api.HandleFunc("/{id}/vessel", h.assignVessel).Methods(http.MethodPut)
	api.HandleFunc("/{id}/certifications", h.addCertification).Methods(http.MethodPost)
	api.HandleFunc("/{id}/certifications", h.listCertifications).Methods(http.MethodGet)
	api.HandleFunc("/vessel/{vessel_id}", h.listByVessel).Methods(http.MethodGet)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		respond(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !claims.IsAllowedRole("admin", "port_authority", "captain") {
		respond(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var req models.RegisterCrewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id := uuid.New()
	now := time.Now().UTC()
	c := models.CrewMember{
		ID:             id,
		CrewRef:        "CRW-" + strings.ToUpper(id.String()[:8]),
		FullName:       req.FullName,
		Nationality:    req.Nationality,
		PassportNumber: req.PassportNumber,
		Rank:           req.Rank,
		VesselID:       req.VesselID,
		IsActive:       true,
		TenantID:       claims.TenantID.String(),
		JoinedAt:       now,
		UpdatedAt:      now,
	}
	if err := h.queries.RegisterCrew(r.Context(), c); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "register failed"})
		return
	}
	reqTotal.WithLabelValues("POST", "/crew", "201").Inc()
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": c})
}

func (h *Handler) getCrew(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	c, err := h.queries.GetCrew(r.Context(), id)
	if err != nil {
		respond(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": c})
}

func (h *Handler) assignVessel(w http.ResponseWriter, r *http.Request) {
	crewID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req models.AssignVesselRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.queries.AssignVessel(r.Context(), crewID, req.VesselID, time.Now().UTC()); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "assign failed"})
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "crew_id": crewID, "vessel_id": req.VesselID})
}

func (h *Handler) addCertification(w http.ResponseWriter, r *http.Request) {
	crewID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req models.AddCertificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	status := models.CertValid
	if time.Until(req.ExpiresAt) < 30*24*time.Hour {
		status = models.CertExpiring
	}
	if req.ExpiresAt.Before(time.Now()) {
		status = models.CertExpired
	}
	cert := models.CrewCertification{
		ID:         uuid.New(),
		CrewID:     crewID,
		CertType:   req.CertType,
		CertNumber: req.CertNumber,
		IssuedBy:   req.IssuedBy,
		IssuedAt:   req.IssuedAt,
		ExpiresAt:  req.ExpiresAt,
		Status:     status,
	}
	if err := h.queries.AddCertification(r.Context(), cert); err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "add failed"})
		return
	}
	respond(w, http.StatusCreated, map[string]interface{}{"success": true, "data": cert})
}

func (h *Handler) listCertifications(w http.ResponseWriter, r *http.Request) {
	crewID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	certs, err := h.queries.ListCertifications(r.Context(), crewID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if certs == nil {
		certs = []models.CrewCertification{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": certs})
}

func (h *Handler) listByVessel(w http.ResponseWriter, r *http.Request) {
	vesselID, err := uuid.Parse(mux.Vars(r)["vessel_id"])
	if err != nil {
		respond(w, http.StatusBadRequest, map[string]string{"error": "invalid vessel_id"})
		return
	}
	members, err := h.queries.ListByVessel(r.Context(), vesselID)
	if err != nil {
		respond(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	if members == nil {
		members = []models.CrewMember{}
	}
	respond(w, http.StatusOK, map[string]interface{}{"success": true, "data": members})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
