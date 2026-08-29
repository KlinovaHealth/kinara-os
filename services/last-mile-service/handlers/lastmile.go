package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/last-mile-service/db"
	"github.com/klinova/kinara-os/last-mile-service/middleware"
	"github.com/klinova/kinara-os/last-mile-service/models"
)

type Store interface {
	CreateDelivery(ctx context.Context, d models.Delivery) error
	GetDelivery(ctx context.Context, id uuid.UUID) (*models.Delivery, error)
	ListDeliveries(ctx context.Context, p db.ListDeliveryParams) ([]models.Delivery, error)
	AssignDriver(ctx context.Context, id, driverID uuid.UUID, now time.Time) error
	RecordDelivered(ctx context.Context, id uuid.UUID, photoURL, sigURL, notes string, now time.Time) error
	RecordFailure(ctx context.Context, id uuid.UUID, reason models.FailureReason, nextAt *time.Time, notes string, now time.Time) error
	InsertAuditLog(ctx context.Context, l models.LastMileAuditLog) error
}

type LastMileHandler struct{ s Store }

func NewLastMileHandler(q *db.Queries) *LastMileHandler       { return &LastMileHandler{s:q} }
func NewLastMileHandlerWithStore(s Store) *LastMileHandler     { return &LastMileHandler{s:s} }

func (h *LastMileHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/deliveries", h.create).Methods(http.MethodPost)
	r.HandleFunc("/deliveries", h.list).Methods(http.MethodGet)
	r.HandleFunc("/deliveries/{id}", h.get).Methods(http.MethodGet)
	r.HandleFunc("/deliveries/{id}/assign", h.assignDriver).Methods(http.MethodPut)
	r.HandleFunc("/deliveries/{id}/delivered", h.recordDelivered).Methods(http.MethodPut)
	r.HandleFunc("/deliveries/{id}/failed", h.recordFailed).Methods(http.MethodPut)
}

func (h *LastMileHandler) create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreateDeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.RecipientName=="" || req.RecipientPhone=="" || req.DeliveryAddress=="" { writeError(w,400,"VALIDATION_ERROR","recipient_name, recipient_phone, delivery_address required"); return }
	now := time.Now().UTC()
	var cargoID *uuid.UUID
	if req.CargoID!="" { if id,err:=uuid.Parse(req.CargoID); err==nil { cargoID=&id } }
	var winStart, winEnd *time.Time
	if req.WindowStart!="" { if t,err:=time.Parse(time.RFC3339,req.WindowStart); err==nil { tUTC:=t.UTC(); winStart=&tUTC } }
	if req.WindowEnd!="" { if t,err:=time.Parse(time.RFC3339,req.WindowEnd); err==nil { tUTC:=t.UTC(); winEnd=&tUTC } }
	d := models.Delivery{ID:uuid.New(), DeliveryCode:"DL-"+uuid.New().String()[:8], CargoID:cargoID,
		RecipientName:req.RecipientName, RecipientPhone:req.RecipientPhone, DeliveryAddress:req.DeliveryAddress,
		DeliveryLat:req.DeliveryLat, DeliveryLng:req.DeliveryLng, Status:models.DeliveryPending,
		WindowStart:winStart, WindowEnd:winEnd, AttemptCount:0, SMSNotified:false, Country:req.Country, Notes:req.Notes, CreatedAt:now, UpdatedAt:now}
	if err := h.s.CreateDelivery(r.Context(), d); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, d.ID, claims.UserID, "create_delivery", "last_mile")
	writeJSON(w,201,models.APIResponse{Success:true, Data:d})
}

func (h *LastMileHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := db.ListDeliveryParams{Page:queryInt(q.Get("page"),1), Limit:queryInt(q.Get("limit"),20)}
	if v:=q.Get("status"); v!="" { s:=models.DeliveryStatus(v); p.Status=&s }
	deliveries,err := h.s.ListDeliveries(r.Context(), p)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:deliveries})
}

func (h *LastMileHandler) get(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid delivery id"); return }
	d,err := h.s.GetDelivery(r.Context(), id)
	if err != nil { writeError(w,404,"NOT_FOUND","delivery not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:d})
}

func (h *LastMileHandler) assignDriver(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid delivery id"); return }
	var req models.AssignDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	did,err := uuid.Parse(req.DriverID)
	if err != nil { writeError(w,400,"INVALID_ID","invalid driver_id"); return }
	if err := h.s.AssignDriver(r.Context(), id, did, time.Now().UTC()); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, id, claims.UserID, "assign_driver", "last_mile")
	d,_ := h.s.GetDelivery(r.Context(), id)
	writeJSON(w,200,models.APIResponse{Success:true, Data:d})
}

func (h *LastMileHandler) recordDelivered(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid delivery id"); return }
	var req models.RecordDeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if err := h.s.RecordDelivered(r.Context(), id, req.ProofPhotoURL, req.SignatureURL, req.Notes, time.Now().UTC()); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, id, claims.UserID, "record_delivered", "last_mile")
	d,_ := h.s.GetDelivery(r.Context(), id)
	writeJSON(w,200,models.APIResponse{Success:true, Data:d})
}

func (h *LastMileHandler) recordFailed(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid delivery id"); return }
	var req models.RecordFailureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	now := time.Now().UTC()
	var nextAt *time.Time
	if req.NextAttemptAt!="" { if t,err:=time.Parse(time.RFC3339,req.NextAttemptAt); err==nil { tUTC:=t.UTC(); nextAt=&tUTC } }
	if err := h.s.RecordFailure(r.Context(), id, req.FailureReason, nextAt, req.Notes, now); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, id, claims.UserID, "record_failed:"+string(req.FailureReason), "last_mile")
	d,_ := h.s.GetDelivery(r.Context(), id)
	writeJSON(w,200,models.APIResponse{Success:true, Data:d})
}

func (h *LastMileHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid,_ := uuid.Parse(userID); eid := entityID
	h.s.InsertAuditLog(r.Context(), models.LastMileAuditLog{ID:uuid.New(),EntityID:&eid,UserID:uid,Action:action,Resource:resource,IPAddress:r.RemoteAddr,CreatedAt:time.Now().UTC()})
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
