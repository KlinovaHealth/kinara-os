package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/shipment-service/db"
	"github.com/klinova/kinara-os/shipment-service/middleware"
	"github.com/klinova/kinara-os/shipment-service/models"
)

type Store interface {
	CreateShipment(ctx context.Context, s models.Shipment) error
	GetShipment(ctx context.Context, id uuid.UUID) (*models.Shipment, error)
	GetShipmentByTrackingCode(ctx context.Context, code string) (*models.Shipment, error)
	ListShipments(ctx context.Context, p db.ListShipmentsParams) ([]models.Shipment, error)
	UpdateShipmentStatus(ctx context.Context, id uuid.UUID, status models.ShipmentStatus, now time.Time) error
	AddEvent(ctx context.Context, e models.ShipmentEvent) error
	ListEvents(ctx context.Context, shipmentID uuid.UUID) ([]models.ShipmentEvent, error)
	InsertAuditLog(ctx context.Context, l models.ShipmentAuditLog) error
}

type ShipmentHandler struct{ s Store }

func NewShipmentHandler(q *db.Queries) *ShipmentHandler       { return &ShipmentHandler{s:q} }
func NewShipmentHandlerWithStore(s Store) *ShipmentHandler     { return &ShipmentHandler{s:s} }

func (h *ShipmentHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/shipments", h.create).Methods(http.MethodPost)
	r.HandleFunc("/shipments", h.list).Methods(http.MethodGet)
	r.HandleFunc("/shipments/{id}", h.get).Methods(http.MethodGet)
	r.HandleFunc("/shipments/track/{code}", h.getByCode).Methods(http.MethodGet)
	r.HandleFunc("/shipments/{id}/status", h.updateStatus).Methods(http.MethodPut)
	r.HandleFunc("/shipments/{id}/events", h.addEvent).Methods(http.MethodPost)
	r.HandleFunc("/shipments/{id}/events", h.listEvents).Methods(http.MethodGet)
}

func (h *ShipmentHandler) create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreateShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.RecipientName=="" || req.OriginAddress=="" || req.DestAddress=="" || req.WeightKg<=0 { writeError(w,400,"VALIDATION_ERROR","recipient_name, origin_address, destination_address, weight_kg required"); return }
	if req.ServiceLevel=="" { req.ServiceLevel=models.ServiceStandard }
	if req.Currency=="" { req.Currency="USD" }
	senderID,_ := uuid.Parse(claims.UserID)
	freightCharge := req.WeightKg * 2.5
	insuranceCharge := req.DeclaredValue * 0.005
	now := time.Now().UTC()
	s := models.Shipment{ID:uuid.New(), TrackingCode:"KN-"+uuid.New().String()[:10],
		SenderID:senderID, RecipientName:req.RecipientName, RecipientPhone:req.RecipientPhone,
		OriginAddress:req.OriginAddress, OriginCountry:req.OriginCountry, DestAddress:req.DestAddress, DestCountry:req.DestCountry,
		WeightKg:req.WeightKg, LengthCm:req.LengthCm, WidthCm:req.WidthCm, HeightCm:req.HeightCm, DeclaredValue:req.DeclaredValue,
		Currency:req.Currency, ServiceLevel:req.ServiceLevel, Status:models.ShipmentCreated,
		FreightCharge:freightCharge, InsuranceCharge:insuranceCharge, TotalCharge:freightCharge+insuranceCharge,
		Notes:req.Notes, CreatedAt:now, UpdatedAt:now}
	if err := h.s.CreateShipment(r.Context(), s); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, s.ID, claims.UserID, "create_shipment", "shipment")
	writeJSON(w,201,models.APIResponse{Success:true, Data:s})
}

func (h *ShipmentHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := db.ListShipmentsParams{Page:queryInt(q.Get("page"),1), Limit:queryInt(q.Get("limit"),20)}
	if v:=q.Get("status"); v!="" { s:=models.ShipmentStatus(v); p.Status=&s }
	shipments,err := h.s.ListShipments(r.Context(), p)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:shipments})
}

func (h *ShipmentHandler) get(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid shipment id"); return }
	s,err := h.s.GetShipment(r.Context(), id)
	if err != nil { writeError(w,404,"NOT_FOUND","shipment not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:s})
}

func (h *ShipmentHandler) getByCode(w http.ResponseWriter, r *http.Request) {
	code := mux.Vars(r)["code"]
	s,err := h.s.GetShipmentByTrackingCode(r.Context(), code)
	if err != nil { writeError(w,404,"NOT_FOUND","shipment not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:s})
}

func (h *ShipmentHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid shipment id"); return }
	var body struct{ Status models.ShipmentStatus `json:"status"` }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if err := h.s.UpdateShipmentStatus(r.Context(), id, body.Status, time.Now().UTC()); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, id, claims.UserID, "update_status:"+string(body.Status), "shipment")
	s,_ := h.s.GetShipment(r.Context(), id)
	writeJSON(w,200,models.APIResponse{Success:true, Data:s})
}

func (h *ShipmentHandler) addEvent(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid shipment id"); return }
	var req models.AddShipmentEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	now := time.Now().UTC()
	e := models.ShipmentEvent{ID:uuid.New(), ShipmentID:id, Status:req.Status, Location:req.Location, Notes:req.Notes, EventTime:now, CreatedAt:now}
	if err := h.s.AddEvent(r.Context(), e); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,201,models.APIResponse{Success:true, Data:e})
}

func (h *ShipmentHandler) listEvents(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid shipment id"); return }
	events,err := h.s.ListEvents(r.Context(), id)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:events})
}

func (h *ShipmentHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid,_ := uuid.Parse(userID); eid := entityID
	h.s.InsertAuditLog(r.Context(), models.ShipmentAuditLog{ID:uuid.New(),EntityID:&eid,UserID:uid,Action:action,Resource:resource,IPAddress:r.RemoteAddr,CreatedAt:time.Now().UTC()})
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
