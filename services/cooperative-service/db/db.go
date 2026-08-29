package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/cooperative-service/models"
)

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

// ─── Cooperatives ─────────────────────────────────────────────────────────────

func (q *Queries) CreateCoop(ctx context.Context, c models.Cooperative) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO cooperatives
			(id, name, registration_no, coop_type, status, country, region, district,
			 total_members, total_farm_ha, description, contact_phone, contact_email, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		c.ID, c.Name, c.RegistrationNo, c.CoopType, c.Status,
		c.Country, c.Region, c.District, c.TotalMembers, c.TotalFarmHa,
		c.Description, c.ContactPhone, c.ContactEmail, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (q *Queries) GetCoop(ctx context.Context, id uuid.UUID) (*models.Cooperative, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, name, registration_no, coop_type, status, country, region, district,
		       total_members, total_farm_ha, description, contact_phone, contact_email, created_at, updated_at
		FROM cooperatives WHERE id = $1`, id)
	return scanCoopRow(row)
}

type ListCoopsParams struct {
	Country *string
	Region  *string
	Status  *models.CoopStatus
	Page    int
	Limit   int
}

func (q *Queries) ListCoops(ctx context.Context, p ListCoopsParams) ([]models.Cooperative, error) {
	where, args := buildCoopWhere(p)
	offset := (p.Page - 1) * p.Limit
	n := len(args) + 1
	args = append(args, p.Limit, offset)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, name, registration_no, coop_type, status, country, region, district,
		       total_members, total_farm_ha, description, contact_phone, contact_email, created_at, updated_at
		FROM cooperatives %s ORDER BY name ASC LIMIT $%d OFFSET $%d`,
		where, n, n+1), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Cooperative
	for rows.Next() {
		c, err := scanCoopRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	return result, rows.Err()
}

func (q *Queries) CountCoops(ctx context.Context, p ListCoopsParams) (int, error) {
	where, args := buildCoopWhere(p)
	var total int
	err := q.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM cooperatives %s`, where), args...).Scan(&total)
	return total, err
}

func buildCoopWhere(p ListCoopsParams) (string, []interface{}) {
	where := "WHERE 1=1"
	var args []interface{}
	n := 1
	if p.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, *p.Status)
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
	}
	return where, args
}

func (q *Queries) UpdateCoopStats(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE cooperatives SET
			total_members = (SELECT COUNT(*) FROM coop_members WHERE coop_id = $1 AND status = 'active'),
			updated_at    = $2
		WHERE id = $1`, id, now)
	return err
}

// ─── Members ──────────────────────────────────────────────────────────────────

func (q *Queries) AddMember(ctx context.Context, m models.CoopMember) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO coop_members (id, coop_id, farmer_id, role, status, shares_held, joined_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		m.ID, m.CoopID, m.FarmerID, m.Role, m.Status, m.SharesHeld, m.JoinedAt, m.UpdatedAt,
	)
	return err
}

func (q *Queries) GetMember(ctx context.Context, id uuid.UUID) (*models.CoopMember, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, coop_id, farmer_id, role, status, shares_held, joined_at, exited_at, updated_at
		FROM coop_members WHERE id = $1`, id)
	return scanMemberRow(row)
}

func (q *Queries) GetMemberByFarmer(ctx context.Context, coopID, farmerID uuid.UUID) (*models.CoopMember, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, coop_id, farmer_id, role, status, shares_held, joined_at, exited_at, updated_at
		FROM coop_members WHERE coop_id = $1 AND farmer_id = $2`, coopID, farmerID)
	return scanMemberRow(row)
}

func (q *Queries) ListMembers(ctx context.Context, coopID uuid.UUID, page, limit int) ([]models.CoopMember, error) {
	offset := (page - 1) * limit
	rows, err := q.pool.Query(ctx, `
		SELECT id, coop_id, farmer_id, role, status, shares_held, joined_at, exited_at, updated_at
		FROM coop_members WHERE coop_id = $1 ORDER BY joined_at DESC LIMIT $2 OFFSET $3`,
		coopID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.CoopMember
	for rows.Next() {
		m, err := scanMemberRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *m)
	}
	return result, rows.Err()
}

func (q *Queries) UpdateMember(ctx context.Context, id uuid.UUID, req models.UpdateMemberRequest, now time.Time) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE coop_members SET
			role        = COALESCE($1, role),
			status      = COALESCE($2, status),
			shares_held = COALESCE($3, shares_held),
			exited_at   = CASE WHEN $2 = 'exited' THEN $4 ELSE exited_at END,
			updated_at  = $4
		WHERE id = $5`,
		req.Role, req.Status, req.SharesHeld, now, id,
	)
	return err
}

// ─── Selling pools ────────────────────────────────────────────────────────────

func (q *Queries) CreatePool(ctx context.Context, p models.SellingPool) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO selling_pools
			(id, coop_id, crop_type, target_qty_kg, collected_qty_kg, price_per_kg, currency,
			 status, open_until, total_revenue, description, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		p.ID, p.CoopID, p.CropType, p.TargetQtyKg, p.CollectedQtyKg, p.PricePerKg, p.Currency,
		p.Status, p.OpenUntil, p.TotalRevenue, p.Description, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (q *Queries) GetPool(ctx context.Context, id uuid.UUID) (*models.SellingPool, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, coop_id, crop_type, target_qty_kg, collected_qty_kg, price_per_kg, currency,
		       status, open_until, sold_at, total_revenue, description, created_at, updated_at
		FROM selling_pools WHERE id = $1`, id)
	return scanPoolRow(row)
}

