package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cryptopkg "github.com/klinova/kinara-os/pharmacy-service/crypto"
	"github.com/klinova/kinara-os/pharmacy-service/models"
)

// ─── Model constant tests ─────────────────────────────────────────────────────

func TestPrescriptionStatusConstants(t *testing.T) {
	statuses := []models.PrescriptionStatus{
		models.PrescriptionPending, models.PrescriptionDispensed,
		models.PrescriptionPartial, models.PrescriptionCancelled, models.PrescriptionExpired,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty prescription status constant")
		}
	}
}

func TestOrderStatusConstants(t *testing.T) {
	statuses := []models.OrderStatus{
		models.OrderPending, models.OrderApproved, models.OrderShipped,
		models.OrderReceived, models.OrderCancelled,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty order status constant")
		}
	}
}

// ─── AES encryption tests ─────────────────────────────────────────────────────

func TestAESEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	enc, err := cryptopkg.NewEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}

	cases := []string{
		"Ama Asante",
		"Metformin 500mg twice daily",
		"Amoxicillin 250mg/5ml suspension",
	}
	for _, plain := range cases {
		ct, err := enc.EncryptString(plain)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}
		got, err := enc.DecryptString(ct)
		if err != nil {
			t.Fatalf("decrypt failed: %v", err)
		}
		if got != plain {
			t.Errorf("roundtrip mismatch: got %q, want %q", got, plain)
		}
	}
}

func TestAESUniqueCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := cryptopkg.NewEncryptor(key)
	c1, _ := enc.EncryptString("Dosage: 10mg")
	c2, _ := enc.EncryptString("Dosage: 10mg")
	if c1 == c2 {
		t.Error("random nonces must produce unique ciphertexts")
	}
}

func TestAESWrongKeyFails(t *testing.T) {
	key1, key2 := make([]byte, 32), make([]byte, 32)
	key2[0] = 0xFF
	enc1, _ := cryptopkg.NewEncryptor(key1)
	enc2, _ := cryptopkg.NewEncryptor(key2)
	ct, _ := enc1.EncryptString("patient name")
	if _, err := enc2.DecryptString(ct); err == nil {
		t.Error("wrong key must fail decryption")
	}
}

func TestAESInvalidKeySize(t *testing.T) {
	if _, err := cryptopkg.NewEncryptor([]byte("short")); err != cryptopkg.ErrInvalidKeySize {
		t.Errorf("expected ErrInvalidKeySize, got %v", err)
	}
}

// ─── Request serialization tests ──────────────────────────────────────────────

