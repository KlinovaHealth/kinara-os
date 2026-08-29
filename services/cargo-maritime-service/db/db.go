package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/cargo-maritime-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) RegisterContainer(ctx context.Context, c models.Container) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO containers (id,container_no,container_type,owner_id,status,weight_kg,tare_weight_kg,is_hazmat,hazmat_class,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		c.ID, c.ContainerNo, c.ContainerType, c.OwnerID, c.Status, c.WeightKg, c.TareWeightKg,
		c.IsHazmat, c.HazmatClass, c.CreatedAt, c.UpdatedAt)
	return err
}

func (q *Queries) GetContainer(ctx context.Context, id uuid.UUID) (*models.Container, error) {
	c := &models.Container{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,container_no,container_type,owner_id,status,current_port_id,vessel_id,weight_kg,tare_weight_kg,payload_kg,seal_no,temperature_c,is_hazmat,hazmat_class,created_at,updated_at
         FROM containers WHERE id=$1`, id).
		Scan(&c.ID, &c.ContainerNo, &c.ContainerType, &c.OwnerID, &c.Status, &c.CurrentPortID, &c.VesselID,
			&c.WeightKg, &c.TareWeightKg, &c.PayloadKg, &c.SealNo, &c.Temperature,
			&c.IsHazmat, &c.HazmatClass, &c.CreatedAt, &c.UpdatedAt)
	if err != nil { return nil, err }
	return c, nil
}

func (q *Queries) ListContainers(ctx context.Context, status *models.ContainerStatus, vesselID *uuid.UUID) ([]models.Container, error) {
	query := `SELECT id,container_no,container_type,owner_id,status,current_port_id,vessel_id,weight_kg,tare_weight_kg,payload_kg,seal_no,temperature_c,is_hazmat,hazmat_class,created_at,updated_at FROM containers WHERE 1=1`
	args := []interface{}{}
	i := 1
	if status != nil {
		query += " AND status=$1"
		args = append(args, *status)
		i++
	}
	if vesselID != nil {
		query += " AND vessel_id=$" + itoa(i)
		args = append(args, *vesselID)
	}
	query += " ORDER BY created_at DESC LIMIT 200"
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var containers []models.Container
	for rows.Next() {
		var c models.Container
		if err := rows.Scan(&c.ID, &c.ContainerNo, &c.ContainerType, &c.OwnerID, &c.Status, &c.CurrentPortID, &c.VesselID,
			&c.WeightKg, &c.TareWeightKg, &c.PayloadKg, &c.SealNo, &c.Temperature, &c.IsHazmat, &c.HazmatClass, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		containers = append(containers, c)
	}
	return containers, nil
}

func itoa(i int) string {
	return string(rune('0' + i))
}

func (q *Queries) UpdateContainerStatus(ctx context.Context, id uuid.UUID, status models.ContainerStatus, sealNo string, portID, vesselID *uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE containers SET status=$1,seal_no=COALESCE(NULLIF($2,''),seal_no),current_port_id=COALESCE($3,current_port_id),vessel_id=COALESCE($4,vessel_id),updated_at=$5 WHERE id=$6`,
		status, sealNo, portID, vesselID, now, id)
	return err
}

func (q *Queries) CreateManifest(ctx context.Context, m models.CargoManifest) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO cargo_manifests (id,manifest_no,voyage_id,vessel_id,port_of_loading,port_of_discharge,shipper_name,consignee_name,commodity,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		m.ID, m.ManifestNo, m.VoyageID, m.VesselID, m.PortOfLoading, m.PortOfDischarge,
		m.ShipperName, m.ConsigneeName, m.Commodity, m.CreatedAt, m.UpdatedAt)
	return err
}

func (q *Queries) GetManifest(ctx context.Context, id uuid.UUID) (*models.CargoManifest, error) {
	m := &models.CargoManifest{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,manifest_no,voyage_id,vessel_id,port_of_loading,port_of_discharge,shipper_name,consignee_name,total_containers,total_weight_kg,commodity,is_finalized,created_at,updated_at
         FROM cargo_manifests WHERE id=$1`, id).
		Scan(&m.ID, &m.ManifestNo, &m.VoyageID, &m.VesselID, &m.PortOfLoading, &m.PortOfDischarge,
			&m.ShipperName, &m.ConsigneeName, &m.TotalContainers, &m.TotalWeightKg, &m.Commodity,
			&m.IsFinalized, &m.CreatedAt, &m.UpdatedAt)
	if err != nil { return nil, err }
	return m, nil
}

func (q *Queries) AddContainerToManifest(ctx context.Context, mc models.ManifestContainer, weightKg float64) error {
	tx, err := q.pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx,
		`INSERT INTO manifest_containers (id,manifest_id,container_id,container_no,added_at) VALUES ($1,$2,$3,$4,$5)`,
		mc.ID, mc.ManifestID, mc.ContainerID, mc.ContainerNo, mc.AddedAt)
	if err != nil { return err }
	_, err = tx.Exec(ctx,
		`UPDATE cargo_manifests SET total_containers=total_containers+1,total_weight_kg=total_weight_kg+$1,updated_at=NOW() WHERE id=$2`,
		weightKg, mc.ManifestID)
	if err != nil { return err }
	return tx.Commit(ctx)
}

func (q *Queries) FinalizeManifest(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE cargo_manifests SET is_finalized=TRUE,updated_at=$1 WHERE id=$2`, now, id)
	return err
}

func (q *Queries) ReportDamage(ctx context.Context, d models.DamageReport) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO damage_reports (id,container_id,container_no,damage_level,description,photo_url,reported_by,estimated_cost,currency,port_id,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		d.ID, d.ContainerID, d.ContainerNo, d.DamageLevel, d.Description,
		d.PhotoURL, d.ReportedBy, d.EstimatedCost, d.Currency, d.PortID, d.CreatedAt)
	return err
}

func (q *Queries) ListDamageReports(ctx context.Context, containerID uuid.UUID) ([]models.DamageReport, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,container_id,container_no,damage_level,description,photo_url,reported_by,estimated_cost,currency,port_id,created_at
         FROM damage_reports WHERE container_id=$1 ORDER BY created_at DESC`, containerID)
	if err != nil { return nil, err }
	defer rows.Close()
	var reports []models.DamageReport
	for rows.Next() {
		var d models.DamageReport
		if err := rows.Scan(&d.ID, &d.ContainerID, &d.ContainerNo, &d.DamageLevel, &d.Description, &d.PhotoURL, &d.ReportedBy, &d.EstimatedCost, &d.Currency, &d.PortID, &d.CreatedAt); err != nil {
			return nil, err
		}
		reports = append(reports, d)
	}
	return reports, nil
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.CargoMaritimeAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO cargo_maritime_audit_log (id,actor_id,action,entity_type,entity_id,created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		l.ID, l.ActorID, l.Action, l.EntityType, l.EntityID, l.CreatedAt)
	return err
}
