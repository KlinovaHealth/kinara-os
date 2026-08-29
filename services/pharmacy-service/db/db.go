package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/pharmacy-service/models"
)

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

// ─── Medications ──────────────────────────────────────────────────────────────

func (q *Queries) CreateMedication(ctx context.Context, row models.MedicationRow) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO medications
			(id, name, generic_name, description, unit_price, currency,
			 stock_level, reorder_point, reorder_qty, unit, supplier_id,
			 expiration_date, batch_number, requires_cold, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		row.ID, row.Name, row.GenericName, row.Description, row.UnitPrice, row.Currency,
		row.StockLevel, row.ReorderPoint, row.ReorderQty, row.Unit, row.SupplierID,
		row.ExpirationDate, row.BatchNumber, row.RequiresCold, row.IsActive,
		row.CreatedAt, row.UpdatedAt,
	)
	return err
}

func (q *Queries) GetMedication(ctx context.Context, id uuid.UUID) (*models.MedicationRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, name, generic_name, description, unit_price, currency,
		       stock_level, reorder_point, reorder_qty, unit, supplier_id,
		       expiration_date, batch_number, requires_cold, is_active, created_at, updated_at
		FROM medications WHERE id = $1`, id)
	return scanMedicationRow(row)
}

type ListInventoryParams struct {
	LowStockOnly bool
	Page         int
	Limit        int
}

func (q *Queries) ListInventory(ctx context.Context, p ListInventoryParams) ([]models.MedicationRow, error) {
	where := "WHERE is_active = TRUE"
	var args []interface{}
	n := 1
	if p.LowStockOnly {
		where += fmt.Sprintf(" AND stock_level <= reorder_point")
	}
	offset := (p.Page - 1) * p.Limit
	args = append(args, p.Limit, offset)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, name, generic_name, description, unit_price, currency,
		       stock_level, reorder_point, reorder_qty, unit, supplier_id,
		       expiration_date, batch_number, requires_cold, is_active, created_at, updated_at
		FROM medications %s ORDER BY name ASC LIMIT $%d OFFSET $%d`,
		where, n, n+1), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.MedicationRow
	for rows.Next() {
		m, err := scanMedicationRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *m)
	}
	return result, rows.Err()
}

func (q *Queries) CountInventory(ctx context.Context, p ListInventoryParams) (int, error) {
	where := "WHERE is_active = TRUE"
	if p.LowStockOnly {
		where += " AND stock_level <= reorder_point"
	}
	var total int
	err := q.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM medications %s`, where)).Scan(&total)
	return total, err
}

type UpdateStockParams struct {
	ID             uuid.UUID
	StockLevel     *int
	ReorderPoint   *int
	ReorderQty     *int
	UnitPrice      *float64
	BatchNumber    *string
	ExpirationDate *time.Time
	Now            time.Time
}

func (q *Queries) UpdateStock(ctx context.Context, p UpdateStockParams) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE medications SET
			stock_level     = COALESCE($1, stock_level),
			reorder_point   = COALESCE($2, reorder_point),
			reorder_qty     = COALESCE($3, reorder_qty),
			unit_price      = COALESCE($4, unit_price),
			batch_number    = COALESCE($5, batch_number),
			expiration_date = COALESCE($6, expiration_date),
			updated_at      = $7
		WHERE id = $8`,
		p.StockLevel, p.ReorderPoint, p.ReorderQty, p.UnitPrice,
		p.BatchNumber, p.ExpirationDate, p.Now, p.ID,
	)
	return err
}

func (q *Queries) DecrementStock(ctx context.Context, medicationID uuid.UUID, qty int, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE medications SET stock_level = stock_level - $1, updated_at = $2 WHERE id = $3 AND stock_level >= $1`,
		qty, now, medicationID,
	)
	return err
}

// GetStockAlerts returns medications that are low on stock or expiring within 30 days.
func (q *Queries) GetStockAlerts(ctx context.Context) ([]models.MedicationRow, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, name, generic_name, description, unit_price, currency,
		       stock_level, reorder_point, reorder_qty, unit, supplier_id,
		       expiration_date, batch_number, requires_cold, is_active, created_at, updated_at
		FROM medications
		WHERE is_active = TRUE AND (
			stock_level <= reorder_point
			OR (expiration_date IS NOT NULL AND expiration_date < NOW() + INTERVAL '30 days')
		)
		ORDER BY stock_level ASC, expiration_date ASC NULLS LAST`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.MedicationRow
	for rows.Next() {
		m, err := scanMedicationRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *m)
	}
	return result, rows.Err()
}

