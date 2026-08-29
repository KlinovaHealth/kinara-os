package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/immunization-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateRecord(ctx context.Context, r models.ImmunizationRecord) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO immunization_records (id,record_ref,patient_id,vaccine_code,vaccine_name,dose_number,administered_by,administered_at,lot_number,expiry_date,site_of_injection,clinic_id,next_dose_date,status,tenant_id,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		r.ID, r.RecordRef, r.PatientID, r.VaccineCode, r.VaccineName, r.DoseNumber,
		r.AdministeredBy, r.AdministeredAt, r.LotNumber, r.ExpiryDate, r.SiteOfInjection,
		r.ClinicID, r.NextDoseDate, r.Status, r.TenantID, r.CreatedAt)
	return err
}

func (q *Queries) ListByPatient(ctx context.Context, patientID uuid.UUID) ([]models.ImmunizationRecord, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,record_ref,patient_id,vaccine_code,vaccine_name,dose_number,administered_by,administered_at,lot_number,expiry_date,site_of_injection,clinic_id,next_dose_date,status,tenant_id,created_at
		 FROM immunization_records WHERE patient_id=$1 ORDER BY administered_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ImmunizationRecord
	for rows.Next() {
		var r models.ImmunizationRecord
		if err := rows.Scan(&r.ID, &r.RecordRef, &r.PatientID, &r.VaccineCode, &r.VaccineName,
			&r.DoseNumber, &r.AdministeredBy, &r.AdministeredAt, &r.LotNumber, &r.ExpiryDate,
			&r.SiteOfInjection, &r.ClinicID, &r.NextDoseDate, &r.Status, &r.TenantID, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (q *Queries) CountOverdue(ctx context.Context, patientID uuid.UUID) (int, error) {
	var count int
	err := q.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM immunization_records WHERE patient_id=$1 AND status='overdue'`, patientID).Scan(&count)
	return count, err
}
