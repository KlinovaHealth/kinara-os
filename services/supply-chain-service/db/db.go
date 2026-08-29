package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/supply-chain-service/models"
)

var ErrNotFound = errors.New("db: record not found")

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateShipment(ctx context.Context, s models.Shipment) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO shipments
		 (id,shipment_ref,farmer_id,cooperative_id,commodity_name,quantity_kg,origin_location,
		  destination_location,buyer_id,status,pillar_handoff,estimated_cost_usd,created_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13)`,
		s.ID, s.ShipmentRef, s.FarmerID, s.CooperativeID, s.CommodityName, s.QuantityKg,
		s.OriginLocation, s.DestLocation, s.BuyerID, s.Status, s.PillarHandoff,
		s.EstimatedCostUSD, s.CreatedAt)
	return err
}

func (q *Queries) GetShipment(ctx context.Context, id uuid.UUID) (*models.Shipment, error) {
	var s models.Shipment
	err := q.pool.QueryRow(ctx,
		`SELECT id,shipment_ref,farmer_id,cooperative_id,commodity_name,quantity_kg,origin_location,
		        destination_location,buyer_id,status,pillar_handoff,estimated_cost_usd,actual_cost_usd,
		        picked_up_at,delivered_at,created_at,updated_at
		 FROM shipments WHERE id=$1`, id).
		Scan(&s.ID, &s.ShipmentRef, &s.FarmerID, &s.CooperativeID, &s.CommodityName, &s.QuantityKg,
			&s.OriginLocation, &s.DestLocation, &s.BuyerID, &s.Status, &s.PillarHandoff,
			&s.EstimatedCostUSD, &s.ActualCostUSD, &s.PickedUpAt, &s.DeliveredAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &s, nil
}

type ListShipmentsParams struct {
	FarmerID *uuid.UUID
	Status   *string
	Page     int
	Limit    int
}

func (q *Queries) ListShipments(ctx context.Context, p ListShipmentsParams) ([]models.Shipment, error) {
	query := `SELECT id,shipment_ref,farmer_id,cooperative_id,commodity_name,quantity_kg,origin_location,
	                 destination_location,buyer_id,status,pillar_handoff,estimated_cost_usd,actual_cost_usd,
	                 picked_up_at,delivered_at,created_at,updated_at
	          FROM shipments WHERE 1=1`
	args := []interface{}{}
	i := 1
	if p.FarmerID != nil {
		query += fmt.Sprintf(" AND farmer_id=$%d", i)
		args = append(args, *p.FarmerID)
		i++
	}
	if p.Status != nil {
		query += fmt.Sprintf(" AND status=$%d", i)
		args = append(args, *p.Status)
		i++
	}
	if p.Limit == 0 {
		p.Limit = 50
	}
	if p.Page < 1 {
		p.Page = 1
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, p.Limit, (p.Page-1)*p.Limit)

	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Shipment
	for rows.Next() {
		var s models.Shipment
		if err := rows.Scan(&s.ID, &s.ShipmentRef, &s.FarmerID, &s.CooperativeID, &s.CommodityName, &s.QuantityKg,
			&s.OriginLocation, &s.DestLocation, &s.BuyerID, &s.Status, &s.PillarHandoff,
			&s.EstimatedCostUSD, &s.ActualCostUSD, &s.PickedUpAt, &s.DeliveredAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func (q *Queries) UpdateShipmentStatus(ctx context.Context, id uuid.UUID, status models.ShipmentStatus, actualCost *float64, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE shipments SET status=$1, actual_cost_usd=COALESCE($2,actual_cost_usd),
		   picked_up_at=CASE WHEN $1='picked_up' THEN COALESCE(picked_up_at,$3) ELSE picked_up_at END,
		   delivered_at=CASE WHEN $1='delivered' THEN COALESCE(delivered_at,$3) ELSE delivered_at END,
		   updated_at=$3
		 WHERE id=$4`, status, actualCost, now, id)
	return err
}

func (q *Queries) AddTrackingEvent(ctx context.Context, e models.TrackingEvent) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO tracking_events (id,shipment_id,status,location,note,recorded_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		e.ID, e.ShipmentID, e.Status, e.Location, e.Note, e.RecordedAt)
	return err
}

func (q *Queries) ListTrackingEvents(ctx context.Context, shipmentID uuid.UUID) ([]models.TrackingEvent, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,shipment_id,status,location,note,recorded_at
		 FROM tracking_events WHERE shipment_id=$1 ORDER BY recorded_at ASC`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.TrackingEvent
	for rows.Next() {
		var e models.TrackingEvent
		if err := rows.Scan(&e.ID, &e.ShipmentID, &e.Status, &e.Location, &e.Note, &e.RecordedAt); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.SupplyAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO supply_audit_log (id,shipment_id,actor_id,action,detail,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		l.ID, l.ShipmentID, l.ActorID, l.Action, l.Detail, l.CreatedAt)
	return err
}
