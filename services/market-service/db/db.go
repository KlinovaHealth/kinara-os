package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/market-service/models"
)

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

// ─── Listings ─────────────────────────────────────────────────────────────────

func (q *Queries) CreateListing(ctx context.Context, l models.MarketListing) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO market_listings
			(id, farmer_id, crop_type, variety, quantity_kg, quantity_available,
			 price_per_unit, currency, price_unit, quality_grade, country, region, market,
			 harvested_at, available_from, available_until, status, description, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		l.ID, l.FarmerID, l.CropType, l.Variety, l.QuantityKg, l.QuantityAvail,
		l.PricePerUnit, l.Currency, l.PriceUnit, l.QualityGrade,
		l.Country, l.Region, l.Market, l.HarvestedAt, l.AvailableFrom,
		l.AvailableUntil, l.Status, l.Description, l.CreatedAt, l.UpdatedAt,
	)
	return err
}

func (q *Queries) GetListing(ctx context.Context, id uuid.UUID) (*models.MarketListing, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, farmer_id, crop_type, variety, quantity_kg, quantity_available,
		       price_per_unit, currency, price_unit, quality_grade, country, region, market,
		       harvested_at, available_from, available_until, status, description, created_at, updated_at
		FROM market_listings WHERE id = $1`, id)
	return scanListingRow(row)
}

type ListListingsParams struct {
	CropType    *string
	Country     *string
	Region      *string
	MaxPrice    *float64
	MinQuantity *float64
	FarmerID    *uuid.UUID
	Status      *models.ListingStatus
	Page        int
	Limit       int
}

func (q *Queries) ListListings(ctx context.Context, p ListListingsParams) ([]models.MarketListing, error) {
	where, args := buildListingWhere(p)
	offset := (p.Page - 1) * p.Limit
	n := len(args) + 1
	args = append(args, p.Limit, offset)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, farmer_id, crop_type, variety, quantity_kg, quantity_available,
		       price_per_unit, currency, price_unit, quality_grade, country, region, market,
		       harvested_at, available_from, available_until, status, description, created_at, updated_at
		FROM market_listings %s
		ORDER BY price_per_unit ASC, created_at DESC LIMIT $%d OFFSET $%d`,
		where, n, n+1), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.MarketListing
	for rows.Next() {
		l, err := scanListingRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *l)
	}
	return result, rows.Err()
}

