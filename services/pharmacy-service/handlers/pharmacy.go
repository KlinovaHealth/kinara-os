package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/pharmacy-service/auth"
	"github.com/klinova/kinara-os/pharmacy-service/crypto"
	"github.com/klinova/kinara-os/pharmacy-service/db"
	"github.com/klinova/kinara-os/pharmacy-service/middleware"
	"github.com/klinova/kinara-os/pharmacy-service/models"
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
	api.Use(pkgauth.RequireTenantScope("pharmacy-service", nil))

	// Prescriptions
	api.HandleFunc("/prescriptions", h.registerPrescription).Methods(http.MethodPost)
	api.HandleFunc("/prescriptions", h.listPrescriptions).Methods(http.MethodGet)
	api.HandleFunc("/prescriptions/{id}", h.getPrescription).Methods(http.MethodGet)
	api.HandleFunc("/prescriptions/{id}/dispense", h.dispensePrescription).Methods(http.MethodPost)
	api.HandleFunc("/prescriptions/{id}/dispensing", h.listDispensing).Methods(http.MethodGet)

	// Inventory
	api.HandleFunc("/inventory", h.listInventory).Methods(http.MethodGet)
	api.HandleFunc("/inventory", h.createMedication).Methods(http.MethodPost)
	api.HandleFunc("/inventory/{med_id}", h.getMedication).Methods(http.MethodGet)
	api.HandleFunc("/inventory/{med_id}", h.updateStock).Methods(http.MethodPut)
	api.HandleFunc("/inventory/alerts", h.stockAlerts).Methods(http.MethodGet)

	// Supply orders
	api.HandleFunc("/orders", h.createOrder).Methods(http.MethodPost)
	api.HandleFunc("/orders", h.listOrders).Methods(http.MethodGet)
	api.HandleFunc("/orders/{id}", h.getOrder).Methods(http.MethodGet)

	// Cost summary
	api.HandleFunc("/costs", h.costSummary).Methods(http.MethodGet)
}

// ─── Register prescription ────────────────────────────────────────────────────

func (h *Handler) registerPrescription(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "doctor", "nurse") {
		h.forbidden(w)
		return
	}

	var req models.RegisterPrescriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if req.ClinicalID == "" || req.PatientID == "" || req.MedicationID == "" || req.Quantity <= 0 {
		h.badRequest(w, "clinical_id, patient_id, medication_id, and quantity > 0 are required")
		return
	}

	clinicalID, err := uuid.Parse(req.ClinicalID)
	if err != nil {
		h.badRequest(w, "invalid clinical_id")
		return
	}
	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		h.badRequest(w, "invalid patient_id")
		return
	}
	clinicID, err := uuid.Parse(req.ClinicID)
	if err != nil {
		h.badRequest(w, "invalid clinic_id")
		return
	}
	medicationID, err := uuid.Parse(req.MedicationID)
	if err != nil {
		h.badRequest(w, "invalid medication_id")
		return
	}

	var expiresAt time.Time
	if req.ExpiresAt != "" {
		expiresAt, err = time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			h.badRequest(w, "expires_at must be RFC3339")
			return
		}
	} else {
		expiresAt = time.Now().UTC().Add(30 * 24 * time.Hour)
	}

	patientNameEnc, err := h.enc.EncryptString(req.PatientName)
	if err != nil {
		h.internalError(w, "encryption error")
		return
	}
	dosageEnc, err := h.enc.EncryptString(req.Dosage)
	if err != nil {
		h.internalError(w, "encryption error")
		return
	}

	now := time.Now().UTC()
	unit := req.QuantityUnit
	if unit == "" {
		unit = "tablet"
	}
	row := models.PrescriptionRow{
		ID:             uuid.New(),
		ClinicalID:     clinicalID,
		PatientID:      patientID,
		ClinicID:       clinicID,
		MedicationID:   medicationID,
		PatientNameEnc: patientNameEnc,
		DosageEnc:      dosageEnc,
		Quantity:       req.Quantity,
		QuantityUnit:   unit,
		Instructions:   req.Instructions,
		Status:         models.PrescriptionPending,
		IssuedAt:       now,
		ExpiresAt:      expiresAt.UTC(),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := h.queries.CreatePrescription(r.Context(), row); err != nil {
		h.logger.Error("create prescription failed", "error", err)
		h.internalError(w, "failed to create prescription")
		return
	}

	h.audit(r, claims, &row.ID, "register_prescription", "prescriptions")
	pres := h.decryptPrescription(row)
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: pres})
}

