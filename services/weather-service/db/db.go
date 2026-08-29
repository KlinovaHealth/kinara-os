package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/weather-service/models"
)

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

// ─── Forecasts ────────────────────────────────────────────────────────────────

func (q *Queries) CreateForecast(ctx context.Context, f models.WeatherForecast) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO weather_forecasts
			(id, country, region, district, latitude, longitude, forecast_type, forecast_date,
			 condition, temp_min_c, temp_max_c, temp_avg_c, humidity_pct, wind_speed_kmh, wind_direction,
			 rainfall_mm, rainfall_prob, uv_index, data_source, valid_until, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		f.ID, f.Country, f.Region, f.District, f.Latitude, f.Longitude,
		f.ForecastType, f.ForecastDate, f.Condition, f.TempMinC, f.TempMaxC, f.TempAvgC,
		f.HumidityPct, f.WindSpeedKmh, f.WindDirection, f.RainfallMm, f.RainfallProb,
		f.UVIndex, f.DataSource, f.ValidUntil, f.CreatedAt,
	)
	return err
}

func (q *Queries) GetForecast(ctx context.Context, id uuid.UUID) (*models.WeatherForecast, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, country, region, district, latitude, longitude, forecast_type, forecast_date,
		       condition, temp_min_c, temp_max_c, temp_avg_c, humidity_pct, wind_speed_kmh, wind_direction,
		       rainfall_mm, rainfall_prob, uv_index, data_source, valid_until, created_at
		FROM weather_forecasts WHERE id = $1`, id)
	return scanForecastRow(row)
}

type ListForecastsParams struct {
	Country  string
	Region   string
	DateFrom *time.Time
	DateTo   *time.Time
	Type     *models.ForecastType
	Page     int
	Limit    int
}

func (q *Queries) ListForecasts(ctx context.Context, p ListForecastsParams) ([]models.WeatherForecast, error) {
	where, args := buildForecastWhere(p)
	offset := (p.Page - 1) * p.Limit
	n := len(args) + 1
	args = append(args, p.Limit, offset)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, country, region, district, latitude, longitude, forecast_type, forecast_date,
		       condition, temp_min_c, temp_max_c, temp_avg_c, humidity_pct, wind_speed_kmh, wind_direction,
		       rainfall_mm, rainfall_prob, uv_index, data_source, valid_until, created_at
		FROM weather_forecasts %s ORDER BY forecast_date ASC LIMIT $%d OFFSET $%d`,
		where, n, n+1), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.WeatherForecast
	for rows.Next() {
		f, err := scanForecastRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *f)
	}
	return result, rows.Err()
}

func buildForecastWhere(p ListForecastsParams) (string, []interface{}) {
	where := "WHERE 1=1"
	var args []interface{}
	n := 1
	if p.Country != "" {
		where += fmt.Sprintf(" AND country = $%d", n)
		args = append(args, p.Country)
		n++
	}
	if p.Region != "" {
		where += fmt.Sprintf(" AND region = $%d", n)
		args = append(args, p.Region)
		n++
	}
	if p.DateFrom != nil {
		where += fmt.Sprintf(" AND forecast_date >= $%d", n)
		args = append(args, *p.DateFrom)
		n++
	}
	if p.DateTo != nil {
		where += fmt.Sprintf(" AND forecast_date <= $%d", n)
		args = append(args, *p.DateTo)
		n++
	}
	if p.Type != nil {
		where += fmt.Sprintf(" AND forecast_type = $%d", n)
		args = append(args, *p.Type)
	}
	return where, args
}

// ─── Alerts ───────────────────────────────────────────────────────────────────