func (q *Queries) ListPools(ctx context.Context, coopID uuid.UUID, page, limit int) ([]models.SellingPool, error) {
	offset := (page - 1) * limit
	rows, err := q.pool.Query(ctx, `
		SELECT id, coop_id, crop_type, target_qty_kg, collected_qty_kg, price_per_kg, currency,
		       status, open_until, sold_at, total_revenue, description, created_at, updated_at
		FROM selling_pools WHERE coop_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		coopID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.SellingPool
	for rows.Next() {
		p, err := scanPoolRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *p)
	}
	return result, rows.Err()
}

func (q *Queries) ClosePool(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE selling_pools SET status = 'closed', updated_at = $1 WHERE id = $2`, now, id)
	return err
}

func (q *Queries) RecordSale(ctx context.Context, id uuid.UUID, pricePerKg, totalRevenue float64, now time.Time) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE selling_pools SET
			status        = 'sold',
			price_per_kg  = $1,
			total_revenue = $2,
			sold_at       = $3,
			updated_at    = $3
		WHERE id = $4`, pricePerKg, totalRevenue, now, id)
	return err
}

// ─── Contributions ────────────────────────────────────────────────────────────

func (q *Queries) AddContribution(ctx context.Context, c models.PoolContribution) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO pool_contributions (id, pool_id, farmer_id, quantity_kg, payout_amount, payout_paid, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		c.ID, c.PoolID, c.FarmerID, c.QuantityKg, c.PayoutAmount, c.PayoutPaid, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (q *Queries) GetContribution(ctx context.Context, id uuid.UUID) (*models.PoolContribution, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, pool_id, farmer_id, quantity_kg, payout_amount, payout_paid, paid_at, created_at, updated_at
		FROM pool_contributions WHERE id = $1`, id)
	return scanContributionRow(row)
}

func (q *Queries) ListContributions(ctx context.Context, poolID uuid.UUID) ([]models.PoolContribution, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, pool_id, farmer_id, quantity_kg, payout_amount, payout_paid, paid_at, created_at, updated_at
		FROM pool_contributions WHERE pool_id = $1 ORDER BY quantity_kg DESC`, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.PoolContribution
	for rows.Next() {
		c, err := scanContributionRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	return result, rows.Err()
}

func (q *Queries) AddPoolQuantity(ctx context.Context, poolID uuid.UUID, qty float64, now time.Time) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE selling_pools SET collected_qty_kg = collected_qty_kg + $1, updated_at = $2 WHERE id = $3`,
		qty, now, poolID)
	return err
}

func (q *Queries) DistributePayouts(ctx context.Context, poolID uuid.UUID, totalRevenue float64, now time.Time) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE pool_contributions pc
		SET payout_amount = (pc.quantity_kg / sp.collected_qty_kg) * $1,
		    updated_at    = $2
		FROM selling_pools sp
		WHERE pc.pool_id = $3 AND sp.id = $3`,
		totalRevenue, now, poolID)
	return err
}

func (q *Queries) MarkPayoutPaid(ctx context.Context, contributionID uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE pool_contributions SET payout_paid = true, paid_at = $1, updated_at = $1 WHERE id = $2`, now, contributionID)
	return err
}

// ─── Audit log ────────────────────────────────────────────────────────────────

func (q *Queries) InsertAuditLog(ctx context.Context, log models.CoopAuditLog) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO coop_audit_log (id, entity_id, user_id, action, resource, ip_address, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		log.ID, log.EntityID, log.UserID, log.Action, log.Resource, log.IPAddress, log.CreatedAt,
	)
	return err
}

// ─── Row scanners ─────────────────────────────────────────────────────────────

type scannable interface{ Scan(dest ...any) error }

func scanCoopRow(row scannable) (*models.Cooperative, error) {
	var c models.Cooperative
	err := row.Scan(&c.ID, &c.Name, &c.RegistrationNo, &c.CoopType, &c.Status,
		&c.Country, &c.Region, &c.District, &c.TotalMembers, &c.TotalFarmHa,
		&c.Description, &c.ContactPhone, &c.ContactEmail, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}

func scanMemberRow(row scannable) (*models.CoopMember, error) {
	var m models.CoopMember
	err := row.Scan(&m.ID, &m.CoopID, &m.FarmerID, &m.Role, &m.Status,
		&m.SharesHeld, &m.JoinedAt, &m.ExitedAt, &m.UpdatedAt)
	return &m, err
}

func scanPoolRow(row scannable) (*models.SellingPool, error) {
	var p models.SellingPool
	err := row.Scan(&p.ID, &p.CoopID, &p.CropType, &p.TargetQtyKg, &p.CollectedQtyKg,
		&p.PricePerKg, &p.Currency, &p.Status, &p.OpenUntil, &p.SoldAt,
		&p.TotalRevenue, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	return &p, err
}

func scanContributionRow(row scannable) (*models.PoolContribution, error) {
	var c models.PoolContribution
	err := row.Scan(&c.ID, &c.PoolID, &c.FarmerID, &c.QuantityKg,
		&c.PayoutAmount, &c.PayoutPaid, &c.PaidAt, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}