// ─── Get / List prescriptions ─────────────────────────────────────────────────

func (h *Handler) getPrescription(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "doctor", "nurse", "clinician") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid id")
		return
	}
	row, err := h.queries.GetPrescription(r.Context(), id)
	if err != nil {
		h.notFound(w, "prescription not found")
		return
	}
	h.audit(r, claims, &id, "get_prescription", "prescriptions")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: h.decryptPrescription(*row)})
}

func (h *Handler) listPrescriptions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "doctor", "nurse", "clinician", "facility_admin") {
		h.forbidden(w)
		return
	}

	params := db.ListPrescriptionsParams{
		Page:  pageParam(r, 1),
		Limit: limitParam(r, 50),
	}
	if cid := r.URL.Query().Get("clinic_id"); cid != "" {
		id, err := uuid.Parse(cid)
		if err != nil {
			h.badRequest(w, "invalid clinic_id")
			return
		}
		params.ClinicID = &id
	}
	if pid := r.URL.Query().Get("patient_id"); pid != "" {
		id, err := uuid.Parse(pid)
		if err != nil {
			h.badRequest(w, "invalid patient_id")
			return
		}
		params.PatientID = &id
	}
	if s := r.URL.Query().Get("status"); s != "" {
		st := models.PrescriptionStatus(s)
		params.Status = &st
	}

	rows, err := h.queries.ListPrescriptions(r.Context(), params)
	if err != nil {
		h.internalError(w, "failed to list prescriptions")
		return
	}
	total, _ := h.queries.CountPrescriptions(r.Context(), params)

	pres := make([]models.Prescription, 0, len(rows))
	for _, row := range rows {
		pres = append(pres, h.decryptPrescription(row))
	}
	totalPages := (total + params.Limit - 1) / params.Limit
	h.audit(r, claims, nil, "list_prescriptions", "prescriptions")
	h.json(w, http.StatusOK, models.APIResponse{
		Success: true, Data: pres,
		Meta: &models.PageMeta{Page: params.Page, Limit: params.Limit, Total: total, TotalPages: totalPages},
	})
}

// ─── Dispense prescription ────────────────────────────────────────────────────

func (h *Handler) dispensePrescription(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist") {
		h.forbidden(w)
		return
	}

	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid prescription id")
		return
	}

	pres, err := h.queries.GetPrescription(r.Context(), id)
	if err != nil {
		h.notFound(w, "prescription not found")
		return
	}
	if pres.Status == models.PrescriptionDispensed || pres.Status == models.PrescriptionCancelled || pres.Status == models.PrescriptionExpired {
		h.badRequest(w, "prescription cannot be dispensed in status: "+string(pres.Status))
		return
	}
	if time.Now().UTC().After(pres.ExpiresAt) {
		h.queries.UpdatePrescriptionStatus(r.Context(), id, models.PrescriptionExpired, time.Now().UTC())
		h.badRequest(w, "prescription has expired")
		return
	}

	var req models.DispenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.QuantityDispensed <= 0 {
		h.badRequest(w, "quantity_dispensed > 0 is required")
		return
	}

	now := time.Now().UTC()
	if req.CostAmount < 0 || req.PatientCostShare < 0 {
		h.badRequest(w, "cost_amount and patient_cost_share must be >= 0")
		return
	}

	// Decrement medication stock atomically
	if err := h.queries.DecrementStock(r.Context(), pres.MedicationID, req.QuantityDispensed, now); err != nil {
		h.logger.Error("decrement stock failed", "error", err)
		h.badRequest(w, "insufficient stock or medication not found")
		return
	}

	drow := models.DispensingRow{
		ID:                uuid.New(),
		PrescriptionID:    id,
		MedicationID:      pres.MedicationID,
		DispensedByUserID: claims.UserID,
		QuantityDispensed: req.QuantityDispensed,
		BatchNumber:       req.BatchNumber,
		CostAmount:        req.CostAmount,
		Currency:          "USD",
		PatientCostShare:  req.PatientCostShare,
		Notes:             req.Notes,
		DispensedAt:       now,
	}
	if err := h.queries.CreateDispensing(r.Context(), drow); err != nil {
		h.logger.Error("create dispensing failed", "error", err)
		h.internalError(w, "failed to record dispensing")
		return
	}

	newStatus := models.PrescriptionDispensed
	if req.QuantityDispensed < pres.Quantity {
		newStatus = models.PrescriptionPartial
	}
	h.queries.UpdatePrescriptionStatus(r.Context(), id, newStatus, now)

	h.audit(r, claims, &id, "dispense_prescription", "dispensing")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: drow})
}

