package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/market-service/db"
	"github.com/klinova/kinara-os/market-service/middleware"
	"github.com/klinova/kinara-os/market-service/models"
)

// Store abstracts the database layer for testability.
type Store interface {
	CreateListing(ctx context.Context, l models.MarketListing) error
	GetListing(ctx context.Context, id uuid.UUID) (*models.MarketListing, error)
	ListListings(ctx context.Context, p db.ListListingsParams) ([]models.MarketListing, error)
	CountListings(ctx context.Context, p db.ListListingsParams) (int, error)
	UpdateListing(ctx context.Context, id uuid.UUID, req models.UpdateListingRequest, now time.Time) error

	CreateBid(ctx context.Context, b models.MarketBid) error
	GetBid(ctx context.Context, id uuid.UUID) (*models.MarketBid, error)
	ListBidsForListing(ctx context.Context, listingID uuid.UUID) ([]models.MarketBid, error)
	UpdateBidStatus(ctx context.Context, id uuid.UUID, status models.BidStatus, now time.Time) error

	RecordPrice(ctx context.Context, r models.PriceRecord) error
	GetPriceSummary(ctx context.Context, cropType, market, country string, from, to time.Time) (models.PriceSummary, error)
	ListPriceHistory(ctx context.Context, cropType, country string, days int) ([]models.PriceRecord, error)

	InsertAuditLog(ctx context.Context, log models.MarketAuditLog) error
}

type MarketHandler struct {
	s Store
}

func NewMarketHandler(q *db.Queries) *MarketHandler        { return &MarketHandler{s: q} }
func NewMarketHandlerWithStore(s Store) *MarketHandler      { return &MarketHandler{s: s} }

func (h *MarketHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/listings", h.createListing).Methods(http.MethodPost)
	r.HandleFunc("/listings", h.listListings).Methods(http.MethodGet)
	r.HandleFunc("/listings/{id}", h.getListing).Methods(http.MethodGet)
	r.HandleFunc("/listings/{id}", h.updateListing).Methods(http.MethodPut)
	r.HandleFunc("/listings/{id}/bids", h.placeBid).Methods(http.MethodPost)
	r.HandleFunc("/listings/{id}/bids", h.listBids).Methods(http.MethodGet)
	r.HandleFunc("/bids/{id}", h.getBid).Methods(http.MethodGet)
	r.HandleFunc("/bids/{id}/respond", h.respondBid).Methods(http.MethodPut)
	r.HandleFunc("/prices", h.recordPrice).Methods(http.MethodPost)
	r.HandleFunc("/prices/summary", h.priceSummary).Methods(http.MethodGet)
	r.HandleFunc("/prices/history", h.priceHistory).Methods(http.MethodGet)
}

// ─── Listings ─────────────────────────────────────────────────────────────────

func (h *MarketHandler) createListing(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	var req models.CreateListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.CropType == "" || req.QuantityKg <= 0 || req.PricePerUnit <= 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "crop_type, quantity_kg, and price_per_unit are required")
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.PriceUnit == "" {
		req.PriceUnit = models.UnitKg
	}
	if req.QualityGrade == "" {
		req.QualityGrade = "B"
	}
	now := time.Now().UTC()
	availFrom := now
	if req.AvailableFrom != "" {
		if t, err := time.Parse(time.RFC3339, req.AvailableFrom); err == nil {
			availFrom = t.UTC()
		}
	}
	var harvestedAt *time.Time
	if req.HarvestedAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.HarvestedAt); err == nil {
			tUTC := t.UTC()
			harvestedAt = &tUTC
		}
	}
	var availUntil *time.Time
	if req.AvailableUntil != nil {
		if t, err := time.Parse(time.RFC3339, *req.AvailableUntil); err == nil {
			tUTC := t.UTC()
			availUntil = &tUTC
		}
	}
	farmerID, _ := uuid.Parse(claims.UserID)
	listing := models.MarketListing{
		ID:             uuid.New(),
		FarmerID:       farmerID,
		CropType:       req.CropType,
		Variety:        req.Variety,
		QuantityKg:     req.QuantityKg,
		QuantityAvail:  req.QuantityKg,
		PricePerUnit:   req.PricePerUnit,
		Currency:       req.Currency,
		PriceUnit:      req.PriceUnit,
		QualityGrade:   req.QualityGrade,
		Country:        claims.FacilityID,
		Region:         req.Region,
		Market:         req.Market,
		HarvestedAt:    harvestedAt,
		AvailableFrom:  availFrom,
		AvailableUntil: availUntil,
		Status:         models.ListingActive,
		Description:    req.Description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := h.s.CreateListing(r.Context(), listing); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, listing.ID, claims.UserID, "create_listing", "listing")
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: listing})
}

func (h *MarketHandler) listListings(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := queryInt(q.Get("page"), 1)
	limit := queryInt(q.Get("limit"), 20)
	if limit > 100 {
		limit = 100
	}
	p := db.ListListingsParams{Page: page, Limit: limit}
	if v := q.Get("crop_type"); v != "" {
		p.CropType = &v
	}
	if v := q.Get("country"); v != "" {
		p.Country = &v
	}
	if v := q.Get("region"); v != "" {
		p.Region = &v
	}
	if v := q.Get("max_price"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			p.MaxPrice = &f
		}
	}
	if v := q.Get("min_quantity"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			p.MinQuantity = &f
		}
	}
	listings, err := h.s.ListListings(r.Context(), p)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	total, _ := h.s.CountListings(r.Context(), p)
	tp := total / limit
	if total%limit != 0 {
		tp++
	}
	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    listings,
		Meta:    &models.PageMeta{Page: page, Limit: limit, Total: total, TotalPages: tp},
	})
}

