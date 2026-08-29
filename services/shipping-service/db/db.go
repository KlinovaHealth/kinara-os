package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/shipping-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateBooking(ctx context.Context, b models.FreightBooking) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO freight_bookings (id,booking_ref,shipper_id,shipper_name,consignee_name,shipment_type,port_of_loading,port_of_discharge,commodity_description,container_count,weight_kg,freight_rate_usd,insurance_pct,insurance_amount,declared_value,total_freight,currency,status,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		b.ID, b.BookingRef, b.ShipperID, b.ShipperName, b.ConsigneeName, b.ShipmentType,
		b.PortOfLoading, b.PortOfDischarge, b.CommodityDesc, b.ContainerCount,
		b.WeightKg, b.FreightRate, b.InsurancePct, b.InsuranceAmount, b.DeclaredValue,
		b.TotalFreight, b.Currency, b.Status, b.CreatedAt, b.UpdatedAt)
	return err
}

func (q *Queries) GetBooking(ctx context.Context, id uuid.UUID) (*models.FreightBooking, error) {
	b := &models.FreightBooking{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,booking_ref,shipper_id,shipper_name,consignee_name,shipment_type,port_of_loading,port_of_discharge,vessel_id,commodity_description,container_count,weight_kg,freight_rate_usd,insurance_pct,insurance_amount,declared_value,total_freight,currency,status,created_at,updated_at
         FROM freight_bookings WHERE id=$1`, id).
		Scan(&b.ID, &b.BookingRef, &b.ShipperID, &b.ShipperName, &b.ConsigneeName, &b.ShipmentType,
			&b.PortOfLoading, &b.PortOfDischarge, &b.VesselID, &b.CommodityDesc, &b.ContainerCount,
			&b.WeightKg, &b.FreightRate, &b.InsurancePct, &b.InsuranceAmount, &b.DeclaredValue,
			&b.TotalFreight, &b.Currency, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil { return nil, err }
	return b, nil
}

func (q *Queries) ListBookings(ctx context.Context, shipperID *uuid.UUID, status *models.FreightStatus) ([]models.FreightBooking, error) {
	query := `SELECT id,booking_ref,shipper_id,shipper_name,consignee_name,shipment_type,port_of_loading,port_of_discharge,vessel_id,commodity_description,container_count,weight_kg,freight_rate_usd,insurance_pct,insurance_amount,declared_value,total_freight,currency,status,created_at,updated_at FROM freight_bookings WHERE 1=1`
	args := []interface{}{}
	i := 1
	if shipperID != nil { query += " AND shipper_id=$1"; args = append(args, *shipperID); i++ }
	if status != nil {
		if i == 1 { query += " AND status=$1" } else { query += " AND status=$2" }
		args = append(args, *status)
	}
	query += " ORDER BY created_at DESC LIMIT 100"
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var bookings []models.FreightBooking
	for rows.Next() {
		var b models.FreightBooking
		if err := rows.Scan(&b.ID, &b.BookingRef, &b.ShipperID, &b.ShipperName, &b.ConsigneeName, &b.ShipmentType,
			&b.PortOfLoading, &b.PortOfDischarge, &b.VesselID, &b.CommodityDesc, &b.ContainerCount,
			&b.WeightKg, &b.FreightRate, &b.InsurancePct, &b.InsuranceAmount, &b.DeclaredValue,
			&b.TotalFreight, &b.Currency, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, nil
}

func (q *Queries) UpdateBookingStatus(ctx context.Context, id uuid.UUID, status models.FreightStatus, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE freight_bookings SET status=$1,updated_at=$2 WHERE id=$3`, status, now, id)
	return err
}

func (q *Queries) IssueBOL(ctx context.Context, bol models.BillOfLading) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO bills_of_lading (id,bol_number,booking_id,vessel_name,voyage_no,shipper_name,consignee_name,notify_party,port_of_loading,port_of_discharge,commodity_description,container_count,gross_weight_kg,freight_prepaid,status,issued_at,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		bol.ID, bol.BOLNumber, bol.BookingID, bol.VesselName, bol.VoyageNo,
		bol.ShipperName, bol.ConsigneeName, bol.NotifyParty, bol.POL, bol.POD,
		bol.CommodityDesc, bol.ContainerCount, bol.GrossWeightKg, bol.FreightPrepaid,
		bol.Status, bol.IssuedAt, bol.CreatedAt, bol.UpdatedAt)
	return err
}

func (q *Queries) GetBOL(ctx context.Context, id uuid.UUID) (*models.BillOfLading, error) {
	bol := &models.BillOfLading{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,bol_number,booking_id,vessel_name,voyage_no,shipper_name,consignee_name,notify_party,port_of_loading,port_of_discharge,commodity_description,container_count,gross_weight_kg,freight_prepaid,status,issued_at,surrendered_at,created_at,updated_at
         FROM bills_of_lading WHERE id=$1`, id).
		Scan(&bol.ID, &bol.BOLNumber, &bol.BookingID, &bol.VesselName, &bol.VoyageNo,
			&bol.ShipperName, &bol.ConsigneeName, &bol.NotifyParty, &bol.POL, &bol.POD,
			&bol.CommodityDesc, &bol.ContainerCount, &bol.GrossWeightKg, &bol.FreightPrepaid,
			&bol.Status, &bol.IssuedAt, &bol.SurrenderedAt, &bol.CreatedAt, &bol.UpdatedAt)
	if err != nil { return nil, err }
	return bol, nil
}

func (q *Queries) SurrenderBOL(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE bills_of_lading SET status='surrendered',surrendered_at=COALESCE(surrendered_at,$1),updated_at=$2 WHERE id=$3`, now, now, id)
	return err
}

func (q *Queries) RecordDemurrage(ctx context.Context, d models.DemurrageRecord) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO demurrage_records (id,booking_id,container_no,free_days,used_days,daily_rate_usd,total_charge,currency,port_id,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		d.ID, d.BookingID, d.ContainerNo, d.FreeDays, d.UsedDays, d.DailyRate,
		d.TotalCharge, d.Currency, d.PortID, d.CreatedAt)
	return err
}

func (q *Queries) ListDemurrage(ctx context.Context, bookingID uuid.UUID) ([]models.DemurrageRecord, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,booking_id,container_no,free_days,used_days,daily_rate_usd,total_charge,currency,port_id,created_at
         FROM demurrage_records WHERE booking_id=$1 ORDER BY created_at DESC`, bookingID)
	if err != nil { return nil, err }
	defer rows.Close()
	var records []models.DemurrageRecord
	for rows.Next() {
		var d models.DemurrageRecord
		if err := rows.Scan(&d.ID, &d.BookingID, &d.ContainerNo, &d.FreeDays, &d.UsedDays, &d.DailyRate, &d.TotalCharge, &d.Currency, &d.PortID, &d.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, d)
	}
	return records, nil
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.ShippingAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO shipping_audit_log (id,actor_id,action,entity_type,entity_id,created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		l.ID, l.ActorID, l.Action, l.EntityType, l.EntityID, l.CreatedAt)
	return err
}
