package handlers

import (
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/farmer-service/auth"
	"github.com/klinova/kinara-os/farmer-service/crypto"
	"github.com/klinova/kinara-os/farmer-service/db"
	"github.com/klinova/kinara-os/farmer-service/middleware"
	"github.com/klinova/kinara-os/farmer-service/models"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
)

type Handler struct {
	queries *db.Queries
	enc     *crypto.Encryptor
	logger  *slog.Logger
}

func New(queries *db.Queries, enc *crypto.Encryptor, logger *slog.Logger) *Handler {
	return &Handler{queries: queries, enc: enc, logger: logger}
}

func (h *Handler) Register(r *mux.Router, jwtMW func(http.Handler) http.Handler) {
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(jwtMW)
	api.Use(pkgauth.RequireTenantScope("farmer-service", nil))

	api.HandleFunc("/farmers", h.registerFarmer).Methods(http.MethodPost)
	api.HandleFunc("/farmers", h.listFarmers).Methods(http.MethodGet)
	api.HandleFunc("/farmers/{id}", h.getFarmer).Methods(http.MethodGet)
	api.HandleFunc("/farmers/{id}", h.updateFarmer).Methods(http.MethodPut)
	api.HandleFunc("/farmers/{id}/verify", h.verifyFarmer).Methods(http.MethodPost)

	api.HandleFunc("/farmers/{id}/plots", h.addPlot).Methods(http.MethodPost)
	api.HandleFunc("/farmers/{id}/plots", h.listPlots).Methods(http.MethodGet)

	api.HandleFunc("/farmers/{id}/crops", h.recordCrop).Methods(http.MethodPost)
	api.HandleFunc("/farmers/{id}/crops", h.listCrops).Methods(http.MethodGet)
	api.HandleFunc("/crops/{crop_id}", h.updateCrop).Methods(http.MethodPut)
}

// ─── Register farmer ──────────────────────────────────────────────────────────

func (h *Handler) registerFarmer(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "farmer", "cooperative_manager", "facility_admin", "analyst") {
		h.forbidden(w)
		return
	}

	var req models.RegisterFarmerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if req.FullName == "" || req.Phone == "" || req.Country == "" {
		h.badRequest(w, "full_name, phone, and country are required")
		return
	}

	fullNameEnc, err := h.enc.EncryptString(req.FullName)
	if err != nil {
		h.internalError(w, "encryption error")
		return
	}
	phoneEnc, err := h.enc.EncryptString(req.Phone)
	if err != nil {
		h.internalError(w, "encryption error")
		return
	}
	nationalIDEnc := ""
	if req.NationalID != "" {
		nationalIDEnc, err = h.enc.EncryptString(req.NationalID)
		if err != nil {
			h.internalError(w, "encryption error")
			return
		}
	}

	farmSize := farmSizeCategory(req.FarmSizeHa)
	lang := req.PrimaryLanguage
	if lang == "" {
		lang = "en"
	}

	now := time.Now().UTC()
	row := models.FarmerRow{
		ID:              uuid.New(),
		UserID:          &claims.UserID,
		FullNameEnc:     fullNameEnc,
		PhoneEnc:        phoneEnc,
		NationalIDEnc:   nationalIDEnc,
		Country:         req.Country,
		Region:          req.Region,
		District:        req.District,
		GPSLat:          req.GPSLat,
		GPSLng:          req.GPSLng,
		FarmSizeHa:      req.FarmSizeHa,
		FarmSize:        farmSize,
		PrimaryLanguage: lang,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if req.CooperativeID != nil {
		cid, err := uuid.Parse(*req.CooperativeID)
		if err != nil {
			h.badRequest(w, "invalid cooperative_id")
			return
		}
		row.CooperativeID = &cid
	}

	if err := h.queries.CreateFarmer(r.Context(), row); err != nil {
		h.logger.Error("create farmer failed", "error", err)
		h.internalError(w, "failed to register farmer")
		return
	}

	h.audit(r, claims, &row.ID, "register_farmer", "farmers")
	farmer := h.decryptFarmer(row)
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: farmer})
}

