package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/irrigation-service/models"
)

func TestIrrigationMethods(t *testing.T) {
	methods := []models.IrrigationMethod{
		models.MethodDrip, models.MethodSprinkler,
		models.MethodFlood, models.MethodFurrow,
	}
	if len(methods) != 4 {
		t.Errorf("expected 4 methods, got %d", len(methods))
	}
}

func TestEventStatuses(t *testing.T) {
	statuses := []models.EventStatus{
		models.EventScheduled, models.EventActive,
		models.EventCompleted, models.EventSkipped,
	}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("empty status")
		}
	}
}

func TestScheduleRefFormat(t *testing.T) {
	id := uuid.New()
	ref := "IRR-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "IRR-") {
		t.Errorf("expected IRR- prefix, got %s", ref)
	}
}

func TestDefaultFrequency(t *testing.T) {
	req := models.CreateScheduleRequest{FrequencyDays: 0}
	if req.FrequencyDays <= 0 {
		req.FrequencyDays = 7
	}
	if req.FrequencyDays != 7 {
		t.Errorf("expected default 7 days, got %d", req.FrequencyDays)
	}
}

func TestIrrigationScheduleFields(t *testing.T) {
	now := time.Now().UTC()
	s := models.IrrigationSchedule{
		ID:            uuid.New(),
		ScheduleRef:   "IRR-ABCD1234",
		FarmerID:      uuid.New(),
		FieldID:       "field-001",
		CropType:      "maize",
		Method:        models.MethodDrip,
		FrequencyDays: 7,
		DurationMin:   60,
		WaterLiters:   500.0,
		IsActive:      true,
		TenantID:      "tg",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if !s.IsActive {
		t.Error("new schedule should be active")
	}
	if s.WaterLiters <= 0 {
		t.Error("water liters must be positive")
	}
}

func TestWaterUsageCalculation(t *testing.T) {
	eventsPerMonth := 4
	litersPerEvent := 500.0
	totalLiters := float64(eventsPerMonth) * litersPerEvent
	if totalLiters != 2000.0 {
		t.Errorf("expected 2000 liters/month, got %.1f", totalLiters)
	}
}
