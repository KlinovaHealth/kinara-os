package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/warehouse-service/db"
	"github.com/klinova/kinara-os/warehouse-service/middleware"
	"github.com/klinova/kinara-os/warehouse-service/models"
)

type Store interface {
	CreateWarehouse(ctx context.Context, w models.Warehouse) error
	GetWarehouse(ctx context.Context, id uuid.UUID) (*models.Warehouse, error)
	ListWarehouses(ctx context.Context, p db.ListWarehouseParams) ([]models.Warehouse, error)
	CreateStockItem(ctx context.Context, s models.StockItem) error
	GetStockItem(ctx context.Context, id uuid.UUID) (*models.StockItem, error)
	ListStockItems(ctx context.Context, warehouseID uuid.UUID) ([]models.StockItem, error)
	ListLowStock(ctx context.Context, warehouseID uuid.UUID) ([]models.StockItem, error)
	RecordMovement(ctx context.Context, m models.StockMovement, now time.Time) error
	InsertAuditLog(ctx context.Context, l models.WarehouseAuditLog) error
}

type WarehouseHandler struct{ s Store }

func NewWarehouseHandler(q *db.Queries) *WarehouseHandler       { return &WarehouseHandler{s:q} }
func NewWarehouseHandlerWithStore(s Store) *WarehouseHandler     { return &WarehouseHandler{s:s} }

func (h *WarehouseHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/warehouses", h.create).Methods(http.MethodPost)
	r.HandleFunc("/warehouses", h.list).Methods(http.MethodGet)
	r.HandleFunc("/warehouses/{id}", h.get).Methods(http.MethodGet)
	r.HandleFunc("/warehouses/{id}/stock", h.createStock).Methods(http.MethodPost)
	r.HandleFunc("/warehouses/{id}/stock", h.listStock).Methods(http.MethodGet)
	r.HandleFunc("/warehouses/{id}/stock/low", h.lowStock).Methods(http.MethodGet)
	r.HandleFunc("/warehouses/{id}/movements", h.recordMovement).Methods(http.MethodPost)
}

func (h *WarehouseHandler) create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	var req models.CreateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.Name=="" || req.Country=="" || req.Address=="" { writeError(w,400,"VALIDATION_ERROR","name, country, address required"); return }
	if req.Code=="" { req.Code="WH-"+uuid.New().String()[:8] }
	now := time.Now().UTC()
	wh := models.Warehouse{ID:uuid.New(), Name:req.Name, Code:req.Code, Country:req.Country, Region:req.Region, Address:req.Address,
		Latitude:req.Latitude, Longitude:req.Longitude, CapacityM3:req.CapacityM3, UsedM3:0, Status:models.WarehouseActive,
		ManagerName:req.ManagerName, ContactPhone:req.ContactPhone, Notes:req.Notes, CreatedAt:now, UpdatedAt:now}
	if err := h.s.CreateWarehouse(r.Context(), wh); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, wh.ID, claims.UserID, "create_warehouse", "warehouse")
	writeJSON(w,201,models.APIResponse{Success:true, Data:wh})
}

func (h *WarehouseHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := db.ListWarehouseParams{Page:queryInt(q.Get("page"),1), Limit:queryInt(q.Get("limit"),20)}
	if v:=q.Get("country"); v!="" { p.Country=&v }
	warehouses,err := h.s.ListWarehouses(r.Context(), p)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:warehouses})
}

func (h *WarehouseHandler) get(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid warehouse id"); return }
	wh,err := h.s.GetWarehouse(r.Context(), id)
	if err != nil { writeError(w,404,"NOT_FOUND","warehouse not found"); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:wh})
}

func (h *WarehouseHandler) createStock(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid warehouse id"); return }
	var req models.CreateStockItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	if req.SKU=="" || req.ProductName=="" { writeError(w,400,"VALIDATION_ERROR","sku and product_name required"); return }
	if req.Unit=="" { req.Unit="kg" }
	now := time.Now().UTC()
	s := models.StockItem{ID:uuid.New(), WarehouseID:id, SKU:req.SKU, ProductName:req.ProductName, Category:req.Category,
		BinLocation:req.BinLocation, QuantityOnHand:0, Unit:req.Unit, UnitWeightKg:req.UnitWeightKg, UnitVolumeM3:req.UnitVolumeM3,
		ReorderLevel:req.ReorderLevel, CreatedAt:now, UpdatedAt:now}
	if err := h.s.CreateStockItem(r.Context(), s); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,201,models.APIResponse{Success:true, Data:s})
}

func (h *WarehouseHandler) listStock(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid warehouse id"); return }
	items,err := h.s.ListStockItems(r.Context(), id)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:items})
}

func (h *WarehouseHandler) lowStock(w http.ResponseWriter, r *http.Request) {
	id,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid warehouse id"); return }
	items,err := h.s.ListLowStock(r.Context(), id)
	if err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	writeJSON(w,200,models.APIResponse{Success:true, Data:items})
}

func (h *WarehouseHandler) recordMovement(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims==nil { writeError(w,401,"UNAUTHORIZED","missing auth"); return }
	whID,err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil { writeError(w,400,"INVALID_ID","invalid warehouse id"); return }
	var req models.StockMovementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w,400,"BAD_REQUEST","invalid JSON"); return }
	itemID,err := uuid.Parse(req.StockItemID)
	if err != nil { writeError(w,400,"INVALID_ID","invalid stock_item_id"); return }
	if req.Quantity<=0 { writeError(w,400,"VALIDATION_ERROR","quantity must be positive"); return }
	now := time.Now().UTC()
	recorderID,_ := uuid.Parse(claims.UserID)
	var refID *uuid.UUID
	if req.RefID!="" { if id,err:=uuid.Parse(req.RefID); err==nil { refID=&id } }
	m := models.StockMovement{ID:uuid.New(), WarehouseID:whID, StockItemID:itemID, MovementType:req.MovementType,
		Quantity:req.Quantity, RefID:refID, RefType:req.RefType, Notes:req.Notes, RecordedBy:recorderID, CreatedAt:now}
	if err := h.s.RecordMovement(r.Context(), m, now); err != nil { writeError(w,500,"DB_ERROR",err.Error()); return }
	h.audit(r, whID, claims.UserID, "record_movement:"+string(req.MovementType), "warehouse")
	writeJSON(w,201,models.APIResponse{Success:true, Data:m})
}

func (h *WarehouseHandler) audit(r *http.Request, entityID uuid.UUID, userID, action, resource string) {
	uid,_ := uuid.Parse(userID); eid := entityID
	h.s.InsertAuditLog(r.Context(), models.WarehouseAuditLog{ID:uuid.New(),EntityID:&eid,UserID:uid,Action:action,Resource:resource,IPAddress:r.RemoteAddr,CreatedAt:time.Now().UTC()})
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
