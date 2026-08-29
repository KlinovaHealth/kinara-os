package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/cargo-service/db"
	"github.com/klinova/kinara-os/cargo-service/middleware"
	"github.com/klinova/kinara-os/cargo-service/models"
)

type Store interface {
	CreateBooking(ctx context.Context, b models.CargoBooking) error
	GetBooking(ctx context.Context, id uuid.UUID) (*models.CargoBooking, error)
	GetBookingByRef(ctx context.Context, ref string) (*models.CargoBooking, error)
	ListBookings(ctx context.Context, p db.ListCargoParams) ([]models.CargoBooking, error)
	UpdateBookingStatus(ctx context.Context, id uuid.UUID, status models.CargoStatus, now time.Time) error
	AssignCargo(ctx context.Context, id, vehicleID, driverID uuid.UUID, now time.Time) error
	AddTrackingEvent(ctx context.Context, e models.TrackingEvent) error
	ListTracking(ctx context.Context, cargoID uuid.UUID) ([]models.TrackingEvent, error)
	InsertAuditLog(ctx context.Context, l models.CargoAuditLog) error
}

type CargoHandler struct{ s Store }

func NewCargoHandler(q *db.Queries) *CargoHandler       { return &CargoHandler{s:q} }
func NewCargoHandlerWithStore(s Store) *CargoHandler     { return &CargoHandler{s:s} }

func (h *CargoHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/cargo", h.create).Methods(http.MethodPost)
	r.HandleFunc("/cargo", h.list).Methods(http.MethodGet)
	r.HandleFunc("/cargo/{id}", h.get).Methods(http.MethodGet)
	r.HandleFunc("/cargo/ref/{ref}", h.getByRef).Methods(http.MethodGet)
	r.HandleFunc("/cargo/{id}/assign", h.assign).Methods(http.MethodPut)
	r.HandleFunc("/cargo/{id}/status", h.updateStatus).Methods(http.MethodPut)
	r.HandleFunc("/cargo/{id}/track", h.addEvent).Methods(http.MethodPost)
	r.HandleFunc("/cargo/{id}/track", h.tracking).Methods(http.MethodGet)
}

func (h *CargoHandler) create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreateCargoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.WeightKg<=0 || req.OriginAddress=="" || req.DestinationAddress=="" { writeError(w,400,"VALIDATION_ERROR","weight_kg, origin_address, destination_address required"); return }
	if req.CargoType=="" { req.CargoType=models.CargoGeneral }
	if req.Currency=="" { req.Currency="USD" }
	now := time.Now().UTC()
	var estDelivery *time.Time
	if req.EstimatedDelivery!=nil { if t,err:=time.Parse(time.RFC3339,*req.EstimatedDelivery); err==nil { tUTC:=t.UTC(); estDelivery=&tUTC } }
	shipperID,_ := uuid.Parse(claims.UserID)
	b := models.CargoBooking{ID:uuid.New(), BookingRef:"KN-"+uuid.New().String()[:8],
		ShipperID:shipperID, CargoType:req.CargoType, Description:req.Description,
		WeightKg:req.WeightKg, VolumeM3:req.VolumeM3, Status:models.CargoPending,
		OriginAddress:req.OriginAddress, OriginLat:req.OriginLat, OriginLng:req.OriginLng,
		DestinationAddress:req.DestinationAddress, DestinationLat:req.DestinationLat, DestinationLng:req.DestinationLng,
		EstimatedDelivery:estDelivery, FreightCost:req.FreightCost, Currency:req.Currency,
		Notes:req.Notes, CreatedAt:now, UpdatedAt:now}
	if err := h.s.CreateBooking(r.Context(), b); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, b.ID, claims.UserID, "create_booking", "cargo")
	writeJSON(w,201,models.APIResponse{Success:true, Data:b})
}

func (h *CargoHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := db.ListCargoParams{Page:queryInt(q.Get("page"),1), Limit:queryInt(q.Get("limit"),20)}
	if v:=q.Get("status"); v!="" { s:=models.CargoStatus(v); p.Status=&s }
	bookings,err := h.s.ListBookings(r.Context(), p)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:bookings})
}

func (h *CargoHandler) get(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid cargo id"); return }
	b,err := h.s.GetBooking(r.Context(), id)
	if err != nil { writeError(w,404,"NOT_FOUND","cargo not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:b})
}

func (h *CargoHandler) getByRef(w http.ResponseWriter, r *http.Request) {
	ref := mux.Vars(r)["ref"]
	b,err := h.s.GetBookingByRef(r.Context(), ref)
	if err != nil { writeError(w,404,"NOT_FOUND","cargo not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:b})
}

func (h *CargoHandler) assign(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid cargo id"); return }
	var req models.AssignCargoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	vid,err := uuid.Parse(req.VehicleID)
	if err != nil { writeError(w,400,"INVALID_VEHICLE_ID","invalid vehicle_id"); return }
	did,err := uuid.Parse(req.DriverID)
	if err != nil { writeError(w,400,"INVALID_DRIVER_ID","invalid driver_id"); return }
	if err := h.s.AssignCargo(r.Context(), id, vid, did, time.Now().UTC()); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, id, claims.UserID, "assign_cargo", "cargo")
	b,_ := h.s.GetBooking(r.Context(), id)
	writeJSON(w,200,models.APIResponse{Success:true, Data:b})
}

func (h *CargoHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid cargo id"); return }
	var body struct{ Status models.CargoStatus `json:"status"` }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if err := h.s.UpdateBookingStatus(r.Context(), id, body.Status, time.Now().UTC()); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, id, claims.UserID, "update_status:"+string(body.Status), "cargo")
	b,_ := h.s.GetBooking(r.Context(), id)
	writeJSON(w,200,models.APIResponse{Success:true, Data:b})
}

func (h *CargoHandler) addEvent(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid cargo id"); return }
	var req models.TrackEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	now := time.Now().UTC()
	e := models.TrackingEvent{ID:uuid.New(), CargoID:id, Status:req.Status, Location:req.Location,
		Latitude:req.Latitude, Longitude:req.Longitude, Notes:req.Notes, EventTime:now, CreatedAt:now}
	if err := h.s.AddTrackingEvent(r.Context(), e); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,201,models.APIResponse{Success:true, Data:e})
}

func (h *CargoHandler) tracking(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid cargo id"); return }
	events,err := h.s.ListTracking(r.Context(), id)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:events})
}

func (h *CargoHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid,_ := uuid.Parse(userID); eid := entityID
	h.s.InsertAuditLog(r.Context(), models.CargoAuditLog{ID:uuid.New(),EntityID:&eid,UserID:uid,Action:action,Resource:resource,IPAddress:r.RemoteAddr,CreatedAt:time.Now().UTC()})
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