func (h *MarketHandler) getListing(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid listing id")
		return
	}
	l, err := h.s.GetListing(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "listing not found")
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: l})
}

func (h *MarketHandler) updateListing(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid listing id")
		return
	}
	existing, err := h.s.GetListing(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "listing not found")
		return
	}
	farmerID, _ := uuid.Parse(claims.UserID)
	if existing.FarmerID != farmerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "not your listing")
		return
	}
	var req models.UpdateListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	now := time.Now().UTC()
	if err := h.s.UpdateListing(r.Context(), id, req, now); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, id, claims.UserID, "update_listing", "listing")
	updated, _ := h.s.GetListing(r.Context(), id)
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: updated})
}

// ─── Bids ─────────────────────────────────────────────────────────────────────

func (h *MarketHandler) placeBid(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	listingID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid listing id")
		return
	}
	listing, err := h.s.GetListing(r.Context(), listingID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "listing not found")
		return
	}
	if listing.Status != models.ListingActive {
		writeError(w, http.StatusConflict, "LISTING_UNAVAILABLE", "listing is not active")
		return
	}
	var req models.PlaceBidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.QuantityKg <= 0 || req.BidPrice <= 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "quantity_kg and bid_price are required")
		return
	}
	if req.QuantityKg > listing.QuantityAvail {
		writeError(w, http.StatusBadRequest, "INSUFFICIENT_QUANTITY", "bid quantity exceeds available stock")
		return
	}
	if req.Currency == "" {
		req.Currency = listing.Currency
	}
	now := time.Now().UTC()
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			tUTC := t.UTC()
			expiresAt = &tUTC
		}
	}
	buyerID, _ := uuid.Parse(claims.UserID)
	bid := models.MarketBid{
		ID:         uuid.New(),
		ListingID:  listingID,
		BuyerID:    buyerID,
		QuantityKg: req.QuantityKg,
		BidPrice:   req.BidPrice,
		Currency:   req.Currency,
		Status:     models.BidPending,
		Message:    req.Message,
		ExpiresAt:  expiresAt,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := h.s.CreateBid(r.Context(), bid); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	h.audit(r, bid.ID, claims.UserID, "place_bid", "bid")
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: bid})
}

func (h *MarketHandler) listBids(w http.ResponseWriter, r *http.Request) {
	listingID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid listing id")
		return
	}
	bids, err := h.s.ListBidsForListing(r.Context(), listingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: bids})
}

func (h *MarketHandler) getBid(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid bid id")
		return
	}
	bid, err := h.s.GetBid(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "bid not found")
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: bid})
}

func (h *MarketHandler) respondBid(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid bid id")
		return
	}
	bid, err := h.s.GetBid(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "bid not found")
		return
	}
	if bid.Status != models.BidPending {
		writeError(w, http.StatusConflict, "BID_ALREADY_RESOLVED", "bid is not pending")
		return
	}
	listing, err := h.s.GetListing(r.Context(), bid.ListingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	farmerID, _ := uuid.Parse(claims.UserID)
	if listing.FarmerID != farmerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "not your listing")
		return
	}
	var req models.RespondBidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.Status != models.BidAccepted && req.Status != models.BidRejected {
		writeError(w, http.StatusBadRequest, "INVALID_STATUS", "status must be accepted or rejected")
		return
	}
	now := time.Now().UTC()
	if err := h.s.UpdateBidStatus(r.Context(), id, req.Status, now); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if req.Status == models.BidAccepted {
		newQty := listing.QuantityAvail - bid.QuantityKg
		newStatus := models.ListingActive
		if newQty <= 0 {
			newStatus = models.ListingSold
		}
		h.s.UpdateListing(r.Context(), listing.ID, models.UpdateListingRequest{
			QuantityAvail: &newQty,
			Status:        &newStatus,
		}, now)
	}
	h.audit(r, id, claims.UserID, "respond_bid:"+string(req.Status), "bid")
	updated, _ := h.s.GetBid(r.Context(), id)
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: updated})
}

// ─── Price tracking ───────────────────────────────────────────────────────────

func (h *MarketHandler) recordPrice(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	var req models.RecordPriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if req.CropType == "" || req.PricePerKg <= 0 || req.Country == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "crop_type, price_per_kg, and country are required")
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.Source == "" {
		req.Source = "reported"
	}
	recorderID, _ := uuid.Parse(claims.UserID)
	record := models.PriceRecord{
		ID:         uuid.New(),
		CropType:   req.CropType,
		Market:     req.Market,
		Country:    req.Country,
		Region:     req.Region,
		PricePerKg: req.PricePerKg,
		Currency:   req.Currency,
		Source:     req.Source,
		RecordedAt: time.Now().UTC(),
		RecordedBy: recorderID,
	}
	if err := h.s.RecordPrice(r.Context(), record); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, models.APIResponse{Success: true, Data: record})
}

func (h *MarketHandler) priceSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	cropType := q.Get("crop_type")
	if cropType == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "crop_type is required")
		return
	}
	market := q.Get("market")
	country := q.Get("country")
	days := queryInt(q.Get("days"), 30)
	to := time.Now().UTC()
	from := to.Add(-time.Duration(days) * 24 * time.Hour)
	summary, err := h.s.GetPriceSummary(r.Context(), cropType, market, country, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: summary})
}

func (h *MarketHandler) priceHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	cropType := q.Get("crop_type")
	if cropType == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "crop_type is required")
		return
	}
	country := q.Get("country")
	days := queryInt(q.Get("days"), 30)
	records, err := h.s.ListPriceHistory(r.Context(), cropType, country, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: records})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (h *MarketHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid, _ := uuid.Parse(userID)
	eid := entityID
	log := models.MarketAuditLog{
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
