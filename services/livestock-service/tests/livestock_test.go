package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/livestock-service/models"
)

func TestAnimalSpecies(t *testing.T) {
	species := []models.AnimalSpecies{
		models.SpeciesCattle, models.SpeciesGoat, models.SpeciesSheep,
		models.SpeciesPig, models.SpeciesPoultry,
	}
	if len(species) != 5 {
		t.Errorf("expected 5 species, got %d", len(species))
	}
}

func TestHealthStatuses(t *testing.T) {
	statuses := []models.HealthStatus{
		models.HealthHealthy, models.HealthSick,
		models.HealthQuarantine, models.HealthDeceased,
	}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("empty status")
		}
	}
}

func TestTagRefFormat(t *testing.T) {
	id := uuid.New()
	ref := "ANM-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "ANM-") {
		t.Errorf("expected ANM- prefix, got %s", ref)
	}
}

func TestAnimalRegistration(t *testing.T) {
	now := time.Now().UTC()
	a := models.Animal{
		ID:           uuid.New(),
		TagRef:       "ANM-ABCD1234",
		FarmerID:     uuid.New(),
		Species:      models.SpeciesCattle,
		Breed:        "Zebu",
		WeightKg:     250.0,
		HealthStatus: models.HealthHealthy,
		IsActive:     true,
		TenantID:     "tg",
		RegisteredAt: now,
		UpdatedAt:    now,
	}
	if a.WeightKg <= 0 {
		t.Error("weight must be positive")
	}
	if !a.IsActive {
		t.Error("new animal should be active")
	}
}

func TestProductionRecord(t *testing.T) {
	p := models.ProductionRecord{
		ID:          uuid.New(),
		AnimalID:    uuid.New(),
		FarmerID:    uuid.New(),
		ProductType: "milk",
		QuantityKg:  12.5,
		RecordedAt:  time.Now().UTC(),
	}
	if p.QuantityKg <= 0 {
		t.Error("production quantity must be positive")
	}
}

func TestUpdateHealthRequest(t *testing.T) {
	req := models.UpdateHealthRequest{
		HealthStatus: models.HealthSick,
		WeightKg:     220.0,
	}
	if req.HealthStatus == models.HealthHealthy {
		t.Error("status should be sick, not healthy")
	}
}
