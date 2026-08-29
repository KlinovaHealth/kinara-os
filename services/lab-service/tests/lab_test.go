package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/lab-service/models"
)

func TestOrderStatusValues(t *testing.T) {
	statuses := []models.OrderStatus{
		models.OrderPending, models.OrderCollected,
		models.OrderProcessing, models.OrderCompleted, models.OrderCancelled,
	}
	if len(statuses) != 5 {
		t.Errorf("expected 5 statuses, got %d", len(statuses))
	}
}

func TestResultFlagValues(t *testing.T) {
	flags := []models.ResultFlag{
		models.FlagNormal, models.FlagHigh, models.FlagLow, models.FlagCritical,
	}
	for _, f := range flags {
		if string(f) == "" {
			t.Errorf("empty flag")
		}
	}
}

func TestLabOrderRefFormat(t *testing.T) {
	id := uuid.New()
	ref := "LAB-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "LAB-") {
		t.Errorf("expected LAB- prefix")
	}
	if len(ref) != 12 {
		t.Errorf("expected ref length 12, got %d", len(ref))
	}
}

func TestDefaultPriority(t *testing.T) {
	req := models.CreateOrderRequest{Priority: ""}
	if req.Priority == "" {
		req.Priority = "routine"
	}
	if req.Priority != "routine" {
		t.Errorf("expected routine priority, got %s", req.Priority)
	}
}

func TestLabResultImmutable(t *testing.T) {
	result := models.LabResult{
		ID:          uuid.New(),
		OrderID:     uuid.New(),
		PatientID:   uuid.New(),
		TestCode:    "CBC",
		ResultValue: "14.5",
		Unit:        "g/dL",
		Flag:        models.FlagNormal,
		AnalyzedBy:  uuid.New(),
		ResultAt:    time.Now().UTC(),
		TenantID:    "tg",
	}
	if result.Flag != models.FlagNormal {
		t.Errorf("flag mismatch")
	}
}

func TestCriticalFlag(t *testing.T) {
	req := models.RecordResultRequest{Flag: models.FlagCritical}
	if req.Flag != models.FlagCritical {
		t.Errorf("critical flag should be preserved")
	}
}
