package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/farmer-service/models"
)

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

// ─── Farmers ──────────────────────────────────────────────────────────────────

func (q *Queries) CreateFarmer(ctx context.Context, row models.FarmerRow) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO farmers
			(id, user_id, full_name_enc, phone_enc, national_id_enc,
			 country, region, district, gps_lat, gps_lng,
			 farm_size_ha, farm_size, primary_language, is_active, cooperative_id,
			 created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		row.ID, row.UserID, row.FullNameEnc, row.PhoneEnc, row.NationalIDEnc,
		row.Country, row.Region, row.District, row.GPSLat, row.GPSLng,
		row.FarmSizeHa, row.FarmSize, row.PrimaryLanguage, row.IsActive, row.CooperativeID,
		row.CreatedAt, row.UpdatedAt,
	)
	return err
}

func (q *Queries) GetFarmer(ctx context.Context, id uuid.UUID) (*models.FarmerRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, user_id, full_name_enc, phone_enc, national_id_enc,
		       country, region, district, gps_lat, gps_lng,
		       farm_size_ha, farm_size, primary_language, is_verified, is_active, cooperative_id,
		       created_at, updated_at
		FROM farmers WHERE id = $1`, id)
	return scanFarmerRow(row)
}

type ListFarmersParams struct {
	Country       *string
	Region        *string
	CooperativeID *uuid.UUID
	IsActive      *bool
	Page          int
	Limit         int
}

func (q *Queries) ListFarmers(ctx context.Context, p ListFarmersParams) ([]models.FarmerRow, error) {
	where, args := buildFarmerWhere(p)
	offset := (p.Page - 1) * p.Limit
	n := len(args) + 1
	args = append(args, p.Limit, offset)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, user_id, full_name_enc, phone_enc, national_id_enc,
		       country, region, district, gps_lat, gps_lng,
		       farm_size_ha, farm_size, primary_language, is_verified, is_active, cooperative_id,
		       created_at, updated_at
		FROM farmers %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, n, n+1), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.FarmerRow
	for rows.Next() {
		r, err := scanFarmerRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *r)
	}
	return result, rows.Err()
}

func (q *Queries) CountFarmers(ctx context.Context, p ListFarmersParams) (int, error) {
	where, args := buildFarmerWhere(p)
	var total int
	err := q.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM farmers %s`, where), args...).Scan(&total)
	return total, err
}

func buildFarmerWhere(p ListFarmersParams) (string, []interface{}) {
	where := "WHERE 1=1"
	var args []interface{}
	n := 1
	if p.Country != nil {
		where += fmt.Sprintf(" AND country = $%d", n)
		args = append(args, *p.Country)
		n++
	}
	if p.Region != nil {
		where += fmt.Sprintf(" AND region = $%d", n)
		args = append(args, *p.Region)
		n++
	}
	if p.CooperativeID != nil {
		where += fmt.Sprintf(" AND cooperative_id = $%d", n)
		args = append(args, *p.CooperativeID)
		n++
	}
	if p.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", n)
		args = append(args, *p.IsActive)
	}
	return where, args
}

type UpdateFarmerParams struct {
	ID              uuid.UUID
	PhoneEnc        *string
	Region          *string
	District        *string
	GPSLat          *float64
	GPSLng          *float64
	FarmSizeHa      *float64
	PrimaryLanguage *string
	CooperativeID   *uuid.UUID
	IsActive        *bool
	Now             time.Time
}

func (q *Queries) UpdateFarmer(ctx context.Context, p UpdateFarmerParams) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE farmers SET
			phone_enc        = COALESCE($1, phone_enc),
			region           = COALESCE($2, region),
			district         = COALESCE($3, district),
			gps_lat          = COALESCE($4, gps_lat),
			gps_lng          = COALESCE($5, gps_lng),
			farm_size_ha     = COALESCE($6, farm_size_ha),
			primary_language = COALESCE($7, primary_language),
			cooperative_id   = COALESCE($8, cooperative_id),
			is_active        = COALESCE($9, is_active),
			updated_at       = $10
		WHERE id = $11`,
		p.PhoneEnc, p.Region, p.District, p.GPSLat, p.GPSLng,
		p.FarmSizeHa, p.PrimaryLanguage, p.CooperativeID, p.IsActive,
		p.Now, p.ID,
	)
	return err
}

func (q *Queries) VerifyFarmer(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE farmers SET is_verified = TRUE, updated_at = $1 WHERE id = $2`, now, id)
	return err
}