// ─── Get / List farmers ───────────────────────────────────────────────────────

func (h *Handler) getFarmer(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "farmer", "cooperative_manager", "facility_admin", "analyst", "government") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid id")
		return
	}
	row, err := h.queries.GetFarmer(r.Context(), id)
	if err != nil {
		h.notFound(w, "farmer not found")
		return
	}
	h.audit(r, claims, &id, "get_farmer", "farmers")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: h.decryptFarmer(*row)})
}

func (h *Handler) listFarmers(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "cooperative_manager", "facility_admin", "analyst", "government", "ministry_official") {
		h.forbidden(w)
		return
	}

	params := db.ListFarmersParams{
		Page:  pageParam(r, 1),
		Limit: limitParam(r, 50),
	}
	if c := r.URL.Query().Get("country"); c != "" {
		params.Country = &c
	}
	if rg := r.URL.Query().Get("region"); rg != "" {
		params.Region = &rg
	}
	if cid := r.URL.Query().Get("cooperative_id"); cid != "" {
		id, err := uuid.Parse(cid)
		if err != nil {
			h.badRequest(w, "invalid cooperative_id")
			return
		}
		params.CooperativeID = &id
	}
	if a := r.URL.Query().Get("active"); a == "false" {
		f := false
		params.IsActive = &f
	} else {
		t := true
		params.IsActive = &t
	}

	rows, err := h.queries.ListFarmers(r.Context(), params)
	if err != nil {
		h.internalError(w, "failed to list farmers")
		return
	}
	total, _ := h.queries.CountFarmers(r.Context(), params)

	farmers := make([]models.Farmer, 0, len(rows))
	for _, row := range rows {
		f := h.decryptFarmer(row)
		f.NationalID = "" // never expose in list view
		farmers = append(farmers, f)
	}

	totalPages := (total + params.Limit - 1) / params.Limit
	h.audit(r, claims, nil, "list_farmers", "farmers")
	h.json(w, http.StatusOK, models.APIResponse{
		Success: true, Data: farmers,
		Meta: &models.PageMeta{Page: params.Page, Limit: params.Limit, Total: total, TotalPages: totalPages},
	})
}

// ─── Update farmer ────────────────────────────────────────────────────────────

func (h *Handler) updateFarmer(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "farmer", "cooperative_manager", "facility_admin") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid id")
		return
	}

	var req models.UpdateFarmerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}

	p := db.UpdateFarmerParams{
		ID:              id,
		Region:          req.Region,
		District:        req.District,
		GPSLat:          req.GPSLat,
		GPSLng:          req.GPSLng,
		FarmSizeHa:      req.FarmSizeHa,
		PrimaryLanguage: req.PrimaryLanguage,
		IsActive:        req.IsActive,
		Now:             time.Now().UTC(),
	}
	if req.Phone != nil {
		enc, err := h.enc.EncryptString(*req.Phone)
		if err != nil {
			h.internalError(w, "encryption error")
			return
		}
		p.PhoneEnc = &enc
	}
	if req.CooperativeID != nil {
		cid, err := uuid.Parse(*req.CooperativeID)
		if err != nil {
			h.badRequest(w, "invalid cooperative_id")
			return
		}
		p.CooperativeID = &cid
	}

	if err := h.queries.UpdateFarmer(r.Context(), p); err != nil {
		h.internalError(w, "failed to update farmer")
		return
	}

	row, _ := h.queries.GetFarmer(r.Context(), id)
	h.audit(r, claims, &id, "update_farmer", "farmers")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: h.decryptFarmer(*row)})
}

// ─── Verify farmer ────────────────────────────────────────────────────────────

func (h *Handler) verifyFarmer(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "facility_admin", "government") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid id")
		return
	}
	if err := h.queries.VerifyFarmer(r.Context(), id, time.Now().UTC()); err != nil {
		h.internalError(w, "failed to verify farmer")
		return
	}
	h.audit(r, claims, &id, "verify_farmer", "farmers")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]bool{"is_verified": true}})
}

