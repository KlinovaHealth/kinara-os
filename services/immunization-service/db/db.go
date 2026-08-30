package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klinova/kinara-os/immunization-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) RecordImmunization(ctx context.Context, r models.ImmunizationRecord) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO immunization_records
		 (id,record_ref,patient_id,vaccine_code,vaccine_name,dose_number,administered_by,
		  administered_at,lot_number,expiry_date,site_of_injection,clinic_id,next_dose_date,
		  status,tenant_id,created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		r.ID, r.RecordRef, r.PatientID, r.VaccineCode, r.VaccineName, r.DoseNumber,
		r.AdministeredBy, r.AdministeredAt, r.LotNumber, r.ExpiryDate, r.SiteOfInjection,
		r.ClinicID, r.NextDoseDate, r.Status, r.TenantID, r.CreatedAt)
	return err
}

// CreateRecord is an alias kept for backward compat with existing main.go wiring.
func (q *Queries) CreateRecord(ctx context.Context, r models.ImmunizationRecord) error {
	return q.RecordImmunization(ctx, r)
}

func (q *Queries) ListByPatient(ctx context.Context, patientID uuid.UUID) ([]models.ImmunizationRecord, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id,record_ref,patient_id,vaccine_code,vaccine_name,dose_number,administered_by,
		        administered_at,lot_number,expiry_date,site_of_injection,clinic_id,next_dose_date,
		        status,tenant_id,created_at
		 FROM immunization_records
		 WHERE patient_id=$1
		 ORDER BY administered_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ImmunizationRecord
	for rows.Next() {
		var r models.ImmunizationRecord
		if err := rows.Scan(
			&r.ID, &r.RecordRef, &r.PatientID, &r.VaccineCode, &r.VaccineName,
			&r.DoseNumber, &r.AdministeredBy, &r.AdministeredAt, &r.LotNumber,
			&r.ExpiryDate, &r.SiteOfInjection, &r.ClinicID, &r.NextDoseDate,
			&r.Status, &r.TenantID, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// GetSchedule returns upcoming/overdue vaccines for a patient.
// Full logic requires date_of_birth from patient-service; placeholder returns empty slice.
func (q *Queries) GetSchedule(ctx context.Context, patientID uuid.UUID) ([]models.VaccineDue, error) {
	return []models.VaccineDue{}, nil
}

func (q *Queries) InsertAlert(ctx context.Context, alert models.ImmunizationAlert) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO immunization_alerts (id,patient_id,message,sent_at) VALUES ($1,$2,$3,$4)`,
		alert.ID, alert.PatientID, alert.Message, alert.SentAt)
	return err
}

func (q *Queries) GetClinicCompliance(ctx context.Context, clinicID string) (models.ComplianceReport, error) {
	row := q.pool.QueryRow(ctx,
		`SELECT
		    COUNT(*) AS total_eligible,
		    COUNT(*) FILTER (WHERE status='completed' OR status='administered') AS vaccinated_count
		 FROM immunization_records
		 WHERE clinic_id=$1`, clinicID)
	var report models.ComplianceReport
	report.ClinicID = clinicID
	var total, vaccinated int
	if err := row.Scan(&total, &vaccinated); err != nil {
		return report, err
	}
	report.TotalEligible = total
	report.VaccinatedCount = vaccinated
	if total > 0 {
		report.CompliancePct = float64(vaccinated) / float64(total) * 100.0
	}
	return report, nil
}

func (q *Queries) GetPopulationCoverage(ctx context.Context) ([]models.CoverageItem, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT vaccine_code, COUNT(*) AS coverage_count
		 FROM immunization_records
		 GROUP BY vaccine_code
		 ORDER BY coverage_count DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.CoverageItem
	for rows.Next() {
		var item models.CoverageItem
		if err := rows.Scan(&item.VaccineCode, &item.CoverageCount); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (q *Queries) InsertAudit(ctx context.Context, rec models.ImmunizationRecord, actorID string) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO immunization_audit_log (record_id,action,actor_id,occurred_at)
		 VALUES ($1,$2,$3,NOW())`,
		rec.ID, "record_immunization", actorID)
	return err
}

func (q *Queries) CountOverdue(ctx context.Context, patientID uuid.UUID) (int, error) {
	var count int
	err := q.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM immunization_records WHERE patient_id=$1 AND status='overdue'`,
		patientID).Scan(&count)
	return count, err
}