func (h *Handler) listDispensing(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "doctor", "facility_admin") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid id")
		return
	}
	rows, err := h.queries.ListDispensingForPrescription(r.Context(), id)
	if err != nil {
		h.internalError(w, "failed to list dispensing records")
		return
	}
	h.audit(r, claims, &id, "list_dispensing", "dispensing")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: rows})
}

// ─── Inventory ────────────────────────────────────────────────────────────────

func (h *Handler) createMedication(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "facility_admin") {
		h.forbidden(w)
		return
	}

	var req models.CreateMedicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if req.Name == "" || req.Unit == "" {
		h.badRequest(w, "name and unit are required")
		return
	}

	now := time.Now().UTC()
	row := models.MedicationRow{
		ID:           uuid.New(),
		Name:         req.Name,
		GenericName:  req.GenericName,
		Description:  req.Description,
		UnitPrice:    req.UnitPrice,
		Currency:     req.Currency,
		StockLevel:   req.StockLevel,
		ReorderPoint: req.ReorderPoint,
		ReorderQty:   req.ReorderQty,
		Unit:         req.Unit,
		BatchNumber:  req.BatchNumber,
		RequiresCold: req.RequiresCold,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if req.SupplierID != nil {
		sid, err := uuid.Parse(*req.SupplierID)
		if err != nil {
			h.badRequest(w, "invalid supplier_id")
			return
		}
		row.SupplierID = &sid
	}
	if req.ExpirationDate != nil {
		exp, err := time.Parse(time.RFC3339, *req.ExpirationDate)
		if err != nil {
			h.badRequest(w, "expiration_date must be RFC3339")
			return
		}
		exp = exp.UTC()
		row.ExpirationDate = &exp
	}

	if err := h.queries.CreateMedication(r.Context(), row); err != nil {
		h.logger.Error("create medication failed", "error", err)
		h.internalError(w, "failed to create medication")
		return
	}
	h.audit(r, claims, &row.ID, "create_medication", "medications")
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: row})
}

func (h *Handler) listInventory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "doctor", "nurse", "facility_admin", "analyst") {
		h.forbidden(w)
		return
	}

	params := db.ListInventoryParams{
		LowStockOnly: r.URL.Query().Get("low_stock") == "true",
		Page:         pageParam(r, 1),
		Limit:        limitParam(r, 100),
	}
	rows, err := h.queries.ListInventory(r.Context(), params)
	if err != nil {
		h.internalError(w, "failed to list inventory")
		return
	}
	total, _ := h.queries.CountInventory(r.Context(), params)
	totalPages := (total + params.Limit - 1) / params.Limit
	h.audit(r, claims, nil, "list_inventory", "medications")
	h.json(w, http.StatusOK, models.APIResponse{
		Success: true, Data: rows,
		Meta: &models.PageMeta{Page: params.Page, Limit: params.Limit, Total: total, TotalPages: totalPages},
	})
}

func (h *Handler) getMedication(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "doctor", "nurse", "facility_admin") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "med_id")
	if err != nil {
		h.badRequest(w, "invalid medication id")
		return
	}
	row, err := h.queries.GetMedication(r.Context(), id)
	if err != nil {
		h.notFound(w, "medication not found")
		return
	}
	h.audit(r, claims, &id, "get_medication", "medications")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: row})
}

