package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/driver-service/crypto"
	"github.com/klinova/kinara-os/driver-service/db"
	"github.com/klinova/kinara-os/driver-service/middleware"
	"github.com/klinova/kinara-os/driver-service/models"
)

type Store interface {
	CreateDriver(ctx context.Context, d models.DriverRow) error
	GetDriver(ctx context.Context, id uuid.UUID) (*models.DriverRow, error)
	ListDrivers(ctx context.Context, p db.ListDriversParams) ([]models.DriverRow, error)
	UpdateDriver(ctx context.Context, id uuid.UUID, req models.UpdateDriverRequest, now time.Time) error
	LogTrip(ctx context.Context, t models.DriverTrip) error
	ListTrips(ctx context.Context, driverID uuid.UUID) ([]models.DriverTrip, error)
	IncrementTripStats(ctx context.Context, id uuid.UUID, km float64, now time.Time) error
	InsertAuditLog(ctx context.Context, l models.DriverAuditLog) error
}

type DriverHandler struct{ s Store; enc *crypto.Encryptor }

func NewDriverHandler(q *db.Queries, enc *crypto.Encryptor) *DriverHandler       { return &DriverHandler{s:q,enc:enc} }
func NewDriverHandlerWithStore(s Store, enc *crypto.Encryptor) *DriverHandler     { return &DriverHandler{s:s,enc:enc} }

func (h *DriverHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/drivers", h.create).Methods(http.MethodPost)
	r.HandleFunc("/drivers", h.list).Methods(http.MethodGet)
	r.HandleFunc("/drivers/{id}", h.get).Methods(http.MethodGet)
	r.HandleFunc("/drivers/{id}", h.update).Methods(http.MethodPut)
	r.HandleFunc("/drivers/{id}/trips", h.logTrip).Methods(http.MethodPost)
	r.HandleFunc("/drivers/{id}/trips", h.listTrips).Methods(http.MethodGet)
}

func (h *DriverHandler) create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreateDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.FullName=="" || req.Phone=="" || req.LicenseNo=="" || req.Country=="" { writeError(w,400,"VALIDATION_ERROR","full_name, phone, license_no, country required"); return }
	if req.LicenseClass=="" { req.LicenseClass=models.LicenseB }
	expiry,err := time.Parse("2006-01-02",req.LicenseExpiry)
	if err != nil { writeError(w,400,"INVALID_DATE","license_expiry must be YYYY-MM-DD"); return }
	nameEnc,_ := h.enc.EncryptString(req.FullName)
	phoneEnc,_ := h.enc.EncryptString(req.Phone)
	idEnc,_ := h.enc.EncryptString(req.NationalID)
	now := time.Now().UTC()
	row := models.DriverRow{ID:uuid.New(), FullNameEnc:nameEnc, PhoneEnc:phoneEnc, NationalIDEnc:idEnc,
		LicenseNo:req.LicenseNo, LicenseClass:req.LicenseClass, LicenseExpiry:expiry.UTC(),
		Status:models.DriverAvailable, Country:req.Country, BaseLocation:req.BaseLocation,
		TotalTrips:0, TotalKm:0, Rating:5.0, CreatedAt:now, UpdatedAt:now}
	if err := h.s.CreateDriver(r.Context(), row); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, row.ID, claims.UserID, "create_driver", "driver")
	writeJSON(w,201,models.APIResponse{Success:true, Data:h.decryptDriver(row)})
}

func (h *DriverHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := db.ListDriversParams{Page:queryInt(q.Get("page"),1), Limit:queryInt(q.Get("limit"),20)}
	if v:=q.Get("country"); v!="" { p.Country=&v }
	if v:=q.Get("status"); v!="" { s:=models.DriverStatus(v); p.Status=&s }
	rows,err := h.s.ListDrivers(r.Context(), p)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	var drivers []models.Driver
	for _,row := range rows {
		d := h.decryptDriver(row)
		d.NationalID = "" // omit in list view
		drivers = append(drivers, d)
	}
	writeJSON(w,200,models.APIResponse{Success:true, Data:drivers})
}

func (h *DriverHandler) get(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid driver id"); return }
	row,err := h.s.GetDriver(r.Context(), id)
	if err != nil { writeError(w,404,"NOT_FOUND","driver not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:h.decryptDriver(*row)})
}

func (h *DriverHandler) update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid driver id"); return }
	var req models.UpdateDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if err := h.s.UpdateDriver(r.Context(), id, req, time.Now().UTC()); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, id, claims.UserID, "update_driver", "driver")
	row,_ := h.s.GetDriver(r.Context(), id)
	writeJSON(w,200,models.APIResponse{Success:true, Data:h.decryptDriver(*row)})
}

func (h *DriverHandler) logTrip(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid driver id"); return }
	var req models.LogTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	vid,err := uuid.Parse(req.VehicleID)
	if err != nil { writeError(w,400,"INVALID_VEHICLE_ID","invalid vehicle_id"); return }
	now := time.Now().UTC()
	start := now
	if req.StartTime!="" { if t,err:=time.Parse(time.RFC3339,req.StartTime); err==nil { start=t.UTC() } }
	var routeID *uuid.UUID
	if req.RouteID!="" { if rid,err:=uuid.Parse(req.RouteID); err==nil { routeID=&rid } }
	trip := models.DriverTrip{ID:uuid.New(), DriverID:id, VehicleID:vid, RouteID:routeID,
		DistanceKm:req.DistanceKm, StartTime:start, Status:"active", Notes:req.Notes, CreatedAt:now}
	if err := h.s.LogTrip(r.Context(), trip); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	if req.DistanceKm>0 { h.s.IncrementTripStats(r.Context(), id, req.DistanceKm, now) }
	writeJSON(w,201,models.APIResponse{Success:true, Data:trip})
}

func (h *DriverHandler) listTrips(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid driver id"); return }
	trips,err := h.s.ListTrips(r.Context(), id)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:trips})
}

func (h *DriverHandler) decryptDriver(row models.DriverRow) models.Driver {
	name,_ := h.enc.DecryptString(row.FullNameEnc)
	phone,_ := h.enc.DecryptString(row.PhoneEnc)
	natID,_ := h.enc.DecryptString(row.NationalIDEnc)
	return models.Driver{ID:row.ID, FullName:name, Phone:phone, NationalID:natID,
		LicenseNo:row.LicenseNo, LicenseClass:row.LicenseClass, LicenseExpiry:row.LicenseExpiry,
		Status:row.Status, Country:row.Country, BaseLocation:row.BaseLocation,
		TotalTrips:row.TotalTrips, TotalKm:row.TotalKm, Rating:row.Rating,
		AssignedVehicleID:row.AssignedVehicleID, CreatedAt:row.CreatedAt, UpdatedAt:row.UpdatedAt}
}

func (h *DriverHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid,_ := uuid.Parse(userID); eid := entityID
	h.s.InsertAuditLog(r.Context(), models.DriverAuditLog{ID:uuid.New(),EntityID:&eid,UserID:uid,Action:action,Resource:resource,IPAddress:r.RemoteAddr,CreatedAt:time.Now().UTC()})
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
