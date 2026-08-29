package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/logistics-analytics-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) RecordMetric(ctx context.Context, m models.LogisticsMetric) error {
	onTimeRate := 0.0
	if m.TotalDeliveries > 0 { onTimeRate = float64(m.OnTimeDeliveries) / float64(m.TotalDeliveries) * 100 }
	avgCostPerDelivery := 0.0
	if m.SuccessfulDeliveries > 0 { avgCostPerDelivery = m.TotalRevenue / float64(m.SuccessfulDeliveries) }
	_, err := q.pool.Exec(ctx, `INSERT INTO logistics_metrics(id,period,period_start,period_end,country,total_trips,total_distance_km,total_deliveries,successful_deliveries,on_time_deliveries,on_time_rate,avg_cost_per_km,avg_cost_per_delivery,total_revenue,currency,bottleneck_route,bottleneck_warehouse,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		m.ID,m.Period,m.PeriodStart,m.PeriodEnd,m.Country,m.TotalTrips,m.TotalDistanceKm,m.TotalDeliveries,m.SuccessfulDeliveries,m.OnTimeDeliveries,onTimeRate,m.AvgCostPerKm,avgCostPerDelivery,m.TotalRevenue,m.Currency,m.BottleneckRoute,m.BottleneckWarehouse,m.CreatedAt)
	return err
}

type ListMetricsParams struct{ Country *string; Period *models.MetricPeriod; Page, Limit int }

func (q *Queries) ListMetrics(ctx context.Context, p ListMetricsParams) ([]models.LogisticsMetric, error) {
	where := "WHERE 1=1"; var args []interface{}; n := 1
	if p.Country != nil { where += fmt.Sprintf(" AND country=$%d",n); args=append(args,*p.Country); n++ }
	if p.Period != nil { where += fmt.Sprintf(" AND period=$%d",n); args=append(args,*p.Period); n++ }
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT id,period,period_start,period_end,country,total_trips,total_distance_km,total_deliveries,successful_deliveries,on_time_deliveries,on_time_rate,avg_cost_per_km,avg_cost_per_delivery,total_revenue,currency,bottleneck_route,bottleneck_warehouse,created_at FROM logistics_metrics %s ORDER BY period_start DESC LIMIT $%d OFFSET $%d`,where,n,n+1),args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.LogisticsMetric
	for rows.Next() {
		var m models.LogisticsMetric
		if err := rows.Scan(&m.ID,&m.Period,&m.PeriodStart,&m.PeriodEnd,&m.Country,&m.TotalTrips,&m.TotalDistanceKm,&m.TotalDeliveries,&m.SuccessfulDeliveries,&m.OnTimeDeliveries,&m.OnTimeRate,&m.AvgCostPerKm,&m.AvgCostPerDelivery,&m.TotalRevenue,&m.Currency,&m.BottleneckRoute,&m.BottleneckWarehouse,&m.CreatedAt); err != nil { return nil, err }
		result = append(result, m)
	}
	return result, rows.Err()
}

func (q *Queries) CreateForecast(ctx context.Context, f models.DemandForecast) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO demand_forecasts(id,country,route,forecast_date,predicted_volume,predicted_trips,confidence_pct,notes,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		f.ID,f.Country,f.Route,f.ForecastDate,f.PredictedVolume,f.PredictedTrips,f.ConfidencePct,f.Notes,f.CreatedAt)
	return err
}

func (q *Queries) GetForecast(ctx context.Context, id uuid.UUID) (*models.DemandForecast, error) {
	row := q.pool.QueryRow(ctx, `SELECT id,country,route,forecast_date,predicted_volume,predicted_trips,confidence_pct,notes,created_at FROM demand_forecasts WHERE id=$1`, id)
	var f models.DemandForecast
	err := row.Scan(&f.ID,&f.Country,&f.Route,&f.ForecastDate,&f.PredictedVolume,&f.PredictedTrips,&f.ConfidencePct,&f.Notes,&f.CreatedAt)
	return &f, err
}

func (q *Queries) ListForecasts(ctx context.Context, country string) ([]models.DemandForecast, error) {
	rows, err := q.pool.Query(ctx, `SELECT id,country,route,forecast_date,predicted_volume,predicted_trips,confidence_pct,notes,created_at FROM demand_forecasts WHERE country=$1 ORDER BY forecast_date DESC LIMIT 30`, country)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []models.DemandForecast
	for rows.Next() {
		var f models.DemandForecast
		if err := rows.Scan(&f.ID,&f.Country,&f.Route,&f.ForecastDate,&f.PredictedVolume,&f.PredictedTrips,&f.ConfidencePct,&f.Notes,&f.CreatedAt); err != nil { return nil, err }
		result = append(result, f)
	}
	return result, rows.Err()
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.AnalyticsAuditLog) error {
	_, err := q.pool.Exec(ctx, `INSERT INTO analytics_audit_log(id,entity_id,user_id,action,resource,ip_address,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		l.ID,l.EntityID,l.UserID,l.Action,l.Resource,l.IPAddress,l.CreatedAt)
	return err
}