func (h *Handler) updateStock(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "facility_admin") {
		h.forbidden(w)
		return
	}

	id, err := parseID(r, "med_id")
	if err != nil {
		h.badRequest(w, "invalid medication id")
		return
	}

	var req models.UpdateStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if req.StockLevel != nil && *req.StockLevel < 0 {
		h.badRequest(w, "stock_level must be >= 0")
		return
	}

	now := time.Now().UTC()
	p := db.UpdateStockParams{ID: id, Now: now,
		StockLevel: req.StockLevel, ReorderPoint: req.ReorderPoint,
		ReorderQty: req.ReorderQty, UnitPrice: req.UnitPrice,
		BatchNumber: req.BatchNumber,
	}
	if req.ExpirationDate != nil {
		exp, err := time.Parse(time.RFC3339, *req.ExpirationDate)
		if err != nil {
			h.badRequest(w, "expiration_date must be RFC3339")
			return
		}
		exp = exp.UTC()
		p.ExpirationDate = &exp
	}

	if err := h.queries.UpdateStock(r.Context(), p); err != nil {
		h.logger.Error("update stock failed", "error", err)
		h.internalError(w, "failed to update stock")
		return
	}

	row, _ := h.queries.GetMedication(r.Context(), id)
	h.audit(r, claims, &id, "update_stock", "medications")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: row})
}

func (h *Handler) stockAlerts(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "facility_admin", "analyst") {
		h.forbidden(w)
		return
	}

	meds, err := h.queries.GetStockAlerts(r.Context())
	if err != nil {
		h.internalError(w, "failed to get stock alerts")
		return
	}

	now := time.Now().UTC()
	alerts := make([]models.StockAlert, 0)
	for _, m := range meds {
		alert := models.StockAlert{
			MedicationID:   m.ID,
			MedicationName: m.Name,
			StockLevel:     m.StockLevel,
			ReorderPoint:   m.ReorderPoint,
		}
		if m.ExpirationDate != nil && m.ExpirationDate.Before(now) {
			alert.AlertType = "expired"
			alert.Message = m.Name + " has expired"
		} else if m.ExpirationDate != nil && m.ExpirationDate.Before(now.Add(30*24*time.Hour)) {
			alert.AlertType = "expiring_soon"
			alert.Message = m.Name + " expires within 30 days"
		} else {
			alert.AlertType = "low_stock"
			alert.Message = m.Name + ": stock (" + strconv.Itoa(m.StockLevel) + ") at or below reorder point (" + strconv.Itoa(m.ReorderPoint) + ")"
		}
		alerts = append(alerts, alert)
	}

	h.audit(r, claims, nil, "get_stock_alerts", "medications")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: alerts})
}

// ─── Supply orders ────────────────────────────────────────────────────────────

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "facility_admin") {
		h.forbidden(w)
		return
	}

	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON")
		return
	}
	if req.SupplierID == "" || req.MedicationID == "" || req.QuantityOrdered <= 0 {
		h.badRequest(w, "supplier_id, medication_id, and quantity_ordered > 0 are required")
		return
	}

	supplierID, err := uuid.Parse(req.SupplierID)
	if err != nil {
		h.badRequest(w, "invalid supplier_id")
		return
	}
	medicationID, err := uuid.Parse(req.MedicationID)
	if err != nil {
		h.badRequest(w, "invalid medication_id")
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	now := time.Now().UTC()
	row := models.SupplyOrderRow{
		ID:              uuid.New(),
		SupplierID:      supplierID,
		MedicationID:    medicationID,
		QuantityOrdered: req.QuantityOrdered,
		UnitCost:        req.UnitCost,
		Currency:        currency,
		Status:          models.OrderPending,
		OrderedByID:     claims.UserID,
		Notes:           req.Notes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if req.ExpectedAt != nil {
		exp, err := time.Parse(time.RFC3339, *req.ExpectedAt)
		if err != nil {
			h.badRequest(w, "expected_at must be RFC3339")
			return
		}
		exp = exp.UTC()
		row.ExpectedAt = &exp
	}

	if err := h.queries.CreateOrder(r.Context(), row); err != nil {
		h.logger.Error("create order failed", "error", err)
		h.internalError(w, "failed to create order")
		return
	}
	h.audit(r, claims, &row.ID, "create_order", "orders")
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: row})
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "facility_admin", "analyst") {
		h.forbidden(w)
		return
	}
	page := pageParam(r, 1)
	limit := limitParam(r, 50)
	rows, err := h.queries.ListOrders(r.Context(), page, limit)
	if err != nil {
		h.internalError(w, "failed to list orders")
		return
	}
	h.audit(r, claims, nil, "list_orders", "orders")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: rows})
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "pharmacist", "facility_admin") {
		h.forbidden(w)
		return
	}
	id, err := parseID(r, "id")
	if err != nil {
		h.badRequest(w, "invalid order id")
		return
	}
	row, err := h.queries.GetOrder(r.Context(), id)
	if err != nil {
		h.notFound(w, "order not found")
		return
	}
	h.audit(r, claims, &id, "get_order", "orders")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: row})
}