// ─── Farm plots ───────────────────────────────────────────────────────────────

func (h *Handler) addPlot(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "farmer", "cooperative_manager") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid farmer id")
		return
	}

	var req models.AddPlotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if req.Name == "" || req.SizeHa <= 0 {
		h.badRequest(w, "name and size_ha > 0 are required")
		return
	}

	now := time.Now().UTC()
	plot := models.FarmPlot{
		ID:         uuid.New(),
		FarmerID:   id,
		Name:       req.Name,
		SizeHa:     req.SizeHa,
		SoilType:   req.SoilType,
		Irrigation: req.Irrigation,
		GPSPolygon: req.GPSPolygon,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := h.queries.CreatePlot(r.Context(), plot); err != nil {
		h.logger.Error("create plot failed", "error", err)
		h.internalError(w, "failed to create plot")
		return
	}
	h.audit(r, claims, &id, "add_plot", "farm_plots")
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: plot})
}

func (h *Handler) listPlots(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "farmer", "cooperative_manager", "analyst") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid farmer id")
		return
	}
	plots, err := h.queries.ListPlots(r.Context(), id)
	if err != nil {
		h.internalError(w, "failed to list plots")
		return
	}
	h.audit(r, claims, &id, "list_plots", "farm_plots")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: plots})
}

// ─── Crop records ─────────────────────────────────────────────────────────────

func (h *Handler) recordCrop(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "farmer", "cooperative_manager") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid farmer id")
		return
	}

	var req models.RecordCropRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if req.CropType == "" || req.AreaHa <= 0 || req.PlantedAt == "" || req.ExpectedHarvest == "" {
		h.badRequest(w, "crop_type, area_ha > 0, planted_at, and expected_harvest are required")
		return
	}

	plantedAt, err := time.Parse(time.RFC3339, req.PlantedAt)
	if err != nil {
		h.badRequest(w, "planted_at must be RFC3339")
		return
	}
	expectedHarvest, err := time.Parse(time.RFC3339, req.ExpectedHarvest)
	if err != nil {
		h.badRequest(w, "expected_harvest must be RFC3339")
		return
	}

	now := time.Now().UTC()
	crop := models.CropRecord{
		ID:              uuid.New(),
		FarmerID:        id,
		CropType:        req.CropType,
		Variety:         req.Variety,
		AreaHa:          req.AreaHa,
		PlantedAt:       plantedAt.UTC(),
		ExpectedHarvest: expectedHarvest.UTC(),
		Status:          models.CropPlanted,
		Notes:           req.Notes,
		Season:          req.Season,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if req.PlotID != nil {
		pid, err := uuid.Parse(*req.PlotID)
		if err != nil {
			h.badRequest(w, "invalid plot_id")
			return
		}
		crop.PlotID = &pid
	}

	if err := h.queries.CreateCropRecord(r.Context(), crop); err != nil {
		h.logger.Error("create crop record failed", "error", err)
		h.internalError(w, "failed to record crop")
		return
	}
	h.audit(r, claims, &id, "record_crop", "crop_records")
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: crop})
}

func (h *Handler) listCrops(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "farmer", "cooperative_manager", "analyst", "government") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid farmer id")
		return
	}
	crops, err := h.queries.ListCropRecords(r.Context(), id, pageParam(r, 1), limitParam(r, 50))
	if err != nil {
		h.internalError(w, "failed to list crops")
		return
	}
	h.audit(r, claims, &id, "list_crops", "crop_records")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: crops})
}