// ─── Farm plots ───────────────────────────────────────────────────────────────

func (q *Queries) CreatePlot(ctx context.Context, plot models.FarmPlot) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO farm_plots (id, farmer_id, name, size_ha, soil_type, irrigation, gps_polygon, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		plot.ID, plot.FarmerID, plot.Name, plot.SizeHa, plot.SoilType,
		plot.Irrigation, plot.GPSPolygon, plot.CreatedAt, plot.UpdatedAt,
	)
	return err
}

func (q *Queries) ListPlots(ctx context.Context, farmerID uuid.UUID) ([]models.FarmPlot, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, farmer_id, name, size_ha, soil_type, irrigation, gps_polygon, current_crop, created_at, updated_at
		FROM farm_plots WHERE farmer_id = $1 ORDER BY created_at ASC`, farmerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.FarmPlot
	for rows.Next() {
		var p models.FarmPlot
		if err := rows.Scan(&p.ID, &p.FarmerID, &p.Name, &p.SizeHa, &p.SoilType,
			&p.Irrigation, &p.GPSPolygon, &p.CurrentCrop, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// ─── Crop records ─────────────────────────────────────────────────────────────

func (q *Queries) CreateCropRecord(ctx context.Context, crop models.CropRecord) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO crop_records
			(id, farmer_id, plot_id, crop_type, variety, area_ha,
			 planted_at, expected_harvest, status, notes, season, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		crop.ID, crop.FarmerID, crop.PlotID, crop.CropType, crop.Variety, crop.AreaHa,
		crop.PlantedAt, crop.ExpectedHarvest, crop.Status, crop.Notes, crop.Season,
		crop.CreatedAt, crop.UpdatedAt,
	)
	return err
}

func (q *Queries) ListCropRecords(ctx context.Context, farmerID uuid.UUID, page, limit int) ([]models.CropRecord, error) {
	offset := (page - 1) * limit
	rows, err := q.pool.Query(ctx, `
		SELECT id, farmer_id, plot_id, crop_type, variety, area_ha,
		       planted_at, expected_harvest, actual_harvest, yield_kg,
		       status, notes, season, created_at, updated_at
		FROM crop_records WHERE farmer_id = $1
		ORDER BY planted_at DESC LIMIT $2 OFFSET $3`, farmerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.CropRecord
	for rows.Next() {
		var c models.CropRecord
		if err := rows.Scan(&c.ID, &c.FarmerID, &c.PlotID, &c.CropType, &c.Variety, &c.AreaHa,
			&c.PlantedAt, &c.ExpectedHarvest, &c.ActualHarvest, &c.YieldKg,
			&c.Status, &c.Notes, &c.Season, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (q *Queries) UpdateCropRecord(ctx context.Context, id uuid.UUID, req models.UpdateCropRequest, now time.Time) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE crop_records SET
			status         = $1,
			actual_harvest = COALESCE($2, actual_harvest),
			yield_kg       = COALESCE($3, yield_kg),
			notes          = COALESCE($4, notes),
			updated_at     = $5
		WHERE id = $6`,
		req.Status, req.ActualHarvest, req.YieldKg, req.Notes, now, id,
	)
	return err
}

// ─── Audit log ────────────────────────────────────────────────────────────────

func (q *Queries) InsertAuditLog(ctx context.Context, log models.FarmerAuditLog) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO farmer_audit_log (id, farmer_id, user_id, action, resource, ip_address, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		log.ID, log.FarmerID, log.UserID, log.Action, log.Resource, log.IPAddress, log.CreatedAt,
	)
	return err
}

// ─── Row scanners ─────────────────────────────────────────────────────────────

type scannable interface {
	Scan(dest ...any) error
}

func scanFarmerRow(row scannable) (*models.FarmerRow, error) {
	var f models.FarmerRow
	err := row.Scan(
		&f.ID, &f.UserID, &f.FullNameEnc, &f.PhoneEnc, &f.NationalIDEnc,
		&f.Country, &f.Region, &f.District, &f.GPSLat, &f.GPSLng,
		&f.FarmSizeHa, &f.FarmSize, &f.PrimaryLanguage, &f.IsVerified, &f.IsActive,
		&f.CooperativeID, &f.CreatedAt, &f.UpdatedAt,
	)
	return &f, err
}