func (q *Queries) CountListings(ctx context.Context, p ListListingsParams) (int, error) {
	where, args := buildListingWhere(p)
	var total int
	err := q.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM market_listings %s`, where), args...).Scan(&total)
	return total, err
}

func buildListingWhere(p ListListingsParams) (string, []interface{}) {
	where := "WHERE 1=1"
	var args []interface{}
	n := 1
	status := models.ListingActive
	if p.Status != nil {
		status = *p.Status
	}
	where += fmt.Sprintf(" AND status = $%d", n)
	args = append(args, status)
	n++
	if p.CropType != nil {
		where += fmt.Sprintf(" AND crop_type = $%d", n)
		args = append(args, *p.CropType)
		n++
	}
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
	if p.MaxPrice != nil {
		where += fmt.Sprintf(" AND price_per_unit <= $%d", n)
		args = append(args, *p.MaxPrice)
		n++
	}
	if p.MinQuantity != nil {
		where += fmt.Sprintf(" AND quantity_available >= $%d", n)
		args = append(args, *p.MinQuantity)
		n++
	}
	if p.FarmerID != nil {
		where += fmt.Sprintf(" AND farmer_id = $%d", n)
		args = append(args, *p.FarmerID)
	}
	return where, args
}

func (q *Queries) UpdateListing(ctx context.Context, id uuid.UUID, req models.UpdateListingRequest, now time.Time) error {
	var availUntil *time.Time
	if req.AvailableUntil != nil {
		t, _ := time.Parse(time.RFC3339, *req.AvailableUntil)
		tUTC := t.UTC()
		availUntil = &tUTC
	}
	_, err := q.pool.Exec(ctx, `
		UPDATE market_listings SET
			price_per_unit   = COALESCE($1, price_per_unit),
			quantity_available = COALESCE($2, quantity_available),
			status           = COALESCE($3, status),
			available_until  = COALESCE($4, available_until),
			description      = COALESCE($5, description),
			updated_at       = $6
		WHERE id = $7`,
		req.PricePerUnit, req.QuantityAvail, req.Status, availUntil, req.Description, now, id,
	)
	return err
}

// ─── Price records ────────────────────────────────────────────────────────────

func (q *Queries) RecordPrice(ctx context.Context, r models.PriceRecord) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO price_records (id, crop_type, market, country, region, price_per_kg, currency, source, recorded_at, recorded_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		r.ID, r.CropType, r.Market, r.Country, r.Region,
		r.PricePerKg, r.Currency, r.Source, r.RecordedAt, r.RecordedBy,
	)
	return err
}

func (q *Queries) GetPriceSummary(ctx context.Context, cropType, market, country string, from, to time.Time) (models.PriceSummary, error) {
	var s models.PriceSummary
	s.CropType = cropType
	s.Market = market
	s.Country = country
	err := q.pool.QueryRow(ctx, `
		SELECT
			COALESCE(MIN(price_per_kg), 0),
			COALESCE(MAX(price_per_kg), 0),
			COALESCE(AVG(price_per_kg), 0),
			COUNT(*)::INT,
			COALESCE(MAX(currency), 'USD')
		FROM price_records
		WHERE crop_type = $1
		  AND ($2 = '' OR market = $2)
		  AND ($3 = '' OR country = $3)
		  AND recorded_at BETWEEN $4 AND $5`,
		cropType, market, country, from, to,
	).Scan(&s.MinPrice, &s.MaxPrice, &s.AvgPrice, &s.DataPoints, &s.Currency)
	s.Period = from.Format("2006-01-02") + " to " + to.Format("2006-01-02")
	return s, err
}

func (q *Queries) ListPriceHistory(ctx context.Context, cropType, country string, days int) ([]models.PriceRecord, error) {
	from := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	rows, err := q.pool.Query(ctx, `
		SELECT id, crop_type, market, country, region, price_per_kg, currency, source, recorded_at, recorded_by
		FROM price_records
		WHERE crop_type = $1 AND ($2 = '' OR country = $2) AND recorded_at >= $3
		ORDER BY recorded_at DESC LIMIT 200`,
		cropType, country, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.PriceRecord
	for rows.Next() {
		var r models.PriceRecord
		if err := rows.Scan(&r.ID, &r.CropType, &r.Market, &r.Country, &r.Region,
			&r.PricePerKg, &r.Currency, &r.Source, &r.RecordedAt, &r.RecordedBy); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// ─── Bids ─────────────────────────────────────────────────────────────────────

func (q *Queries) CreateBid(ctx context.Context, b models.MarketBid) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO market_bids (id, listing_id, buyer_id, quantity_kg, bid_price, currency, status, message, expires_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		b.ID, b.ListingID, b.BuyerID, b.QuantityKg, b.BidPrice,
		b.Currency, b.Status, b.Message, b.ExpiresAt, b.CreatedAt, b.UpdatedAt,
	)
	return err
}

func (q *Queries) GetBid(ctx context.Context, id uuid.UUID) (*models.MarketBid, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, listing_id, buyer_id, quantity_kg, bid_price, currency, status, message, expires_at, created_at, updated_at
		FROM market_bids WHERE id = $1`, id)
	return scanBidRow(row)
}

func (q *Queries) ListBidsForListing(ctx context.Context, listingID uuid.UUID) ([]models.MarketBid, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, listing_id, buyer_id, quantity_kg, bid_price, currency, status, message, expires_at, created_at, updated_at
		FROM market_bids WHERE listing_id = $1 ORDER BY bid_price DESC, created_at ASC`, listingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.MarketBid
	for rows.Next() {
		b, err := scanBidRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *b)
	}
	return result, rows.Err()
}

func (q *Queries) UpdateBidStatus(ctx context.Context, id uuid.UUID, status models.BidStatus, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE market_bids SET status = $1, updated_at = $2 WHERE id = $3`, status, now, id)
	return err
}

// ─── Audit log ────────────────────────────────────────────────────────────────

func (q *Queries) InsertAuditLog(ctx context.Context, log models.MarketAuditLog) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO market_audit_log (id, entity_id, user_id, action, resource, ip_address, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		log.ID, log.EntityID, log.UserID, log.Action, log.Resource, log.IPAddress, log.CreatedAt,
	)
	return err
}

// ─── Row scanners ─────────────────────────────────────────────────────────────

type scannable interface{ Scan(dest ...any) error }

func scanListingRow(row scannable) (*models.MarketListing, error) {
	var l models.MarketListing
	err := row.Scan(
		&l.ID, &l.FarmerID, &l.CropType, &l.Variety, &l.QuantityKg, &l.QuantityAvail,
		&l.PricePerUnit, &l.Currency, &l.PriceUnit, &l.QualityGrade,
		&l.Country, &l.Region, &l.Market, &l.HarvestedAt, &l.AvailableFrom,
		&l.AvailableUntil, &l.Status, &l.Description, &l.CreatedAt, &l.UpdatedAt,
	)
	return &l, err
}

func scanBidRow(row scannable) (*models.MarketBid, error) {
	var b models.MarketBid
	err := row.Scan(&b.ID, &b.ListingID, &b.BuyerID, &b.QuantityKg, &b.BidPrice,
		&b.Currency, &b.Status, &b.Message, &b.ExpiresAt, &b.CreatedAt, &b.UpdatedAt)
	return &b, err
}
