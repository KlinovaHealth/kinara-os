package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/cooperative-service/db"
	"github.com/klinova/kinara-os/cooperative-service/middleware"
	"github.com/klinova/kinara-os/cooperative-service/models"
)

// Store abstracts the database layer for testability.
type Store interface {
	CreateCoop(ctx context.Context, c models.Cooperative) error
	GetCoop(ctx context.Context, id uuid.UUID) (*models.Cooperative, error)
	ListCoops(ctx context.Context, p db.ListCoopsParams) ([]models.Cooperative, error)
	CountCoops(ctx context.Context, p db.ListCoopsParams) (int, error)
	UpdateCoopStats(ctx context.Context, id uuid.UUID, now time.Time) error

	AddMember(ctx context.Context, m models.CoopMember) error
	GetMember(ctx context.Context, id uuid.UUID) (*models.CoopMember, error)
	GetMemberByFarmer(ctx context.Context, coopID, farmerID uuid.UUID) (*models.CoopMember, error)
	ListMembers(ctx context.Context, coopID uuid.UUID, page, limit int) ([]models.CoopMember, error)
	UpdateMember(ctx context.Context, id uuid.UUID, req models.UpdateMemberRequest, now time.Time) error

	CreatePool(ctx context.Context, p models.SellingPool) error
	GetPool(ctx context.Context, id uuid.UUID) (*models.SellingPool, error)
	ListPools(ctx context.Context, coopID uuid.UUID, page, limit int) ([]models.SellingPool, error)
	ClosePool(ctx context.Context, id uuid.UUID, now time.Time) error
	RecordSale(ctx context.Context, id uuid.UUID, pricePerKg, totalRevenue float64, now time.Time) error
	AddPoolQuantity(ctx context.Context, poolID uuid.UUID, qty float64, now time.Time) error

	AddContribution(ctx context.Context, c models.PoolContribution) error
	GetContribution(ctx context.Context, id uuid.UUID) (*models.PoolContribution, error)
	ListContributions(ctx context.Context, poolID uuid.UUID) ([]models.PoolContribution, error)
	DistributePayouts(ctx context.Context, poolID uuid.UUID, totalRevenue float64, now time.Time) error
	MarkPayoutPaid(ctx context.Context, contributionID uuid.UUID, now time.Time) error

	InsertAuditLog(ctx context.Context, log models.CoopAuditLog) error
}

type CoopHandler struct {
	s Store
}

func NewCoopHandler(q *db.Queries) *CoopHandler          { return &CoopHandler{s: q} }
func NewCoopHandlerWithStore(s Store) *CoopHandler        { return &CoopHandler{s: s} }

func (h *CoopHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/cooperatives", h.createCoop).Methods(http.MethodPost)
	r.HandleFunc("/cooperatives", h.listCoops).Methods(http.MethodGet)
	r.HandleFunc("/cooperatives/{id}", h.getCoop).Methods(http.MethodGet)

	r.HandleFunc("/cooperatives/{id}/members", h.addMember).Methods(http.MethodPost)
	r.HandleFunc("/cooperatives/{id}/members", h.listMembers).Methods(http.MethodGet)
	r.HandleFunc("/members/{member_id}", h.getMember).Methods(http.MethodGet)
	r.HandleFunc("/members/{member_id}", h.updateMember).Methods(http.MethodPut)

	r.HandleFunc("/cooperatives/{id}/pools", h.createPool).Methods(http.MethodPost)
	r.HandleFunc("/cooperatives/{id}/pools", h.listPools).Methods(http.MethodGet)
	r.HandleFunc("/pools/{pool_id}", h.getPool).Methods(http.MethodGet)
	r.HandleFunc("/pools/{pool_id}/contribute", h.contribute).Methods(http.MethodPost)
	r.HandleFunc("/pools/{pool_id}/contributions", h.listContributions).Methods(http.MethodGet)
	r.HandleFunc("/pools/{pool_id}/close", h.closePool).Methods(http.MethodPut)
	r.HandleFunc("/pools/{pool_id}/sale", h.recordSale).Methods(http.MethodPut)
	r.HandleFunc("/contributions/{contribution_id}/payout", h.markPaid).Methods(http.MethodPut)
}

// ─── Cooperatives ─────────────────────────────────────────────────────────────

