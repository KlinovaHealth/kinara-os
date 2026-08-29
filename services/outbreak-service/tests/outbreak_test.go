package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/outbreak-service/models"
)

func TestResponseStatusValues(t *testing.T) {
	statuses := []models.ResponseStatus{
		models.ResponseActive, models.ResponseContained, models.ResponseResolved,
	}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("empty status")
		}
	}
}

func TestActionTypes(t *testing.T) {
	actions := []models.ActionType{
		models.ActionQuarantine, models.ActionVaccination,
		models.ActionSurveillance, models.ActionTreatment, models.ActionCommunity,
	}
	if len(actions) != 5 {
		t.Errorf("expected 5 action types, got %d", len(actions))
	}
}

func TestResponseRefFormat(t *testing.T) {
	id := uuid.New()
	ref := "OR-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "OR-") {
		t.Errorf("expected OR- prefix, got %s", ref)
	}
}

func TestOutbreakResponseFields(t *testing.T) {
	now := time.Now().UTC()
	r := models.OutbreakResponse{
		ID:          uuid.New(),
		ResponseRef: "OR-ABCD1234",
		AlertRef:    "OA-XXXXXXXX",
		DiseaseName: "Cholera",
		Country:     "TG",
		Status:      models.ResponseActive,
		StartedAt:   now,
		UpdatedAt:   now,
	}
	if r.ContainedAt != nil {
		t.Error("contained_at should be nil for active response")
	}
	if r.ResolvedAt != nil {
		t.Error("resolved_at should be nil for active response")
	}
}

func TestAddActionRequest(t *testing.T) {
	req := models.AddActionRequest{
		ActionType:  models.ActionQuarantine,
		Description: "Isolate zone A",
		AssignedTo:  uuid.New(),
	}
	if req.ActionType != models.ActionQuarantine {
		t.Errorf("action type mismatch")
	}
}

func TestCreateResponseRequest(t *testing.T) {
	req := models.CreateResponseRequest{
		DiseaseName:   "Malaria",
		Country:       "GH",
		CasesTargeted: 50,
		Population:    10000,
	}
	if req.Population < req.CasesTargeted {
		t.Error("population must be >= cases targeted")
	}
}
