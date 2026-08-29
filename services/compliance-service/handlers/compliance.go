package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/compliance-service/db"
	"github.com/klinova/kinara-os/compliance-service/middleware"
	"github.com/klinova/kinara-os/compliance-service/models"
)

type Store interface {
	CreatePermit(ctx context.Context, p models.TransitPermit) error
	GetPermit(ctx context.Context, id uuid.UUID) (*models.TransitPermit, error)
	ListPermits(ctx context.Context, p db.ListPermitsParams) ([]models.TransitPermit, error)
	UpdatePermitStatus(ctx context.Context, id uuid.UUID, status models.PermitStatus, now time.Time) error
	CreateBorderCrossing(ctx context.Context, b models.BorderCrossing) error
	ListBorderCrossings(ctx context.Context, vehicleID uuid.UUID) ([]models.BorderCrossing, error)
	CreateWeightCheck(ctx context.Context, w models.WeightCheck) error
	InsertAuditLog(ctx context.Context, l models.ComplianceAuditLog) error
}

type ComplianceHandler struct{ s Store }

func NewComplianceHandler(q *db.Queries) *ComplianceHandler       { return &ComplianceHandler{s:q} }
func NewComplianceHandlerWithStore(s Store) *ComplianceHandler     { return &ComplianceHandler{s:s} }

func (h *ComplianceHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/permits", h.createPermit).Methods(http.MethodPost)
	r.HandleFunc("/permits", h.listPermits).Methods(http.MethodGet)
	r.HandleFunc("/permits/{id}", h.getPermit).Methods(http.MethodGet)
	r.HandleFunc("/permits/{id}/status", h.updatePermitStatus).Methods(http.MethodPut)
	r.HandleFunc("/border-crossings", h.createCrossing).Methods(http.MethodPost)
	r.HandleFunc("/border-crossings/vehicle/{vehicle_id}", h.listCrossings).Methods(http.MethodGet)
	r.HandleFunc("/weight-checks", h.createWeightCheck).Methods(http.MethodPost)
}

func (h *ComplianceHandler) createPermit(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreatePermitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.VehicleID=="" || req.IssuedBy=="" || req.Country=="" || req.ValidFrom=="" || req.ValidUntil=="" { writeError(w,400,"VALIDATION_ERROR","vehicle_id, issued_by, country, valid_from, valid_until required"); return }
	vid,err := uuid.Parse(req.VehicleID); if err != nil { writeError(w,400,"INVALID_ID","invalid vehicle_id"); return }
	from,err := time.Parse(time.RFC3339,req.ValidFrom); if err != nil { writeError(w,400,"INVALID_DATE","valid_from must be RFC3339"); return }
	until,err := time.Parse(time.RFC3339,req.ValidUntil); if err != nil { writeError(w,400,"INVALID_DATE","valid_until must be RFC3339"); return }
	if req.PermitType=="" { req.PermitType=models.PermitTransit }
	now := time.Now().UTC()
	var driverID *uuid.UUID
	if req.DriverID!="" { if id,err:=uuid.Parse(req.DriverID); err==nil { driverID=&id } }
	p := models.TransitPermit{ID:uuid.New(), PermitNo:"PM-"+uuid.New().String()[:10], VehicleID:vid, DriverID:driverID,
		PermitType:req.PermitType, Status:models.PermitActive, IssuedBy:req.IssuedBy, Country:req.Country,
		RouteRestriction:req.RouteRestriction, MaxWeightKg:req.MaxWeightKg, ValidFrom:from.UTC(), ValidUntil:until.UTC(),
		Notes:req.Notes, CreatedAt:now, UpdatedAt:now}
	if err := h.s.CreatePermit(r.Context(), p); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, p.ID, claims.UserID, "create_permit", "compliance")
	writeJSON(w,201,models.APIResponse{Success:true, Data:p})
}