// ─── Prescriptions ────────────────────────────────────────────────────────────

func (q *Queries) CreatePrescription(ctx context.Context, row models.PrescriptionRow) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO prescriptions
			(id, clinical_id, patient_id, clinic_id, medication_id,
			 patient_name_enc, dosage_enc, quantity, quantity_unit,
			 instructions, status, issued_at, expires_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		row.ID, row.ClinicalID, row.PatientID, row.ClinicID, row.MedicationID,
		row.PatientNameEnc, row.DosageEnc, row.Quantity, row.QuantityUnit,
		row.Instructions, row.Status, row.IssuedAt, row.ExpiresAt,
		row.CreatedAt, row.UpdatedAt,
	)
	return err
}

func (q *Queries) GetPrescription(ctx context.Context, id uuid.UUID) (*models.PrescriptionRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, clinical_id, patient_id, clinic_id, medication_id,
		       patient_name_enc, dosage_enc, quantity, quantity_unit,
		       instructions, status, issued_at, expires_at, created_at, updated_at
		FROM prescriptions WHERE id = $1`, id)
	return scanPrescriptionRow(row)
}

type ListPrescriptionsParams struct {
	ClinicID  *uuid.UUID
	PatientID *uuid.UUID
	Status    *models.PrescriptionStatus
	Page      int
	Limit     int
}

func (q *Queries) ListPrescriptions(ctx context.Context, p ListPrescriptionsParams) ([]models.PrescriptionRow, error) {
	where, args := buildPrescriptionWhere(p)
	offset := (p.Page - 1) * p.Limit
	n := len(args) + 1
	args = append(args, p.Limit, offset)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, clinical_id, patient_id, clinic_id, medication_id,
		       patient_name_enc, dosage_enc, quantity, quantity_unit,
		       instructions, status, issued_at, expires_at, created_at, updated_at
		FROM prescriptions %s ORDER BY issued_at DESC LIMIT $%d OFFSET $%d`,
		where, n, n+1), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.PrescriptionRow
	for rows.Next() {
		pr, err := scanPrescriptionRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *pr)
	}
	return result, rows.Err()
}

func (q *Queries) CountPrescriptions(ctx context.Context, p ListPrescriptionsParams) (int, error) {
	where, args := buildPrescriptionWhere(p)
	var total int
	err := q.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM prescriptions %s`, where), args...).Scan(&total)
	return total, err
}

func buildPrescriptionWhere(p ListPrescriptionsParams) (string, []interface{}) {
	where := "WHERE 1=1"
	var args []interface{}
	n := 1
	if p.ClinicID != nil {
		where += fmt.Sprintf(" AND clinic_id = $%d", n)
		args = append(args, *p.ClinicID)
		n++
	}
	if p.PatientID != nil {
		where += fmt.Sprintf(" AND patient_id = $%d", n)
		args = append(args, *p.PatientID)
		n++
	}
	if p.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, *p.Status)
	}
	return where, args
}

func (q *Queries) UpdatePrescriptionStatus(ctx context.Context, id uuid.UUID, status models.PrescriptionStatus, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE prescriptions SET status = $1, updated_at = $2 WHERE id = $3`,
		status, now, id,
	)
	return err
}

// ─── Dispensing ───────────────────────────────────────────────────────────────

func (q *Queries) CreateDispensing(ctx context.Context, row models.DispensingRow) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO dispensing
			(id, prescription_id, medication_id, dispensed_by_user_id,
			 quantity_dispensed, batch_number, cost_amount, currency,
			 patient_cost_share, notes, dispensed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		row.ID, row.PrescriptionID, row.MedicationID, row.DispensedByUserID,
		row.QuantityDispensed, row.BatchNumber, row.CostAmount, row.Currency,
		row.PatientCostShare, row.Notes, row.DispensedAt,
	)
	return err
}