func (q *Queries) CreateAlert(ctx context.Context, a models.WeatherAlert) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO weather_alerts
			(id, alert_type, severity, country, region, district, title, description,
			 instructions, affected_crops, issued_at, expires_at, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		a.ID, a.AlertType, a.Severity, a.Country, a.Region, a.District,
		a.Title, a.Description, a.Instructions, a.AffectedCrops,
		a.IssuedAt, a.ExpiresAt, a.Active, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (q *Queries) GetAlert(ctx context.Context, id uuid.UUID) (*models.WeatherAlert, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, alert_type, severity, country, region, district, title, description,
		       instructions, affected_crops, issued_at, expires_at, active, created_at, updated_at
		FROM weather_alerts WHERE id = $1`, id)
	return scanAlertRow(row)
}

func (q *Queries) ListActiveAlerts(ctx context.Context, country, region string) ([]models.WeatherAlert, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, alert_type, severity, country, region, district, title, description,
		       instructions, affected_crops, issued_at, expires_at, active, created_at, updated_at
		FROM weather_alerts
		WHERE active = true
		  AND ($1 = '' OR country = $1)
		  AND ($2 = '' OR region = $2)
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY CASE severity
			WHEN 'emergency' THEN 1
			WHEN 'warning'   THEN 2
			WHEN 'watch'     THEN 3
			ELSE 4
		END, issued_at DESC`,
		country, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.WeatherAlert
	for rows.Next() {
		a, err := scanAlertRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *a)
	}
	return result, rows.Err()
}

func (q *Queries) DeactivateAlert(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE weather_alerts SET active = false, updated_at = $1 WHERE id = $2`, now, id)
	return err
}

// ─── Pest/Disease advisories ──────────────────────────────────────────────────

func (q *Queries) CreateAdvisory(ctx context.Context, a models.PestAdvisory) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO pest_advisories
			(id, pest_name, pest_type, affected_crops, country, region, risk_level,
			 description, symptoms, prevention, treatment, reported_cases,
			 valid_from, valid_until, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		a.ID, a.PestName, a.PestType, a.AffectedCrops, a.Country, a.Region,
		a.RiskLevel, a.Description, a.Symptoms, a.Prevention, a.Treatment,
		a.ReportedCases, a.ValidFrom, a.ValidUntil, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (q *Queries) GetAdvisory(ctx context.Context, id uuid.UUID) (*models.PestAdvisory, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, pest_name, pest_type, affected_crops, country, region, risk_level,
		       description, symptoms, prevention, treatment, reported_cases,
		       valid_from, valid_until, created_at, updated_at
		FROM pest_advisories WHERE id = $1`, id)
	return scanAdvisoryRow(row)
}

func (q *Queries) ListAdvisories(ctx context.Context, country, region, cropType string, riskLevel *models.RiskLevel) ([]models.PestAdvisory, error) {
	var riskFilter string
	var args []interface{}
	n := 1
	where := "WHERE (valid_until IS NULL OR valid_until > NOW())"
	if country != "" {
		where += fmt.Sprintf(" AND country = $%d", n)
		args = append(args, country)
		n++
	}
	if region != "" {
		where += fmt.Sprintf(" AND region = $%d", n)
		args = append(args, region)
		n++
	}
	if cropType != "" {
		where += fmt.Sprintf(" AND $%d = ANY(affected_crops)", n)
		args = append(args, cropType)
		n++
	}
	if riskLevel != nil {
		riskFilter = string(*riskLevel)
		where += fmt.Sprintf(" AND risk_level = $%d", n)
		args = append(args, riskFilter)
	}
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, pest_name, pest_type, affected_crops, country, region, risk_level,
		       description, symptoms, prevention, treatment, reported_cases,
		       valid_from, valid_until, created_at, updated_at
		FROM pest_advisories %s
		ORDER BY CASE risk_level
			WHEN 'critical' THEN 1
			WHEN 'high'     THEN 2
			WHEN 'moderate' THEN 3
			ELSE 4
		END, created_at DESC`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.PestAdvisory
	for rows.Next() {
		a, err := scanAdvisoryRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *a)
	}
	return result, rows.Err()
}

// ─── Observations ─────────────────────────────────────────────────────────────

func (q *Queries) CreateObservation(ctx context.Context, o models.WeatherObservation) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO weather_observations
			(id, reporter_id, country, region, district, latitude, longitude, observed_at,
			 temp_c, rainfall_mm, humidity_pct, wind_speed_kmh, condition, notes, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		o.ID, o.ReporterID, o.Country, o.Region, o.District, o.Latitude, o.Longitude,
		o.ObservedAt, o.TempC, o.RainfallMm, o.HumidityPct, o.WindSpeedKmh,
		o.Condition, o.Notes, o.CreatedAt,
	)
	return err
}

func (q *Queries) ListObservations(ctx context.Context, country, region string, from time.Time, limit int) ([]models.WeatherObservation, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, reporter_id, country, region, district, latitude, longitude, observed_at,
		       temp_c, rainfall_mm, humidity_pct, wind_speed_kmh, condition, notes, created_at
		FROM weather_observations
		WHERE ($1 = '' OR country = $1) AND ($2 = '' OR region = $2) AND observed_at >= $3
		ORDER BY observed_at DESC LIMIT $4`,
		country, region, from, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.WeatherObservation
	for rows.Next() {
		o, err := scanObservationRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *o)
	}
	return result, rows.Err()
}

