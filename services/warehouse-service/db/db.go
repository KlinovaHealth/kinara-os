package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/warehouse-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateWarehouse(ctx context.Context, w models.Warehouse) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO warehouses(id,name,code,country,region,address,latitude,longitude,capacity_m3,used_m3,status,manager_name,contact_phone,notes,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		w.ID,w.Name,w.Code,w.Country,w.Region,w.Address,w.Latitude,w.Longitude,w.CapacityM3,w.UsedM3,w.Status,w.ManagerName,w.ContactPhone,w.Notes,w.CreatedAt,w.UpdatedAt)
	return err
}

func (q *Queries) GetWarehouse(ctx context.Context, id uuid.UUID) (*models.Warehouse, error) {
	row := q.pool.QueryRow(ctx, `SELECT id,name,code,country,region,address,latitude,longitude,capacity_m3,used_m3,status,manager_name,contact_phone,notes,created_at,updated_at FROM warehouses WHERE id=$1`, id)
	var w models.Warehouse
	err := row.Scan(&w.ID,&w.Name,&w.Code,&w.Country,&w.Region,&w.Address,&w.Latitude,&w.Longitude,&w.CapacityM3,&w.UsedM3,&w.Status,&w.ManagerName,&w.ContactPhone,&w.Notes,&w.CreatedAt,&w.UpdatedAt)
	return &w, err
}

type ListWarehouseParams struct{ Country *string; Page, Limit int }

func (q *Queries) ListWarehouses(ctx context.Context, p ListWarehouseParams) ([]models.Warehouse, error) {
	where := "WHERE 1=1"; var args []interface{}; n := 1
	if p.Country != nil { where += fmt.Sprintf(" AND country=$%d",n); args=append(args,*p.Country); n++ }
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT id,name,code,country,region,address,latitude,longitude,capacity_m3,used_m3,status,manager_name,contact_phone,notes,created_at,updated_at FROM warehouses %s ORDER BY name LIMIT $%d OFFSET $%d`,where,n,n+1),args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.Warehouse
	for rows.Next() {
		var w models.Warehouse
		if err := rows.Scan(&w.ID,&w.Name,&w.Code,&w.Country,&w.Region,&w.Address,&w.Latitude,&w.Longitude,&w.CapacityM3,&w.UsedM3,&w.Status,&w.ManagerName,&w.ContactPhone,&w.Notes,&w.CreatedAt,&w.UpdatedAt); err != nil { return nil, err }
		result = append(result, w)
	}
	return result, rows.Err()
}

func (q *Queries) CreateStockItem(ctx context.Context, s models.StockItem) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO stock_items(id,warehouse_id,sku,product_name,category,bin_location,quantity_on_hand,unit,unit_weight_kg,unit_volume_m3,reorder_level,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		s.ID,s.WarehouseID,s.SKU,s.ProductName,s.Category,s.BinLocation,s.QuantityOnHand,s.Unit,s.UnitWeightKg,s.UnitVolumeM3,s.ReorderLevel,s.CreatedAt,s.UpdatedAt)
	return err
}

func (q *Queries) GetStockItem(ctx context.Context, id uuid.UUID) (*models.StockItem, error) {
	row := q.pool.QueryRow(ctx, `SELECT id,warehouse_id,sku,product_name,category,bin_location,quantity_on_hand,unit,unit_weight_kg,unit_volume_m3,reorder_level,supplier_id,last_received_at,last_dispatched_at,created_at,updated_at FROM stock_items WHERE id=$1`, id)
	var s models.StockItem
	err := row.Scan(&s.ID,&s.WarehouseID,&s.SKU,&s.ProductName,&s.Category,&s.BinLocation,&s.QuantityOnHand,&s.Unit,&s.UnitWeightKg,&s.UnitVolumeM3,&s.ReorderLevel,&s.SupplierID,&s.LastReceivedAt,&s.LastDispatchedAt,&s.CreatedAt,&s.UpdatedAt)
	return &s, err
}

func (q *Queries) ListStockItems(ctx context.Context, warehouseID uuid.UUID) ([]models.StockItem, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,warehouse_id,sku,product_name,category,bin_location,quantity_on_hand,unit,unit_weight_kg,unit_volume_m3,reorder_level,supplier_id,last_received_at,last_dispatched_at,created_at,updated_at FROM stock_items WHERE warehouse_id=$1 ORDER BY product_name`, warehouseID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.StockItem
	for rows.Next() {
		var s models.StockItem
		if err := rows.Scan(&s.ID,&s.WarehouseID,&s.SKU,&s.ProductName,&s.Category,&s.BinLocation,&s.QuantityOnHand,&s.Unit,&s.UnitWeightKg,&s.UnitVolumeM3,&s.ReorderLevel,&s.SupplierID,&s.LastReceivedAt,&s.LastDispatchedAt,&s.CreatedAt,&s.UpdatedAt); err != nil { return nil, err }
		result = append(result, s)
	}
	return result, rows.Err()
}

func (q *Queries) ListLowStock(ctx context.Context, warehouseID uuid.UUID) ([]models.StockItem, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,warehouse_id,sku,product_name,category,bin_location,quantity_on_hand,unit,unit_weight_kg,unit_volume_m3,reorder_level,supplier_id,last_received_at,last_dispatched_at,created_at,updated_at FROM stock_items WHERE warehouse_id=$1 AND quantity_on_hand <= reorder_level ORDER BY quantity_on_hand ASC`, warehouseID)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.StockItem
	for rows.Next() {
		var s models.StockItem
		if err := rows.Scan(&s.ID,&s.WarehouseID,&s.SKU,&s.ProductName,&s.Category,&s.BinLocation,&s.QuantityOnHand,&s.Unit,&s.UnitWeightKg,&s.UnitVolumeM3,&s.ReorderLevel,&s.SupplierID,&s.LastReceivedAt,&s.LastDispatchedAt,&s.CreatedAt,&s.UpdatedAt); err != nil { return nil, err }
		result = append(result, s)
	}
	return result, rows.Err()
}

func (q *Queries) RecordMovement(ctx context.Context, m models.StockMovement, now time.Time) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO stock_movements(id,warehouse_id,stock_item_id,movement_type,quantity,ref_id,ref_type,notes,recorded_by,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		m.ID,m.WarehouseID,m.StockItemID,m.MovementType,m.Quantity,m.RefID,m.RefType,m.Notes,m.RecordedBy,m.CreatedAt)
	if err != nil { return err }
	if m.MovementType == models.MovementReceive {
		_, err = q.pool.Exec(ctx, `UPDATE stock_items SET quantity_on_hand=quantity_on_hand+$1,last_received_at=$2,updated_at=$2 WHERE id=$3`, m.Quantity,now,m.StockItemID)
	} else if m.MovementType == models.MovementDispatch {
		_, err = q.pool.Exec(ctx, `UPDATE stock_items SET quantity_on_hand=quantity_on_hand-$1,last_dispatched_at=$2,updated_at=$2 WHERE id=$3`, m.Quantity,now,m.StockItemID)
	}
	return err
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.WarehouseAuditLog) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO warehouse_audit_log(id,entity_id,user_id,action,resource,ip_address,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		l.ID,l.EntityID,l.UserID,l.Action,l.Resource,l.IPAddress,l.CreatedAt)
	return err
}