func (h *ComplianceHandler) listPermits(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := db.ListPermitsParams{Page:queryInt(q.Get("page"),1), Limit:queryInt(q.Get("limit"),20)}
	if v:=q.Get("country"); v!="" { p.Country=&v }
	if v:=q.Get("status"); v!="" { s:=models.PermitStatus(v); p.Status=&s }
	if v:=q.Get("vehicle_id"); v!="" { if id,err:=uuid.Parse(v); err==nil { p.VehicleID=&id } }
	permits,err := h.s.ListPermits(r.Context(), p)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:permits})
}

func (h *ComplianceHandler) getPermit(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid permit id"); return }
	p,err := h.s.GetPermit(r.Context(), id)
	if err != nil { writeError(w,404,"NOT_FOUND","permit not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:p})
}

func (h *ComplianceHandler) updatePermitStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid permit id"); return }
	var body struct{ Status models.PermitStatus `json:"status"` }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if err := h.s.UpdatePermitStatus(r.Context(), id, body.Status, time.Now().UTC()); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	p,_ := h.s.GetPermit(r.Context(), id)
	writeJSON(w,200,models.APIResponse{Success:true, Data:p})
}

func (h *ComplianceHandler) createCrossing(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreateBorderCrossingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.VehicleID=="" || req.DriverID=="" || req.FromCountry=="" || req.ToCountry=="" || req.BorderPost=="" { writeError(w,400,"VALIDATION_ERROR","vehicle_id, driver_id, from_country, to_country, border_post required"); return }
	vid,_ := uuid.Parse(req.VehicleID); did,_ := uuid.Parse(req.DriverID)
	now := time.Now().UTC()
	b := models.BorderCrossing{ID:uuid.New(), VehicleID:vid, DriverID:did, FromCountry:req.FromCountry, ToCountry:req.ToCountry,
		BorderPost:req.BorderPost, CargoDesc:req.CargoDesc, GrossWeightKg:req.GrossWeightKg, CrossedAt:now,
		ExitPermitNo:req.ExitPermitNo, EntryPermitNo:req.EntryPermitNo, Notes:req.Notes, CreatedAt:now}
	if err := h.s.CreateBorderCrossing(r.Context(), b); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, b.ID, claims.UserID, "create_border_crossing", "compliance")
	writeJSON(w,201,models.APIResponse{Success:true, Data:b})
}

func (h *ComplianceHandler) listCrossings(w http.ResponseWriter, r *http.Request) {
	vid,err := uuid.Parse(mux.Vars(r)["vehicle_id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid vehicle_id"); return }
	crossings,err := h.s.ListBorderCrossings(r.Context(), vid)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:crossings})
}

func (h *ComplianceHandler) createWeightCheck(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreateWeightCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.VehicleID=="" || req.Country=="" || req.CheckStation=="" { writeError(w,400,"VALIDATION_ERROR","vehicle_id, country, check_station required"); return }
	vid,_ := uuid.Parse(req.VehicleID)
	if req.Currency=="" { req.Currency="USD" }
	now := time.Now().UTC()
	isCompliant := req.GrossWeightKg <= req.LegalLimitKg
	wc := models.WeightCheck{ID:uuid.New(), VehicleID:vid, Country:req.Country, CheckStation:req.CheckStation,
		GrossWeightKg:req.GrossWeightKg, LegalLimitKg:req.LegalLimitKg, IsCompliant:isCompliant,
		FineAmount:req.FineAmount, Currency:req.Currency, CheckedAt:now, Notes:req.Notes, CreatedAt:now}
	if err := h.s.CreateWeightCheck(r.Context(), wc); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, vid, claims.UserID, "weight_check", "compliance")
	writeJSON(w,201,models.APIResponse{Success:true, Data:wc})
}

func (h *ComplianceHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid,_ := uuid.Parse(userID); eid := entityID
	h.s.InsertAuditLog(r.Context(), models.ComplianceAuditLog{ID:uuid.New(),EntityID:&eid,UserID:uid,Action:action,Resource:resource,IPAddress:r.RemoteAddr,CreatedAt:time.Now().UTC()})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type","application/json"); w.WriteHeader(status); json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w,status,models.APIResponse{Success:false,Error:&models.APIError{Code:code,Message:msg}})
}
func queryInt(s string, def int) int {
	if v,err:=strconv.Atoi(s); err==nil && v>0 { return v }; return def
}
