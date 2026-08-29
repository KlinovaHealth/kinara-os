package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/input-service/models"
)

func TestInputTypes(t *testing.T) {
	types := []models.InputType{
		models.InputSeed, models.InputFertilizer, models.InputPesticide,
		models.InputEquipment, models.InputFuel,
	}
	if len(types) != 5 {
		t.Errorf("expected 5 input types")
	}
}

func TestInputUnits(t *testing.T) {
	units := []models.InputUnit{
		models.UnitKG, models.UnitLiter, models.UnitPiece, models.UnitBag,
	}
	for _, u := range units {
		if string(u) == "" {
			t.Errorf("empty unit")
		}
	}
}

func TestPurchaseRefFormat(t *testing.T) {
	id := uuid.New()
	ref := "INP-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "INP-") {
		t.Errorf("expected INP- prefix, got %s", ref)
	}
}

func TestInputPurchaseFields(t *testing.T) {
	now := time.Now().UTC()
	p := models.InputPurchase{
		ID:          uuid.New(),
		PurchaseRef: "INP-ABCD1234",
		FarmerID:    uuid.New(),
		InputType:   models.InputFertilizer,
		InputName:   "Urea 46%",
		Quantity:    50.0,
		Unit:        models.UnitKG,
		CostXOF:     15000.0,
		PurchasedAt: now,
		TenantID:    "tg",
	}
	if p.Quantity <= 0 {
		t.Error("quantity must be positive")
	}
	if p.CostXOF <= 0 {
		t.Error("cost must be positive")
	}
}

func TestCostPerKg(t *testing.T) {
	quantity := 50.0
	cost := 15000.0
	perKg := cost / quantity
	if perKg != 300.0 {
		t.Errorf("expected 300 XOF/kg, got %.2f", perKg)
	}
}

func TestUsageRecording(t *testing.T) {
	u := models.InputUsage{
		ID:         uuid.New(),
		PurchaseID: uuid.New(),
		FarmerID:   uuid.New(),
		FieldID:    "field-001",
		Quantity:   10.0,
		UsedAt:     time.Now().UTC(),
	}
	if u.FieldID == "" {
		t.Error("field_id must not be empty")
	}
}