func (h *CoopHandler) createCoop(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	var req models.CreateCoopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Name == "" || req.Country == "" || req.ContactPhone == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name, country, and contact_phone are required")
		return
	}
	if req.CoopType == "" {
		req.CoopType = models.CoopMulti
	}
	now := time.Now().UTC()
	coop := models.Cooperative{
		ID:             uuid.New(),
		Name:           req.Name,
		RegistrationNo: req.RegistrationNo,
		CoopType:       req.CoopType,
		Status:         models.CoopActive,
		Country:        req.Country,
		Region:         req.Region,
		District:       req.District,
		TotalMembers:   0,
		TotalFarmHa:    0,
		Description:    req.Description,
		ContactPhone:   req.ContactPhone,
		ContactEmail:   req.ContactEmail,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := h.s.CreateCoop(r.Context(), coop); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, coop.ID, claims.UserID, "create_coop", "cooperative")
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: coop})
}

func (h *CoopHandler) listCoops(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := queryInt(q.Get("page"), 1)
	limit := queryInt(q.Get("limit"), 20)
	if limit > 100 {
		limit = 100
	}
	p := db.ListCoopsParams{Page: page, Limit: limit}
	if v := q.Get("country"); v != "" {
		p.Country = &v
	}
	if v := q.Get("region"); v != "" {
		p.Region = &v
	}
	coops, err := h.s.ListCoops(r.Context(), p)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	total, _ := h.s.CountCoops(r.Context(), p)
	tp := total / limit
	if total%limit != 0 {
		tp++
	}
	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    coops,
		Meta:    &models.PageMeta{Page: page, Limit: limit, Total: total, TotalPages: tp},
	})
}

func (h *CoopHandler) getCoop(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid cooperative id")
		return
	}
	coop, err := h.s.GetCoop(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "cooperative not found")
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: coop})
}

// ─── Members ──────────────────────────────────────────────────────────────────

func (h *CoopHandler) addMember(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	coopID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid cooperative id")
		return
	}
	if _, err := h.s.GetCoop(r.Context(), coopID); err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "cooperative not found")
		return
	}
	var req models.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	farmerID, err := uuid.Parse(req.FarmerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FARMER_ID", "invalid farmer_id")
		return
	}
	if req.Role == "" {
		req.Role = models.RoleMember
	}
	if req.SharesHeld < 1 {
		req.SharesHeld = 1
	}
	now := time.Now().UTC()
	member := models.CoopMember{
		ID:         uuid.New(),
		CoopID:     coopID,
		FarmerID:   farmerID,
		Role:       req.Role,
		Status:     models.MemberActive,
		SharesHeld: req.SharesHeld,
		JoinedAt:   now,
		UpdatedAt:  now,
	}
	if err := h.s.AddMember(r.Context(), member); err != nil {
		writeError(w, http.StatusConflict, "ALREADY_MEMBER", "farmer is already a member")
		return
	}
	h.s.UpdateCoopStats(r.Context(), coopID, now)
	h.audit(r, member.ID, claims.UserID, "add_member", "member")
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: member})
}

func (h *CoopHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	coopID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid cooperative id")
		return
	}
	page := queryInt(r.URL.Query().Get("page"), 1)
	limit := queryInt(r.URL.Query().Get("limit"), 50)
	if limit > 200 {
		limit = 200
	}
	members, err := h.s.ListMembers(r.Context(), coopID, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: members})
}

func (h *CoopHandler) getMember(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["member_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid member id")
		return
	}
	m, err := h.s.GetMember(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "member not found")
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: m})
}

func (h *CoopHandler) updateMember(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["member_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid member id")
		return
	}
	var req models.UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	now := time.Now().UTC()
	if err := h.s.UpdateMember(r.Context(), id, req, now); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, id, claims.UserID, "update_member", "member")
	updated, _ := h.s.GetMember(r.Context(), id)
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: updated})
}

// ─── Selling pools ────────────────────────────────────────────────────────────

func (h *CoopHandler) createPool(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	coopID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid cooperative id")
		return
	}
	if _, err := h.s.GetCoop(r.Context(), coopID); err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "cooperative not found")
		return
	}
	var req models.CreatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.CropType == "" || req.TargetQtyKg <= 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "crop_type and target_quantity_kg are required")
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	now := time.Now().UTC()
	var openUntil *time.Time
	if req.OpenUntil != nil {
		if t, err := time.Parse(time.RFC3339, *req.OpenUntil); err == nil {
			tUTC := t.UTC()
			openUntil = &tUTC
		}
	}
	pool := models.SellingPool{
		ID:             uuid.New(),
		CoopID:         coopID,
		CropType:       req.CropType,
		TargetQtyKg:    req.TargetQtyKg,
		CollectedQtyKg: 0,
		PricePerKg:     req.PricePerKg,
		Currency:       req.Currency,
		Status:         models.PoolOpen,
		OpenUntil:      openUntil,
		TotalRevenue:   0,
		Description:    req.Description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := h.s.CreatePool(r.Context(), pool); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, pool.ID, claims.UserID, "create_pool", "selling_pool")
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: pool})
}