// ─── Audit log ────────────────────────────────────────────────────────────────

func (q *Queries) InsertAuditLog(ctx context.Context, log models.WeatherAuditLog) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO weather_audit_log (id, entity_id, user_id, action, resource, ip_address, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		log.ID, log.EntityID, log.UserID, log.Action, log.Resource, log.IPAddress, log.CreatedAt,
	)
	return err
}

// ─── Row scanners ─────────────────────────────────────────────────────────────

type scannable interface{ Scan(dest ...any) error }

func scanForecastRow(row scannable) (*models.WeatherForecast, error) {
	var f models.WeatherForecast
	err := row.Scan(
		&f.ID, &f.Country, &f.Region, &f.District, &f.Latitude, &f.Longitude,
		&f.ForecastType, &f.ForecastDate, &f.Condition, &f.TempMinC, &f.TempMaxC, &f.TempAvgC,
		&f.HumidityPct, &f.WindSpeedKmh, &f.WindDirection, &f.RainfallMm, &f.RainfallProb,
		&f.UVIndex, &f.DataSource, &f.ValidUntil, &f.CreatedAt,
	)
	return &f, err
}

func scanAlertRow(row scannable) (*models.WeatherAlert, error) {
	var a models.WeatherAlert
	err := row.Scan(&a.ID, &a.AlertType, &a.Severity, &a.Country, &a.Region, &a.District,
		&a.Title, &a.Description, &a.Instructions, &a.AffectedCrops,
		&a.IssuedAt, &a.ExpiresAt, &a.Active, &a.CreatedAt, &a.UpdatedAt)
	return &a, err
}

func scanAdvisoryRow(row scannable) (*models.PestAdvisory, error) {
	var a models.PestAdvisory
	err := row.Scan(&a.ID, &a.PestName, &a.PestType, &a.AffectedCrops, &a.Country, &a.Region,
		&a.RiskLevel, &a.Description, &a.Symptoms, &a.Prevention, &a.Treatment,
		&a.ReportedCases, &a.ValidFrom, &a.ValidUntil, &a.CreatedAt, &a.UpdatedAt)
	return &a, err
}

func scanObservationRow(row scannable) (*models.WeatherObservation, error) {
	var o models.WeatherObservation
	err := row.Scan(&o.ID, &o.ReporterID, &o.Country, &o.Region, &o.District,
		&o.Latitude, &o.Longitude, &o.ObservedAt, &o.TempC, &o.RainfallMm,
		&o.HumidityPct, &o.WindSpeedKmh, &o.Condition, &o.Notes, &o.CreatedAt)
	return &o, err
}
