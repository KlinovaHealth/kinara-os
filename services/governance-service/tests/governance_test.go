package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/klinova/kinara-os/governance-service/crypto"
	"github.com/klinova/kinara-os/governance-service/models"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	enc, err := crypto.NewEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := `{"case_count":142,"death_count":3,"icd10":"A09"}`
	ciphertext, err := enc.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	got, err := enc.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if got != plaintext {
		t.Errorf("got %q, want %q", got, plaintext)
	}
}

func TestEncryptUniqueCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := crypto.NewEncryptor(key)
	c1, _ := enc.EncryptString("case_count:100")
	c2, _ := enc.EncryptString("case_count:100")
	if c1 == c2 {
		t.Error("two encryptions identical — nonce not random")
	}
}

func TestWrongKeyFails(t *testing.T) {
	k1, k2 := make([]byte, 32), make([]byte, 32)
	k2[0] = 0xFF
	e1, _ := crypto.NewEncryptor(k1)
	e2, _ := crypto.NewEncryptor(k2)
	ct, _ := e1.EncryptString("outbreak data")
	_, err := e2.DecryptString(ct)
	if err == nil {
		t.Error("expected error with wrong key")
	}
}

func TestInvalidKeySize(t *testing.T) {
	_, err := crypto.NewEncryptor([]byte("short"))
	if err != crypto.ErrInvalidKeySize {
		t.Errorf("expected ErrInvalidKeySize, got %v", err)
	}
}

func TestComplianceStatusConstants(t *testing.T) {
	statuses := []models.ComplianceStatus{
		models.CompliancePending,
		models.ComplianceCompliant,
		models.ComplianceViolation,
		models.ComplianceExempted,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty compliance status constant")
		}
	}
}

func TestReportTypeConstants(t *testing.T) {
	types := []models.ReportType{
		models.ReportEpidemiology,
		models.ReportCompliance,
		models.ReportDiseaseBurden,
		models.ReportOutbreak,
		models.ReportMortality,
		models.ReportImmunization,
	}
	for _, rt := range types {
		if rt == "" {
			t.Error("empty report type constant")
		}
	}
}

func TestAlertSeverityConstants(t *testing.T) {
	sev := []models.AlertSeverity{
		models.AlertInfo,
		models.AlertWarning,
		models.AlertCritical,
	}
	for _, s := range sev {
		if s == "" {
			t.Error("empty alert severity constant")
		}
	}
}

func TestCreateComplianceReportRequestJSON(t *testing.T) {
	req := models.CreateComplianceReportRequest{
		ReportType:  models.ReportEpidemiology,
		Frequency:   models.FrequencyMonthly,
		PeriodStart: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		Country:     "Kenya",
		Summary:     "Monthly disease burden report for Nairobi",
		DataPayload: map[string]interface{}{"malaria_cases": 1420, "typhoid_cases": 83},
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var back models.CreateComplianceReportRequest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}

	if back.Country != "Kenya" {
		t.Errorf("country mismatch: %q", back.Country)
	}
	if back.Summary != req.Summary {
		t.Errorf("summary mismatch: %q", back.Summary)
	}
}

func TestCreateAlertRequestJSON(t *testing.T) {
	req := models.CreateAlertRequest{
		Severity:    models.AlertCritical,
		Title:       "Cholera outbreak threshold exceeded",
		Description: "Case count in Mombasa exceeded reporting threshold of 50 in 7 days",
		Country:     "Kenya",
		Region:      "Coast",
		Metadata:    map[string]interface{}{"case_count": 67, "threshold": 50},
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var back models.CreateAlertRequest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}

	if back.Severity != models.AlertCritical {
		t.Errorf("severity mismatch: %q", back.Severity)
	}
	if back.Country != "Kenya" {
		t.Errorf("country mismatch: %q", back.Country)
	}
}

func TestAPIResponseShape(t *testing.T) {
	resp := models.APIResponse{
		Success: true,
		Data:    map[string]string{"id": "abc"},
		Meta:    &models.PageMeta{Page: 1, Limit: 20, Total: 150, TotalPages: 8},
	}

	b, _ := json.Marshal(resp)
	var m map[string]interface{}
	json.Unmarshal(b, &m)

	if m["success"] != true {
		t.Error("expected success=true")
	}
	if m["meta"] == nil {
		t.Error("expected meta field")
	}
}

func TestAPIResponseErrorShape(t *testing.T) {
	resp := models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: "FORBIDDEN", Message: "insufficient role"},
	}

	b, _ := json.Marshal(resp)
	var m map[string]interface{}
	json.Unmarshal(b, &m)

	errObj, ok := m["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object")
	}
	if errObj["code"] != "FORBIDDEN" {
		t.Errorf("unexpected code: %v", errObj["code"])
	}
}

func TestHealthEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"governance-service"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "governance-service" {
		t.Errorf("unexpected service: %q", body["service"])
	}
}

func TestMissingAuthHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, `{"success":false}`, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance-reports", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestEpidemiologyRecordJSON(t *testing.T) {
	req := models.CreateEpidemiologyRecordRequest{
		ICD10Code:      "A09",
		ICD10Desc:      "Infectious gastroenteritis and colitis",
		Country:        "Ghana",
		Region:         "Greater Accra",
		CaseCount:      234,
		DeathCount:     4,
		RecoveredCount: 189,
		PeriodStart:    time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:      time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC),
		AgeGroup:       "0-4",
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var back models.CreateEpidemiologyRecordRequest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}

	if back.CaseCount != 234 {
		t.Errorf("case_count mismatch: %d", back.CaseCount)
	}
	if back.ICD10Code != "A09" {
		t.Errorf("icd10 mismatch: %q", back.ICD10Code)
	}
}

func TestAuditLogSerialization(t *testing.T) {
	log := models.GovernanceAuditLog{
		ResourceType: "compliance_report",
		Action:       models.AuditCreate,
		AccessorRole: "government",
		IPAddress:    "10.0.1.5",
		RequestID:    "req-456",
		CreatedAt:    time.Now(),
	}

	b, err := json.Marshal(log)
	if err != nil {
		t.Fatal(err)
	}

	var back models.GovernanceAuditLog
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}

	if back.Action != models.AuditCreate {
		t.Errorf("unexpected action: %q", back.Action)
	}
}
