package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/fleet-service/db"
	"github.com/klinova/kinara-os/fleet-service/middleware"
	"github.com/klinova/kinara-os/fleet-service/models"
)

type Store interface {
	CreateVehicle(ctx context.Context, v models.Vehicle) error
	GetVehicle(ctx context.Context, id uuid.UUID) (*models.Vehicle, error)
	ListVehicles(ctx context.Context, p db.ListVehiclesParams) ([]models.Vehicle, error)
	UpdateVehicle(ctx context.Context, id uuid.UUID, req models.UpdateVehicleRequest, now time.Time) error
	LogMaintenance(ctx context.Context, m models.MaintenanceRecord) error
	ListMaintenance(ctx context.Context, vehicleID uuid.UUID) ([]models.MaintenanceRecord, error)
	LogFuel(ctx context.Context, f models.FuelLog) error
	ListFuelLogs(ctx context.Context, vehicleID uuid.UUID) ([]models.FuelLog, error)
	InsertAuditLog(ctx context.Context, l models.FleetAuditLog) error
}

type FleetHandler struct{ s Store }

func NewFleetHandler(q *db.Queries) *FleetHandler       { return &FleetHandler{s: q} }
func NewFleetHandlerWithStore(s Store) *FleetHandler     { return &FleetHandler{s: s} }

func (h *FleetHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/vehicles", h.create).Methods(http.MethodPost)
	r.HandleFunc("/vehicles", h.list).Methods(http.MethodGet)
	r.HandleFunc("/vehicles/{id}", h.get).Methods(http.MethodGet)
	r.HandleFunc("/vehicles/{id}", h.update).Methods(http.MethodPut)
	r.HandleFunc("/vehicles/{id}/maintenance", h.logMaintenance).Methods(http.MethodPost)
	r.HandleFunc("/vehicles/{id}/maintenance", h.listMaintenance).Methods(http.MethodGet)
	r.HandleFunc("/vehicles/{id}/fuel", h.logFuel).Methods(http.MethodPost)
	r.HandleFunc("/vehicles/{id}/fuel", h.listFuel).Methods(http.MethodGet)
}

func (h *FleetHandler) create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreateVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.RegistrationNo=="" || req.Country=="" || req.VehicleType=="" { writeError(w,400,"VALIDATION_ERROR","registration_no, country, and vehicle_type required"); return }
	if req.FuelType=="" { req.FuelType=models.FuelDiesel }
	now := time.Now().UTC()
	v := models.Vehicle{ID:uuid.New(), RegistrationNo:req.RegistrationNo, VehicleType:req.VehicleType,
		Make:req.Make, Model:req.Model, Year:req.Year, FuelType:req.FuelType,
		PayloadCapacityKg:req.PayloadCapacityKg, VolumeCapacityM3:req.VolumeCapacityM3,
		Status:models.VehicleAvailable, Country:req.Country, BaseLocation:req.BaseLocation,
		Notes:req.Notes, CreatedAt:now, UpdatedAt:now}
	if err := h.s.CreateVehicle(r.Context(), v); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, v.ID, claims.UserID.String(), "create_vehicle", "vehicle")
	writeJSON(w,201,models.APIResponse{Success:true, Data:v})
}

func (h *FleetHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := db.ListVehiclesParams{Page:queryInt(q.Get("page"),1), Limit:queryInt(q.Get("limit"),20)}
	if v:=q.Get("country"); v!="" { p.Country=&v }
	if v:=q.Get("status"); v!="" { s:=models.VehicleStatus(v); p.Status=&s }
	vehicles, err := h.s.ListVehicles(r.Context(), p)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:vehicles})
}

func (h *FleetHandler) get(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid vehicle id"); return }
	v,err := h.s.GetVehicle(r.Context(), id)
	if err != nil { writeError(w,404,"NOT_FOUND","vehicle not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:v})
}

func (h *FleetHandler) update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid vehicle id"); return }
	var req models.UpdateVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if err := h.s.UpdateVehicle(r.Context(), id, req, time.Now().UTC()); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, id, claims.UserID.String(), "update_vehicle", "vehicle")
	v,_ := h.s.GetVehicle(r.Context(), id)
	writeJSON(w,200,models.APIResponse{Success:true, Data:v})
}

func (h *FleetHandler) logMaintenance(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid vehicle id"); return }
	var req models.LogMaintenanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.ServiceType=="" { writeError(w,400,"VALIDATION_ERROR","service_type required"); return }
	now := time.Now().UTC()
	svcAt := now
	if req.ServicedAt!="" { if t,err:=time.Parse(time.RFC3339,req.ServicedAt); err==nil { svcAt=t.UTC() } }
	m := models.MaintenanceRecord{ID:uuid.New(), VehicleID:id, ServiceType:req.ServiceType,
		Description:req.Description, OdometerKm:req.OdometerKm, Cost:req.Cost, Currency:req.Currency,
		ServicedBy:req.ServicedBy, ServicedAt:svcAt, NextServiceKm:req.NextServiceKm, CreatedAt:now}
	if err := h.s.LogMaintenance(r.Context(), m); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,201,models.APIResponse{Success:true, Data:m})
}

func (h *FleetHandler) listMaintenance(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid vehicle id"); return }
	records,err := h.s.ListMaintenance(r.Context(), id)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:records})
}

func (h *FleetHandler) logFuel(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid vehicle id"); return }
	var req models.LogFuelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.LitresFilled<=0 { writeError(w,400,"VALIDATION_ERROR","litres_filled must be positive"); return }
	if req.Currency=="" { req.Currency="USD" }
	now := time.Now().UTC()
	f := models.FuelLog{ID:uuid.New(), VehicleID:id, LitresFilled:req.LitresFilled,
		CostPerLitre:req.CostPerLitre, TotalCost:req.LitresFilled*req.CostPerLitre,
		Currency:req.Currency, OdometerKm:req.OdometerKm, Station:req.Station, FilledAt:now, CreatedAt:now}
	if err := h.s.LogFuel(r.Context(), f); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,201,models.APIResponse{Success:true, Data:f})
}

func (h *FleetHandler) listFuel(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid vehicle id"); return }
	logs,err := h.s.ListFuelLogs(r.Context(), id)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:logs})
}

func (h *FleetHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid,_ := uuid.Parse(userID); eid := entityID
	h.s.InsertAuditLog(r.Context(), models.FleetAuditLog{ID:uuid.New(),EntityID:&eid,UserID:uid,Action:action,Resource:resource,IPAddress:r.RemoteAddr,CreatedAt:time.Now().UTC()})
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