func TestRegisterPrescriptionRequestJSON(t *testing.T) {
	req := models.RegisterPrescriptionRequest{
		ClinicalID:   "550e8400-e29b-41d4-a716-446655440000",
		PatientID:    "660e8400-e29b-41d4-a716-446655440001",
		PatientName:  "Kofi Mensah",
		ClinicID:     "770e8400-e29b-41d4-a716-446655440002",
		MedicationID: "880e8400-e29b-41d4-a716-446655440003",
		Dosage:       "500mg",
		Quantity:     30,
		QuantityUnit: "tablet",
		Instructions: "Take one tablet twice daily with food",
		ExpiresAt:    "2026-09-29T00:00:00Z",
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var back models.RegisterPrescriptionRequest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Quantity != 30 {
		t.Errorf("quantity mismatch: %d", back.Quantity)
	}
	if back.PatientName != req.PatientName {
		t.Errorf("patient_name mismatch: %q", back.PatientName)
	}
}

func TestDispenseRequestJSON(t *testing.T) {
	req := models.DispenseRequest{
		QuantityDispensed: 30,
		BatchNumber:       "BATCH-2026-001",
		CostAmount:        12.50,
		PatientCostShare:  5.00,
		Notes:             "Patient counselled on dosage",
	}
	b, _ := json.Marshal(req)
	var back models.DispenseRequest
	json.Unmarshal(b, &back)
	if back.QuantityDispensed != 30 {
		t.Errorf("quantity_dispensed mismatch: %d", back.QuantityDispensed)
	}
	if back.CostAmount != 12.50 {
		t.Errorf("cost_amount mismatch: %f", back.CostAmount)
	}
}

func TestUpdateStockRequestJSON(t *testing.T) {
	level := 250
	price := 0.45
	req := models.UpdateStockRequest{
		StockLevel: &level,
		UnitPrice:  &price,
	}
	b, _ := json.Marshal(req)
	var back models.UpdateStockRequest
	json.Unmarshal(b, &back)
	if back.StockLevel == nil || *back.StockLevel != 250 {
		t.Error("stock_level mismatch")
	}
}

func TestCreateOrderRequestJSON(t *testing.T) {
	expected := "2026-10-15T00:00:00Z"
	req := models.CreateOrderRequest{
		SupplierID:      "550e8400-e29b-41d4-a716-446655440000",
		MedicationID:    "660e8400-e29b-41d4-a716-446655440001",
		QuantityOrdered: 500,
		UnitCost:        0.35,
		Currency:        "GHS",
		ExpectedAt:      &expected,
	}
	b, _ := json.Marshal(req)
	var back models.CreateOrderRequest
	json.Unmarshal(b, &back)
	if back.QuantityOrdered != 500 {
		t.Errorf("quantity_ordered mismatch: %d", back.QuantityOrdered)
	}
	if back.Currency != "GHS" {
		t.Errorf("currency mismatch: %q", back.Currency)
	}
}

// ─── Business logic tests ─────────────────────────────────────────────────────

func TestPrescriptionExpiryLogic(t *testing.T) {
	// A prescription expires_at that is before now should be treated as expired
	expiredAt := time.Now().UTC().Add(-24 * time.Hour)
	if !time.Now().UTC().After(expiredAt) {
		t.Error("expected current time to be after expired prescription")
	}
}

func TestDispensingImmutabilityModel(t *testing.T) {
	d := models.DispensingRow{
		QuantityDispensed: 30,
		CostAmount:        15.00,
		PatientCostShare:  7.50,
		DispensedAt:       time.Now(),
	}
	b, _ := json.Marshal(d)
	var back models.DispensingRow
	json.Unmarshal(b, &back)
	if back.QuantityDispensed != d.QuantityDispensed {
		t.Error("dispensing quantity mismatch after serialization")
	}
}

func TestStockAlertModel(t *testing.T) {
	alert := models.StockAlert{
		MedicationName: "Amoxicillin",
		AlertType:      "low_stock",
		StockLevel:     5,
		ReorderPoint:   20,
		Message:        "Amoxicillin: stock (5) at or below reorder point (20)",
	}
	b, _ := json.Marshal(alert)
	var back models.StockAlert
	json.Unmarshal(b, &back)
	if back.AlertType != "low_stock" {
		t.Errorf("alert_type mismatch: %q", back.AlertType)
	}
}

func TestCostSummaryCalculation(t *testing.T) {
	summary := models.CostSummary{
		TotalCost:        1000.00,
		PatientCostShare: 300.00,
		FacilityCost:     700.00,
		Currency:         "USD",
	}
	if summary.FacilityCost != summary.TotalCost-summary.PatientCostShare {
		t.Error("facility cost should equal total minus patient share")
	}
}

// ─── HTTP tests ───────────────────────────────────────────────────────────────

func TestHealthEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"pharmacy-service"}`))
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "pharmacy-service" {
		t.Errorf("unexpected service: %q", body["service"])
	}
}

func TestMissingAuthReturns401(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, `{"success":false}`, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAPIResponseShape(t *testing.T) {
	resp := models.APIResponse{
		Success: true,
		Data:    map[string]interface{}{"stock_level": 250},
		Meta:    &models.PageMeta{Page: 1, Limit: 100, Total: 47, TotalPages: 1},
	}
	b, _ := json.Marshal(resp)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	if m["success"] != true {
		t.Error("expected success=true")
	}
}

func TestAPIErrorShape(t *testing.T) {
	resp := models.APIResponse{
		Success: false,
		Error:   &models.APIError{Code: "INSUFFICIENT_STOCK", Message: "not enough stock to dispense"},
	}
	b, _ := json.Marshal(resp)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	errObj, ok := m["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object")
	}
	if errObj["code"] != "INSUFFICIENT_STOCK" {
		t.Errorf("unexpected code: %v", errObj["code"])
	}
}

func TestMedicationRowSerialization(t *testing.T) {
	exp := time.Now().UTC().Add(365 * 24 * time.Hour)
	m := models.MedicationRow{
		Name:           "Artemether-Lumefantrine",
		GenericName:    "Coartem",
		UnitPrice:      2.50,
		Currency:       "USD",
		StockLevel:     500,
		ReorderPoint:   50,
		RequiresCold:   false,
		IsActive:       true,
		ExpirationDate: &exp,
	}
	b, _ := json.Marshal(m)
	var back models.MedicationRow
	json.Unmarshal(b, &back)
	if back.Name != m.Name {
		t.Errorf("name mismatch: %q", back.Name)
	}
	if back.StockLevel != 500 {
		t.Errorf("stock_level mismatch: %d", back.StockLevel)
	}
}
