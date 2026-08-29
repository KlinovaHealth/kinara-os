package models

import (
	"time"

	"github.com/google/uuid"
)

type WarehouseStatus string

const (
	WarehouseActive   WarehouseStatus = "active"
	WarehouseInactive WarehouseStatus = "inactive"
	WarehouseFull     WarehouseStatus = "full"
)

type StockMovementType string

const (
	MovementReceive  StockMovementType = "receive"
	MovementDispatch StockMovementType = "dispatch"
	MovementTransfer StockMovementType = "transfer"
	MovementAdjust   StockMovementType = "adjust"
)

type Warehouse struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Code            string          `json:"code"`
	Country         string          `json:"country"`
	Region          string          `json:"region"`
	Address         string          `json:"address"`
	Latitude        float64         `json:"latitude"`
	Longitude       float64         `json:"longitude"`
	CapacityM3      float64         `json:"capacity_m3"`
	UsedM3          float64         `json:"used_m3"`
	Status          WarehouseStatus `json:"status"`
	ManagerName     string          `json:"manager_name"`
	ContactPhone    string          `json:"contact_phone"`
	Notes           string          `json:"notes,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type StockItem struct {
	ID              uuid.UUID  `json:"id"`
	WarehouseID     uuid.UUID  `json:"warehouse_id"`
	SKU             string     `json:"sku"`
	ProductName     string     `json:"product_name"`
	Category        string     `json:"category"`
	BinLocation     string     `json:"bin_location"`
	QuantityOnHand  float64    `json:"quantity_on_hand"`
	Unit            string     `json:"unit"`
	UnitWeightKg    float64    `json:"unit_weight_kg"`
	UnitVolumeM3    float64    `json:"unit_volume_m3"`
	ReorderLevel    float64    `json:"reorder_level"`
	SupplierID      *uuid.UUID `json:"supplier_id,omitempty"`
	LastReceivedAt  *time.Time `json:"last_received_at,omitempty"`
	LastDispatchedAt *time.Time `json:"last_dispatched_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type StockMovement struct {
	ID           uuid.UUID         `json:"id"`
	WarehouseID  uuid.UUID         `json:"warehouse_id"`
	StockItemID  uuid.UUID         `json:"stock_item_id"`
	MovementType StockMovementType `json:"movement_type"`
	Quantity     float64           `json:"quantity"`
	RefID        *uuid.UUID        `json:"ref_id,omitempty"`
	RefType      string            `json:"ref_type,omitempty"`
	Notes        string            `json:"notes,omitempty"`
	RecordedBy   uuid.UUID         `json:"recorded_by"`
	CreatedAt    time.Time         `json:"created_at"`
}

type WarehouseAuditLog struct {
	ID        uuid.UUID  `json:"id"`
	EntityID  *uuid.UUID `json:"entity_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Action    string     `json:"action"`
	Resource  string     `json:"resource"`
	IPAddress string     `json:"ip_address"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateWarehouseRequest struct {
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	Country      string  `json:"country"`
	Region       string  `json:"region"`
	Address      string  `json:"address"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	CapacityM3   float64 `json:"capacity_m3"`
	ManagerName  string  `json:"manager_name"`
	ContactPhone string  `json:"contact_phone"`
	Notes        string  `json:"notes,omitempty"`
}

type CreateStockItemRequest struct {
	SKU          string  `json:"sku"`
	ProductName  string  `json:"product_name"`
	Category     string  `json:"category"`
	BinLocation  string  `json:"bin_location"`
	Unit         string  `json:"unit"`
	UnitWeightKg float64 `json:"unit_weight_kg"`
	UnitVolumeM3 float64 `json:"unit_volume_m3"`
	ReorderLevel float64 `json:"reorder_level"`
}

type StockMovementRequest struct {
	StockItemID  string            `json:"stock_item_id"`
	MovementType StockMovementType `json:"movement_type"`
	Quantity     float64           `json:"quantity"`
	RefID        string            `json:"ref_id,omitempty"`
	RefType      string            `json:"ref_type,omitempty"`
	Notes        string            `json:"notes,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