func (q *Queries) ListDispensingForPrescription(ctx context.Context, prescriptionID uuid.UUID) ([]models.DispensingRow, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, prescription_id, medication_id, dispensed_by_user_id,
		       quantity_dispensed, batch_number, cost_amount, currency,
		       patient_cost_share, notes, dispensed_at
		FROM dispensing WHERE prescription_id = $1 ORDER BY dispensed_at ASC`, prescriptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.DispensingRow
	for rows.Next() {
		var d models.DispensingRow
		if err := rows.Scan(&d.ID, &d.PrescriptionID, &d.MedicationID, &d.DispensedByUserID,
			&d.QuantityDispensed, &d.BatchNumber, &d.CostAmount, &d.Currency,
			&d.PatientCostShare, &d.Notes, &d.DispensedAt); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// GetCostSummary returns aggregate dispensing costs for a clinic in a given period.
func (q *Queries) GetCostSummary(ctx context.Context, clinicID uuid.UUID, from, to time.Time) (models.CostSummary, error) {
	var summary models.CostSummary
	summary.ClinicID = clinicID.String()
	err := q.pool.QueryRow(ctx, `
		SELECT
			COUNT(d.id)::INT,
			COALESCE(SUM(d.cost_amount), 0),
			COALESCE(SUM(d.patient_cost_share), 0),
			COALESCE(MAX(d.currency), 'USD')
		FROM dispensing d
		JOIN prescriptions p ON d.prescription_id = p.id
		WHERE p.clinic_id = $1 AND d.dispensed_at BETWEEN $2 AND $3`,
		clinicID, from, to,
	).Scan(&summary.TotalDispensed, &summary.TotalCost, &summary.PatientCostShare, &summary.Currency)
	if err != nil {
		return summary, err
	}
	summary.FacilityCost = summary.TotalCost - summary.PatientCostShare
	return summary, nil
}

// ─── Supply orders ────────────────────────────────────────────────────────────

func (q *Queries) CreateOrder(ctx context.Context, row models.SupplyOrderRow) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO supply_orders
			(id, supplier_id, medication_id, quantity_ordered, quantity_received,
			 unit_cost, currency, status, ordered_by_id, expected_at, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		row.ID, row.SupplierID, row.MedicationID, row.QuantityOrdered, 0,
		row.UnitCost, row.Currency, row.Status, row.OrderedByID,
		row.ExpectedAt, row.Notes, row.CreatedAt, row.UpdatedAt,
	)
	return err
}

func (q *Queries) GetOrder(ctx context.Context, id uuid.UUID) (*models.SupplyOrderRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, supplier_id, medication_id, quantity_ordered, quantity_received,
		       unit_cost, currency, status, ordered_by_id, expected_at, received_at, notes, created_at, updated_at
		FROM supply_orders WHERE id = $1`, id)
	return scanOrderRow(row)
}

func (q *Queries) ListOrders(ctx context.Context, page, limit int) ([]models.SupplyOrderRow, error) {
	offset := (page - 1) * limit
	rows, err := q.pool.Query(ctx, `
		SELECT id, supplier_id, medication_id, quantity_ordered, quantity_received,
		       unit_cost, currency, status, ordered_by_id, expected_at, received_at, notes, created_at, updated_at
		FROM supply_orders ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.SupplyOrderRow
	for rows.Next() {
		o, err := scanOrderRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *o)
	}
	return result, rows.Err()
}

// ─── Audit log ────────────────────────────────────────────────────────────────

func (q *Queries) InsertAuditLog(ctx context.Context, log models.PharmacyAuditLog) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO pharmacy_audit_log (id, entity_id, user_id, action, resource, ip_address, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		log.ID, log.EntityID, log.UserID, log.Action, log.Resource, log.IPAddress, log.CreatedAt,
	)
	return err
}

// ─── Row scanners ─────────────────────────────────────────────────────────────

type scannable interface {
	Scan(dest ...any) error
}

func scanMedicationRow(row scannable) (*models.MedicationRow, error) {
	var m models.MedicationRow
	err := row.Scan(
		&m.ID, &m.Name, &m.GenericName, &m.Description, &m.UnitPrice, &m.Currency,
		&m.StockLevel, &m.ReorderPoint, &m.ReorderQty, &m.Unit, &m.SupplierID,
		&m.ExpirationDate, &m.BatchNumber, &m.RequiresCold, &m.IsActive,
		&m.CreatedAt, &m.UpdatedAt,
	)
	return &m, err
}

func scanPrescriptionRow(row scannable) (*models.PrescriptionRow, error) {
	var p models.PrescriptionRow
	err := row.Scan(
		&p.ID, &p.ClinicalID, &p.PatientID, &p.ClinicID, &p.MedicationID,
		&p.PatientNameEnc, &p.DosageEnc, &p.Quantity, &p.QuantityUnit,
		&p.Instructions, &p.Status, &p.IssuedAt, &p.ExpiresAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	return &p, err
}

func scanOrderRow(row scannable) (*models.SupplyOrderRow, error) {
	var o models.SupplyOrderRow
	err := row.Scan(
		&o.ID, &o.SupplierID, &o.MedicationID, &o.QuantityOrdered, &o.QuantityReceived,
		&o.UnitCost, &o.Currency, &o.Status, &o.OrderedByID,
		&o.ExpectedAt, &o.ReceivedAt, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)
	return &o, err
}