func (h *Handler) updateCrop(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "farmer", "cooperative_manager") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "crop_id")
	if err != nil {
		h.badRequest(w, "invalid crop id")
		return
	}

	var req models.UpdateCropRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}

	var actualHarvestTime *time.Time
	if req.ActualHarvest != nil {
		t, err := time.Parse(time.RFC3339, *req.ActualHarvest)
		if err != nil {
			h.badRequest(w, "actual_harvest must be RFC3339")
			return
		}
		tUTC := t.UTC()
		actualHarvestTime = &tUTC
		req.ActualHarvest = nil
	}
	_ = actualHarvestTime // stored via SQL COALESCE

	if err := h.queries.UpdateCropRecord(r.Context(), id, req, time.Now().UTC()); err != nil {
		h.internalError(w, "failed to update crop")
		return
	}
	h.audit(r, claims, nil, "update_crop", "crop_records")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: map[string]string{"status": string(req.Status)}})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (h *Handler) decryptFarmer(row models.FarmerRow) models.Farmer {
	fullName, _ := h.enc.DecryptString(row.FullNameEnc)
	phone, _ := h.enc.DecryptString(row.PhoneEnc)
	nationalID := ""
	if row.NationalIDEnc != "" {
		nationalID, _ = h.enc.DecryptString(row.NationalIDEnc)
	}
	return models.Farmer{
		ID:              row.ID,
		UserID:          row.UserID,
		FullName:        fullName,
		Phone:           phone,
		NationalID:      nationalID,
		Country:         row.Country,
		Region:          row.Region,
		District:        row.District,
		GPSLat:          row.GPSLat,
		GPSLng:          row.GPSLng,
		FarmSizeHa:      row.FarmSizeHa,
		FarmSize:        row.FarmSize,
		PrimaryLanguage: row.PrimaryLanguage,
		IsVerified:      row.IsVerified,
		IsActive:        row.IsActive,
		CooperativeID:   row.CooperativeID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func (h *Handler) audit(r *http.Request, claims *auth.Claims, farmerID *uuid.UUID, action, resource string) {
	h.queries.InsertAuditLog(r.Context(), models.FarmerAuditLog{
		ID:        uuid.New(),
		FarmerID:  farmerID,
		UserID:    claims.UserID,
		Action:    action,
		Resource:  resource,
		IPAddress: remoteIP(r),
		CreatedAt: time.Now().UTC(),
	})
}

func farmSizeCategory(ha float64) models.FarmSize {
	switch {
	case ha < 2:
		return models.FarmSmallholder
	case ha < 10:
		return models.FarmSmall
	case ha < 100:
		return models.FarmMedium
	default:
		return models.FarmLarge
	}
}

func (h *Handler) json(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func (h *Handler) badRequest(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: &models.APIError{Code: "BAD_REQUEST", Message: msg}})
}
func (h *Handler) forbidden(w http.ResponseWriter) {
	h.json(w, http.StatusForbidden, models.APIResponse{Success: false, Error: &models.APIError{Code: "FORBIDDEN", Message: "insufficient role"}})
}
func (h *Handler) notFound(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusNotFound, models.APIResponse{Success: false, Error: &models.APIError{Code: "NOT_FOUND", Message: msg}})
}
func (h *Handler) internalError(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusInternalServerError, models.APIResponse{Success: false, Error: &models.APIError{Code: "INTERNAL_ERROR", Message: msg}})
}

func roleAllowed(claims *auth.Claims, roles ...string) bool {
	if claims == nil {
		return false
	}
	for _, r := range roles {
		if claims.Role == r {
			return true
		}
	}
	return false
}

func parseID(r *http.Request, key string) (uuid.UUID, error) {
	return uuid.Parse(mux.Vars(r)[key])
}

func pageParam(r *http.Request, def int) int {
	p, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || p < 1 {
		return def
	}
	return p
}

func limitParam(r *http.Request, def int) int {
	l, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || l < 1 || l > 200 {
		return def
	}
	return l
}

func remoteIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	return r.RemoteAddr
}

// roundTo2 rounds a float to 2 decimal places (used in GPS display).
func roundTo2(f float64) float64 {
	return math.Round(f*100) / 100
}

var _ = roundTo2 // suppress unused warning — available for future GPS formatting