func (h *CoopHandler) listPools(w http.ResponseWriter, r *http.Request) {
	coopID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid cooperative id")
		return
	}
	page := queryInt(r.URL.Query().Get("page"), 1)
	limit := queryInt(r.URL.Query().Get("limit"), 20)
	pools, err := h.s.ListPools(r.Context(), coopID, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: pools})
}

func (h *CoopHandler) getPool(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["pool_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid pool id")
		return
	}
	pool, err := h.s.GetPool(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "pool not found")
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: pool})
}

func (h *CoopHandler) contribute(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	poolID, err := uuid.Parse(mux.Vars(r)["pool_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid pool id")
		return
	}
	pool, err := h.s.GetPool(r.Context(), poolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "pool not found")
		return
	}
	if pool.Status != models.PoolOpen {
		writeError(w, http.StatusConflict, "POOL_NOT_OPEN", "pool is not accepting contributions")
		return
	}
	var req models.ContributeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.QuantityKg <= 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "quantity_kg must be positive")
		return
	}
	farmerID := uuid.New()
	if req.FarmerID != "" {
		if fid, err := uuid.Parse(req.FarmerID); err == nil {
			farmerID = fid
		}
	}
	now := time.Now().UTC()
	contribution := models.PoolContribution{
		ID:           uuid.New(),
		PoolID:       poolID,
		FarmerID:     farmerID,
		QuantityKg:   req.QuantityKg,
		PayoutAmount: 0,
		PayoutPaid:   false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.s.AddContribution(r.Context(), contribution); err != nil {
		writeError(w, http.StatusConflict, "ALREADY_CONTRIBUTED", "farmer already contributed to this pool")
		return
	}
	h.s.AddPoolQuantity(r.Context(), poolID, req.QuantityKg, now)
	h.audit(r, contribution.ID, claims.UserID, "contribute", "pool_contribution")
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: contribution})
}

func (h *CoopHandler) listContributions(w http.ResponseWriter, r *http.Request) {
	poolID, err := uuid.Parse(mux.Vars(r)["pool_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid pool id")
		return
	}
	contributions, err := h.s.ListContributions(r.Context(), poolID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: contributions})
}

func (h *CoopHandler) closePool(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	poolID, err := uuid.Parse(mux.Vars(r)["pool_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid pool id")
		return
	}
	pool, err := h.s.GetPool(r.Context(), poolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "pool not found")
		return
	}
	if pool.Status != models.PoolOpen {
		writeError(w, http.StatusConflict, "POOL_NOT_OPEN", "pool is not open")
		return
	}
	now := time.Now().UTC()
	if err := h.s.ClosePool(r.Context(), poolID, now); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, poolID, claims.UserID, "close_pool", "selling_pool")
	updated, _ := h.s.GetPool(r.Context(), poolID)
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: updated})
}

func (h *CoopHandler) recordSale(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	poolID, err := uuid.Parse(mux.Vars(r)["pool_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid pool id")
		return
	}
	pool, err := h.s.GetPool(r.Context(), poolID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "pool not found")
		return
	}
	if pool.Status != models.PoolClosed && pool.Status != models.PoolOpen {
		writeError(w, http.StatusConflict, "INVALID_STATUS", "pool must be open or closed to record a sale")
		return
	}
	var req models.RecordSaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.TotalRevenue <= 0 || req.PricePerKg <= 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "price_per_kg and total_revenue are required")
		return
	}
	now := time.Now().UTC()
	if err := h.s.RecordSale(r.Context(), poolID, req.PricePerKg, req.TotalRevenue, now); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.s.DistributePayouts(r.Context(), poolID, req.TotalRevenue, now)
	h.audit(r, poolID, claims.UserID, "record_sale", "selling_pool")
	updated, _ := h.s.GetPool(r.Context(), poolID)
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: updated})
}

func (h *CoopHandler) markPaid(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["contribution_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid contribution id")
		return
	}
	now := time.Now().UTC()
	if err := h.s.MarkPayoutPaid(r.Context(), id, now); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, id, claims.UserID, "mark_payout_paid", "pool_contribution")
	updated, _ := h.s.GetContribution(r.Context(), id)
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: updated})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (h *CoopHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid, _ := uuid.Parse(userID)
	eid := entityID
	log := models.CoopAuditLog{
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