// ─── Cost summary ─────────────────────────────────────────────────────────────

func (h *Handler) costSummary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !roleAllowed(claims, "admin", "facility_admin", "analyst", "government", "ministry_official") {
		h.forbidden(w)
		return
	}

	clinicIDStr := r.URL.Query().Get("clinic_id")
	if clinicIDStr == "" {
		h.badRequest(w, "clinic_id is required")
		return
	}
	clinicID, err := uuid.Parse(clinicIDStr)
	if err != nil {
		h.badRequest(w, "invalid clinic_id")
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	from := time.Now().UTC().Add(-30 * 24 * time.Hour)
	to := time.Now().UTC()
	if fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			h.badRequest(w, "from must be RFC3339")
			return
		}
	}
	if toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			h.badRequest(w, "to must be RFC3339")
			return
		}
	}

	summary, err := h.queries.GetCostSummary(r.Context(), clinicID, from, to)
	if err != nil {
		h.internalError(w, "failed to get cost summary")
		return
	}
	summary.Period = from.Format("2006-01-02") + " to " + to.Format("2006-01-02")

	h.audit(r, claims, nil, "get_cost_summary", "costs")
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: summary})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (h *Handler) decryptPrescription(row models.PrescriptionRow) models.Prescription {
	patientName, _ := h.enc.DecryptString(row.PatientNameEnc)
	dosage, _ := h.enc.DecryptString(row.DosageEnc)
	return models.Prescription{
		ID:           row.ID,
		ClinicalID:   row.ClinicalID,
		PatientID:    row.PatientID,
		ClinicID:     row.ClinicID,
		MedicationID: row.MedicationID,
		PatientName:  patientName,
		Dosage:       dosage,
		Quantity:     row.Quantity,
		QuantityUnit: row.QuantityUnit,
		Instructions: row.Instructions,
		Status:       row.Status,
		IssuedAt:     row.IssuedAt,
		ExpiresAt:    row.ExpiresAt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func (h *Handler) audit(r *http.Request, claims *auth.Claims, entityID *uuid.UUID, action, resource string) {
	h.queries.InsertAuditLog(r.Context(), models.PharmacyAuditLog{
		ID:        uuid.New(),
		EntityID:  entityID,
		UserID:    claims.UserID,
		Action:    action,
		Resource:  resource,
		IPAddress: remoteIP(r),
		CreatedAt: time.Now().UTC(),
	})
}

func (h *Handler) json(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) badRequest(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusBadRequest, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "BAD_REQUEST", Message: msg},
	})
}
func (h *Handler) forbidden(w http.ResponseWriter) {
	h.json(w, http.StatusForbidden, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "FORBIDDEN", Message: "insufficient role"},
	})
}
func (h *Handler) notFound(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusNotFound, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "NOT_FOUND", Message: msg},
	})
}
func (h *Handler) internalError(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusInternalServerError, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "INTERNAL_ERROR", Message: msg},
	})
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
