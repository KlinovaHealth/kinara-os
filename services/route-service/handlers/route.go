package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/route-service/db"
	"github.com/klinova/kinara-os/route-service/middleware"
	"github.com/klinova/kinara-os/route-service/models"
)

type Store interface {
	CreateRoute(ctx context.Context, r models.Route) error
	GetRoute(ctx context.Context, id uuid.UUID) (*models.Route, error)
	ListRoutes(ctx context.Context, p db.ListRoutesParams) ([]models.Route, error)
	ScheduleRoute(ctx context.Context, s models.RouteSchedule) error
	ListSchedules(ctx context.Context, routeID uuid.UUID) ([]models.RouteSchedule, error)
	UpdateScheduleStatus(ctx context.Context, id uuid.UUID, status string, now time.Time) error
	InsertAuditLog(ctx context.Context, l models.RouteAuditLog) error
}

type RouteHandler struct{ s Store }

func NewRouteHandler(q *db.Queries) *RouteHandler       { return &RouteHandler{s:q} }
func NewRouteHandlerWithStore(s Store) *RouteHandler     { return &RouteHandler{s:s} }

func (h *RouteHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/routes", h.create).Methods(http.MethodPost)
	r.HandleFunc("/routes", h.list).Methods(http.MethodGet)
	r.HandleFunc("/routes/{id}", h.get).Methods(http.MethodGet)
	r.HandleFunc("/routes/{id}/schedule", h.schedule).Methods(http.MethodPost)
	r.HandleFunc("/routes/{id}/schedules", h.listSchedules).Methods(http.MethodGet)
	r.HandleFunc("/schedules/{schedule_id}/status", h.updateScheduleStatus).Methods(http.MethodPut)
}

func (h *RouteHandler) create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.Name=="" || req.Country=="" || req.OriginName=="" || req.DestName=="" { writeError(w,400,"VALIDATION_ERROR","name, country, origin_name, destination_name required"); return }
	if req.RouteType=="" { req.RouteType=models.RouteFixed }
	now := time.Now().UTC()
	route := models.Route{ID:uuid.New(), Name:req.Name, RouteCode:req.RouteCode, RouteType:req.RouteType,
		Status:models.RouteActive, Country:req.Country,
		OriginName:req.OriginName, OriginLat:req.OriginLat, OriginLng:req.OriginLng,
		DestName:req.DestName, DestLat:req.DestLat, DestLng:req.DestLng,
		DistanceKm:req.DistanceKm, EstHours:req.EstHours, Waypoints:req.Waypoints,
		FreightClass:req.FreightClass, Notes:req.Notes, CreatedAt:now, UpdatedAt:now}
	if route.RouteCode=="" { route.RouteCode="RT-"+uuid.New().String()[:8] }
	if err := h.s.CreateRoute(r.Context(), route); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, route.ID, claims.UserID, "create_route", "route")
	writeJSON(w,201,models.APIResponse{Success:true, Data:route})
}

func (h *RouteHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := db.ListRoutesParams{Page:queryInt(q.Get("page"),1), Limit:queryInt(q.Get("limit"),50)}
	if v:=q.Get("country"); v!="" { p.Country=&v }
	routes,err := h.s.ListRoutes(r.Context(), p)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:routes})
}

func (h *RouteHandler) get(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid route id"); return }
	route,err := h.s.GetRoute(r.Context(), id)
	if err != nil { writeError(w,404,"NOT_FOUND","route not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:route})
}

func (h *RouteHandler) schedule(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	routeID,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid route id"); return }
	var req models.ScheduleRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.DepartureTime=="" { writeError(w,400,"VALIDATION_ERROR","departure_time required"); return }
	deptTime,err := time.Parse(time.RFC3339, req.DepartureTime)
	if err != nil { writeError(w,400,"INVALID_DATE","departure_time must be RFC3339"); return }
	now := time.Now().UTC()
	var vid, did *uuid.UUID
	if req.VehicleID!="" { if v,err:=uuid.Parse(req.VehicleID); err==nil { vid=&v } }
	if req.DriverID!=""  { if d,err:=uuid.Parse(req.DriverID);  err==nil { did=&d } }
	s := models.RouteSchedule{ID:uuid.New(), RouteID:routeID, VehicleID:vid, DriverID:did,
		DepartureTime:deptTime.UTC(), Status:"scheduled", Notes:req.Notes, CreatedAt:now, UpdatedAt:now}
	if err := h.s.ScheduleRoute(r.Context(), s); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, routeID, claims.UserID, "schedule_route", "schedule")
	writeJSON(w,201,models.APIResponse{Success:true, Data:s})
}

func (h *RouteHandler) listSchedules(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid route id"); return }
	schedules,err := h.s.ListSchedules(r.Context(), id)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:schedules})
}

func (h *RouteHandler) updateScheduleStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["schedule_id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid schedule id"); return }
	var body struct{ Status string `json:"status"` }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if err := h.s.UpdateScheduleStatus(r.Context(), id, body.Status, time.Now().UTC()); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:map[string]string{"id":id.String(),"status":body.Status}})
}

func (h *RouteHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid,_ := uuid.Parse(userID); eid := entityID
	h.s.InsertAuditLog(r.Context(), models.RouteAuditLog{ID:uuid.New(),EntityID:&eid,UserID:uid,Action:action,Resource:resource,IPAddress:r.RemoteAddr,CreatedAt:time.Now().UTC()})
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
