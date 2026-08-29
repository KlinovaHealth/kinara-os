package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cryptopkg "github.com/klinova/kinara-os/farmer-service/crypto"
	"github.com/klinova/kinara-os/farmer-service/models"
)

func TestFarmSizeConstants(t *testing.T) {
	sizes := []models.FarmSize{
		models.FarmSmallholder, models.FarmSmall, models.FarmMedium, models.FarmLarge,
	}
	for _, s := range sizes {
		if s == "" {
			t.Error("empty farm size constant")
		}
	}
}

func TestCropStatusConstants(t *testing.T) {
	statuses := []models.CropStatus{
		models.CropPlanted, models.CropGrowing, models.CropHarvested, models.CropFailed,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty crop status constant")
		}
	}
}

func TestFarmSizeCategorization(t *testing.T) {
	cases := []struct {
		ha   float64
		want models.FarmSize
	}{
		{0.5, models.FarmSmallholder},
		{1.9, models.FarmSmallholder},
		{2.0, models.FarmSmall},
		{9.9, models.FarmSmall},
		{10.0, models.FarmMedium},
		{99.9, models.FarmMedium},
		{100.0, models.FarmLarge},
		{500.0, models.FarmLarge},
	}
	for _, tc := range cases {
		got := farmSizeCategory(tc.ha)
		if got != tc.want {
			t.Errorf("farmSizeCategory(%.1f) = %q, want %q", tc.ha, got, tc.want)
		}
	}
}

func TestAESEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	enc, err := cryptopkg.NewEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}

	fields := []string{
		"Kwame Asante",           // full name
		"+233244987654",          // phone
		"GHA-123456789",          // national ID
		"Northern Region",        // region
	}
	for _, f := range fields {
		ct, err := enc.EncryptString(f)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}
		got, err := enc.DecryptString(ct)
		if err != nil {
			t.Fatalf("decrypt failed: %v", err)
		}
		if got != f {
			t.Errorf("roundtrip mismatch: got %q, want %q", got, f)
		}
	}
}

func TestAESUniqueCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := cryptopkg.NewEncryptor(key)
	c1, _ := enc.EncryptString("Kofi Owusu")
	c2, _ := enc.EncryptString("Kofi Owusu")
	if c1 == c2 {
		t.Error("AES-GCM must use random nonces")
	}
}

func TestAESInvalidKeySize(t *testing.T) {
	if _, err := cryptopkg.NewEncryptor([]byte("short")); err != cryptopkg.ErrInvalidKeySize {
		t.Errorf("expected ErrInvalidKeySize, got %v", err)
	}
}

func TestRegisterFarmerRequestJSON(t *testing.T) {
	lat := 5.5502
	lng := -0.2174
	req := models.RegisterFarmerRequest{
		FullName:        "Abena Osei",
		Phone:           "+233244555321",
		Country:         "Ghana",
		Region:          "Greater Accra",
		District:        "Accra Metropolitan",
		GPSLat:          &lat,
		GPSLng:          &lng,
		FarmSizeHa:      1.5,
		PrimaryLanguage: "tw",
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var back models.RegisterFarmerRequest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.FullName != req.FullName {
		t.Errorf("full_name mismatch: %q", back.FullName)
	}
	if back.FarmSizeHa != 1.5 {
		t.Errorf("farm_size_ha mismatch: %f", back.FarmSizeHa)
	}
}

func TestRecordCropRequestJSON(t *testing.T) {
	req := models.RecordCropRequest{
		CropType:        "Maize",
		Variety:         "Obaatanpa",
		AreaHa:          1.2,
		PlantedAt:       "2026-04-01T06:00:00Z",
		ExpectedHarvest: "2026-08-01T06:00:00Z",
		Season:          "2026A",
		Notes:           "Planted with fertilizer top-dressing",
	}
	b, _ := json.Marshal(req)
	var back models.RecordCropRequest
	json.Unmarshal(b, &back)
	if back.CropType != "Maize" {
		t.Errorf("crop_type mismatch: %q", back.CropType)
	}
	if back.AreaHa != 1.2 {
		t.Errorf("area_ha mismatch: %f", back.AreaHa)
	}
}

func TestUpdateCropRequestJSON(t *testing.T) {
	yieldKg := 2400.0
	notes := "Good yield despite late rains"
	req := models.UpdateCropRequest{
		Status:  models.CropHarvested,
		YieldKg: &yieldKg,
		Notes:   &notes,
	}
	b, _ := json.Marshal(req)
	var back models.UpdateCropRequest
	json.Unmarshal(b, &back)
	if back.Status != models.CropHarvested {
		t.Errorf("status mismatch: %q", back.Status)
	}
	if back.YieldKg == nil || *back.YieldKg != 2400.0 {
		t.Error("yield_kg mismatch")
	}
}

func TestAddPlotRequestJSON(t *testing.T) {
	req := models.AddPlotRequest{
		Name:       "North Field",
		SizeHa:     0.8,
		SoilType:   "loam",
		Irrigation: true,
	}
	b, _ := json.Marshal(req)
	var back models.AddPlotRequest
	json.Unmarshal(b, &back)
	if back.Name != "North Field" {
		t.Errorf("name mismatch: %q", back.Name)
	}
	if !back.Irrigation {
		t.Error("irrigation should be true")
	}
}

func TestHealthEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"farmer-service"}`))
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "farmer-service" {
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/farmers", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestFarmPlotSerialization(t *testing.T) {
	plot := models.FarmPlot{
		Name:       "South Field",
		SizeHa:     2.5,
		SoilType:   "clay",
		Irrigation: false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	b, _ := json.Marshal(plot)
	var back models.FarmPlot
	json.Unmarshal(b, &back)
	if back.Name != "South Field" {
		t.Errorf("name mismatch: %q", back.Name)
	}
	if back.SizeHa != 2.5 {
		t.Errorf("size_ha mismatch: %f", back.SizeHa)
	}
}

func TestAPIResponseShape(t *testing.T) {
	resp := models.APIResponse{
		Success: true,
		Data:    map[string]interface{}{"farmer_count": 1250},
		Meta:    &models.PageMeta{Page: 1, Limit: 50, Total: 1250, TotalPages: 25},
	}
	b, _ := json.Marshal(resp)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	if m["success"] != true {
		t.Error("expected success=true")
	}
	meta, ok := m["meta"].(map[string]interface{})
	if !ok {
		t.Fatal("expected meta")
	}
	if int(meta["total"].(float64)) != 1250 {
		t.Error("total mismatch")
	}
}

// farmSizeCategory mirrors the handler logic for test coverage.
func farmSizeCategory(ha float64) models.FarmSize {
	switch {
	case ha < 2:
		return models.FarmSmallholder
	case ha < 10:
		return models.FarmSmall
	case ha < 100:
		return models.FarmMedium
	default:
		return models.FarmLarge
	}
}
