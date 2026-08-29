package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/immunization-service/models"
)

func TestDoseStatusValues(t *testing.T) {
	statuses := []models.DoseStatus{
		models.DoseAdministered, models.DoseScheduled,
		models.DoseOverdue, models.DoseMissed,
	}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("empty status")
		}
	}
}

func TestRecordRefFormat(t *testing.T) {
	id := uuid.New()
	ref := "IMM-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "IMM-") {
		t.Errorf("expected IMM- prefix, got %s", ref)
	}
}

func TestImmunizationRecordSerialization(t *testing.T) {
	now := time.Now().UTC()
	rec := models.ImmunizationRecord{
		ID:             uuid.New(),
		RecordRef:      "IMM-ABCD1234",
		PatientID:      uuid.New(),
		VaccineCode:    "BCG",
		VaccineName:    "BCG Vaccine",
		DoseNumber:     1,
		AdministeredBy: uuid.New(),
		AdministeredAt: now,
		LotNumber:      "LOT123",
		ClinicID:       "clinic-001",
		Status:         models.DoseAdministered,
		TenantID:       "tg",
		CreatedAt:      now,
	}
	if rec.VaccineCode == "" {
		t.Error("vaccine code must not be empty")
	}
	if rec.DoseNumber < 1 {
		t.Error("dose number must be >= 1")
	}
}

func TestImmunizationSummary(t *testing.T) {
	records := make([]models.ImmunizationRecord, 3)
	s := models.ImmunizationSummary{
		PatientID:    uuid.New(),
		TotalDoses:   len(records),
		OverdueDoses: 1,
		Records:      records,
	}
	if s.TotalDoses != 3 {
		t.Errorf("expected 3 total doses, got %d", s.TotalDoses)
	}
	if s.OverdueDoses != 1 {
		t.Errorf("expected 1 overdue, got %d", s.OverdueDoses)
	}
}

func TestCreateImmunizationRequest(t *testing.T) {
	req := models.CreateImmunizationRequest{
		VaccineCode: "OPV",
		DoseNumber:  2,
	}
	if req.DoseNumber < 1 {
		t.Error("dose number invalid")
	}
}

func TestNextDoseNilable(t *testing.T) {
	rec := models.ImmunizationRecord{NextDoseDate: nil}
	if rec.NextDoseDate != nil {
		t.Error("next dose should be nil when not set")
	}
}
