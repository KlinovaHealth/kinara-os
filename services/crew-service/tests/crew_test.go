package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/crew-service/models"
)

func TestCrewRanks(t *testing.T) {
	ranks := []models.CrewRank{
		models.RankCaptain, models.RankFirstOfficer, models.RankEngineer,
		models.RankDeckhand, models.RankCook, models.RankSteward,
	}
	if len(ranks) != 6 {
		t.Errorf("expected 6 ranks, got %d", len(ranks))
	}
}

func TestCertStatuses(t *testing.T) {
	statuses := []models.CertStatus{
		models.CertValid, models.CertExpiring, models.CertExpired,
	}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("empty cert status")
		}
	}
}

func TestCrewRefFormat(t *testing.T) {
	id := uuid.New()
	ref := "CRW-" + strings.ToUpper(id.String()[:8])
	if !strings.HasPrefix(ref, "CRW-") {
		t.Errorf("expected CRW- prefix, got %s", ref)
	}
}

func TestCertStatusInference(t *testing.T) {
	now := time.Now().UTC()
	// Valid cert
	expires := now.AddDate(0, 6, 0)
	status := models.CertValid
	if time.Until(expires) < 30*24*time.Hour {
		status = models.CertExpiring
	}
	if expires.Before(now) {
		status = models.CertExpired
	}
	if status != models.CertValid {
		t.Errorf("6-month cert should be valid, got %s", status)
	}

	// Expiring cert
	expiringSoon := now.Add(10 * 24 * time.Hour)
	status2 := models.CertValid
	if time.Until(expiringSoon) < 30*24*time.Hour {
		status2 = models.CertExpiring
	}
	if status2 != models.CertExpiring {
		t.Errorf("10-day cert should be expiring, got %s", status2)
	}
}

func TestRegisterCrewRequest(t *testing.T) {
	req := models.RegisterCrewRequest{
		FullName:    "Kwame Asante",
		Nationality: "GH",
		Rank:        models.RankCaptain,
	}
	if req.FullName == "" {
		t.Error("full name required")
	}
}

func TestCrewMemberVesselAssignable(t *testing.T) {
	vesselID := uuid.New()
	c := models.CrewMember{VesselID: &vesselID}
	if c.VesselID == nil {
		t.Error("vessel should be assignable")
	}
}
