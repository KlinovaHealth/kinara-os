package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/transport-service/db"
	"github.com/klinova/kinara-os/transport-service/middleware"
	"github.com/klinova/kinara-os/transport-service/models"
)

type Store interface {
	CreateTrip(ctx context.Context, t models.TransportTrip) error
	GetTrip(ctx context.Context, id uuid.UUID) (*models.TransportTrip, error)
	ListTrips(ctx context.Context, p db.ListTripsParams) ([]models.TransportTrip, error)
	UpdateTripStatus(ctx context.Context, id uuid.UUID, status models.TripStatus, delay string, now time.Time) error
	UpdateGPS(ctx context.Context, id uuid.UUID, lat, lng float64, now time.Time) error
	AddGPSUpdate(ctx context.Context, g models.GPSUpdate) error
	InsertAuditLog(ctx context.Context, l models.TransportAuditLog) error
}

type TransportHandler struct{ s Store }

func NewTransportHandler(q *db.Queries) *TransportHandler       { return &TransportHandler{s:q} }
func NewTransportHandlerWithStore(s Store) *TransportHandler     { return &TransportHandler{s:s} }

func (h *TransportHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/trips", h.create).Methods(http.MethodPost)
	r.HandleFunc("/trips", h.list).Methods(http.MethodGet)
	r.HandleFunc("/trips/{id}", h.get).Methods(http.MethodGet)
	r.HandleFunc("/trips/{id}/status", h.updateStatus).Methods(http.MethodPut)
	r.HandleFunc("/trips/{id}/gps", h.updateGPS).Methods(http.MethodPut)
}

func (h *TransportHandler) create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.VehicleID=="" || req.DriverID=="" || req.OriginAddress=="" || req.DestAddress=="" || req.Country=="" { writeError(w,400,"VALIDATION_ERROR","vehicle_id, driver_id, origin_address, destination_address, country required"); return }
	vid,err := uuid.Parse(req.VehicleID); if err != nil { writeError(w,400,"INVALID_ID","invalid vehicle_id"); return }
	did,err := uuid.Parse(req.DriverID); if err != nil { writeError(w,400,"INVALID_ID","invalid driver_id"); return }
	pickup,err := time.Parse(time.RFC3339,req.ScheduledPickup); if err != nil { writeError(w,400,"INVALID_DATE","scheduled_pickup must be RFC3339"); return }
	now := time.Now().UTC()
	var routeID, cargoID *uuid.UUID
	if req.RouteID!="" { if id,err:=uuid.Parse(req.RouteID); err==nil { routeID=&id } }
	if req.CargoID!="" { if id,err:=uuid.Parse(req.CargoID); err==nil { cargoID=&id } }
	var schedDel *time.Time
	if req.ScheduledDelivery!="" { if t,err:=time.Parse(time.RFC3339,req.ScheduledDelivery); err==nil { tUTC:=t.UTC(); schedDel=&tUTC } }
	if req.Currency=="" { req.Currency="USD" }
	t := models.TransportTrip{ID:uuid.New(), TripCode:"TR-"+uuid.New().String()[:8], RouteID:routeID, VehicleID:vid, DriverID:did, CargoID:cargoID,
		Status:models.TripScheduled, Country:req.Country, OriginAddress:req.OriginAddress, OriginLat:req.OriginLat, OriginLng:req.OriginLng,
		DestAddress:req.DestAddress, DestLat:req.DestLat, DestLng:req.DestLng, ScheduledPickup:pickup.UTC(), ScheduledDelivery:schedDel,
		DistanceKm:req.DistanceKm, CostPerKm:req.CostPerKm, TotalCost:req.DistanceKm*req.CostPerKm, Currency:req.Currency, Notes:req.Notes, CreatedAt:now, UpdatedAt:now}
	if err := h.s.CreateTrip(r.Context(), t); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, t.ID, claims.UserID, "create_trip", "transport")
	writeJSON(w,201,models.APIResponse{Success:true, Data:t})
}

func (h *TransportHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := db.ListTripsParams{Page:queryInt(q.Get("page"),1), Limit:queryInt(q.Get("limit"),20)}
	if v:=q.Get("country"); v!="" { p.Country=&v }
	if v:=q.Get("status"); v!="" { s:=models.TripStatus(v); p.Status=&s }
	trips,err := h.s.ListTrips(r.Context(), p)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:trips})
}

func (h *TransportHandler) get(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid trip id"); return }
	t,err := h.s.GetTrip(r.Context(), id)
	if err != nil { writeError(w,404,"NOT_FOUND","trip not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:t})
}

func (h *TransportHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid trip id"); return }
	var req models.UpdateTripStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if err := h.s.UpdateTripStatus(r.Context(), id, req.Status, req.DelayReasonCode, time.Now().UTC()); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, id, claims.UserID, "update_status:"+string(req.Status), "transport")
	t,_ := h.s.GetTrip(r.Context(), id)
	writeJSON(w,200,models.APIResponse{Success:true, Data:t})
}

func (h *TransportHandler) updateGPS(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid trip id"); return }
	var req models.UpdateGPSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	now := time.Now().UTC()
	h.s.UpdateGPS(r.Context(), id, req.Latitude, req.Longitude, now)
	g := models.GPSUpdate{ID:uuid.New(), TripID:id, Latitude:req.Latitude, Longitude:req.Longitude, SpeedKph:req.SpeedKph, Heading:req.Heading, RecordedAt:now, CreatedAt:now}
	h.s.AddGPSUpdate(r.Context(), g)
	writeJSON(w,200,models.APIResponse{Success:true, Data:g})
}

func (h *TransportHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid,_ := uuid.Parse(userID); eid := entityID
	h.s.InsertAuditLog(r.Context(), models.TransportAuditLog{ID:uuid.New(),EntityID:&eid,UserID:uid,Action:action,Resource:resource,IPAddress:r.RemoteAddr,CreatedAt:time.Now().UTC()})
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
