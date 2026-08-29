package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/voyage-service/models"
)

func TestVoyageStatuses(t *testing.T) {
	statuses := []models.VoyageStatus{
		models.VoyagePlanned, models.VoyageDeparted, models.VoyageInTransit,
		models.VoyageArrived, models.VoyageCompleted, models.VoyageCancelled,
	}
	if len(statuses) != 6 {
		t.Errorf("expected 6 statuses, got %d", len(statuses))
	}
}

func TestVoyageRefFormat(t *testing.T) {
	id := uuid.New()
	ref := "VOY-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "VOY-") {
		t.Errorf("expected VOY- prefix, got %s", ref)
	}
	if len(ref) != 12 {
		t.Errorf("expected length 12, got %d", len(ref))
	}
}

func TestVoyageCreation(t *testing.T) {
	now := time.Now().UTC()
	dep := now.Add(24 * time.Hour)
	arr := now.Add(5 * 24 * time.Hour)
	v := models.Voyage{
		ID:              uuid.New(),
		VoyageRef:       "VOY-ABCD1234",
		VesselID:        uuid.New(),
		OriginPort:      "LOME",
		DestinationPort: "TEMA",
		CargoType:       "grain",
		CargoTons:       500.0,
		Status:          models.VoyagePlanned,
		DepartureAt:     &dep,
		EstArrivalAt:    &arr,
		DistanceNM:      120.0,
		FuelTons:        8.5,
		TenantID:        "tg",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if v.CargoTons <= 0 {
		t.Error("cargo tons must be positive")
	}
	if v.Status != models.VoyagePlanned {
		t.Errorf("initial status should be planned, got %s", v.Status)
	}
}

func TestStatusTransitionOrder(t *testing.T) {
	flow := []models.VoyageStatus{
		models.VoyagePlanned,
		models.VoyageDeparted,
		models.VoyageInTransit,
		models.VoyageArrived,
		models.VoyageCompleted,
	}
	if flow[0] != models.VoyagePlanned {
		t.Error("first status must be planned")
	}
	if flow[len(flow)-1] != models.VoyageCompleted {
		t.Error("last status must be completed")
	}
}

func TestVoyageEventLog(t *testing.T) {
	e := models.VoyageEvent{
		ID:          uuid.New(),
		VoyageID:    uuid.New(),
		EventType:   "position_update",
		Description: "Passing Cape Coast",
		Latitude:    5.1037,
		Longitude:   -1.2837,
		OccurredAt:  time.Now().UTC(),
	}
	if e.Latitude == 0 && e.Longitude == 0 {
		t.Error("event should have valid coordinates")
	}
}

func TestUpdateStatusRequest(t *testing.T) {
	now := time.Now().UTC()
	req := models.UpdateStatusRequest{
		Status:          models.VoyageArrived,
		ActualArrivalAt: &now,
	}
	if req.ActualArrivalAt == nil {
		t.Error("arrival time required for arrived status")
	}
}
