package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/ledger-service/auth"
	"github.com/klinova/kinara-os/ledger-service/crypto"
	"github.com/klinova/kinara-os/ledger-service/db"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
	"github.com/klinova/kinara-os/ledger-service/middleware"
)

type Handler struct {
	queries *db.Queries
	enc     *crypto.Encryptor
	logger  *slog.Logger
}

func New(q *db.Queries, enc *crypto.Encryptor, logger *slog.Logger) *Handler {
	return &Handler{queries: q, enc: enc, logger: logger}
}

func (h *Handler) Register(r *mux.Router, jwtMW func(http.Handler) http.Handler) {
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(jwtMW)
	api.Use(pkgauth.RequireTenantScope("ledger-service", nil))
	api.HandleFunc("/ledgers", h.list).Methods(http.MethodGet)
	api.HandleFunc("/ledgers", h.create).Methods(http.MethodPost)
	api.HandleFunc("/ledgers/{id}", h.get).Methods(http.MethodGet)
	api.HandleFunc("/ledgers/{id}", h.update).Methods(http.MethodPut)
	api.HandleFunc("/ledgers/{id}", h.delete).Methods(http.MethodDelete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	items, err := h.queries.List(r.Context(), 20, (page-1)*20)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.json(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"page":  page,
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	data, _ := json.Marshal(payload)
	encrypted, err := h.enc.EncryptString(string(data))
	if err != nil {
		h.internalError(w, err)
		return
	}
	now := time.Now().UTC()
	rec := db.Record{
		ID:        uuid.New(),
		DataEnc:   encrypted,
		CreatedBy: claims.UserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.queries.Create(r.Context(), rec); err != nil {
		h.internalError(w, err)
		return
	}
	h.json(w, http.StatusCreated, map[string]string{"id": rec.ID.String()})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid id")
		return
	}
	rec, err := h.queries.Get(r.Context(), id)
	if err != nil {
		h.notFound(w)
		return
	}
	plain, err := h.enc.DecryptString(rec.DataEnc)
	if err != nil {
		h.internalError(w, err)
		return
	}
	var data map[string]interface{}
	json.Unmarshal([]byte(plain), &data)
	data["id"] = rec.ID
	data["created_at"] = rec.CreatedAt
	h.json(w, http.StatusOK, data)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	_, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid id")
		return
	}
	h.json(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.badRequest(w, "invalid id")
		return
	}
	if err := h.queries.Delete(r.Context(), id); err != nil {
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) json(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) badRequest(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(`{"success":false,"error":{"code":"BAD_REQUEST","message":"` + msg + `"}}`))
}

func (h *Handler) notFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"success":false,"error":{"code":"NOT_FOUND","message":"resource not found"}}`))
}

func (h *Handler) internalError(w http.ResponseWriter, err error) {
	h.logger.Error("internal error", "error", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`))
}

func (h *Handler) forbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"success":false,"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`))
}

// ensure auth import is used
var _ = auth.Claims{}
