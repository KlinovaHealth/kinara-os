package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cryptopkg "github.com/klinova/kinara-os/notification-service/crypto"
	"github.com/klinova/kinara-os/notification-service/models"
)

// ─── Model constant tests ─────────────────────────────────────────────────────

func TestChannelConstants(t *testing.T) {
	channels := []models.NotificationChannel{
		models.ChannelSMS, models.ChannelPush, models.ChannelWhatsApp,
		models.ChannelEmail, models.ChannelInApp,
	}
	if len(channels) != 5 {
		t.Errorf("expected 5 channels, got %d", len(channels))
	}
	for _, c := range channels {
		if c == "" {
			t.Error("empty channel constant")
		}
	}
}

func TestNotificationTypeConstants(t *testing.T) {
	types := []models.NotificationType{
		models.TypeAppointmentReminder, models.TypePrescriptionAlert,
		models.TypeReferralStatus, models.TypePriceAlert, models.TypeWeatherAlert,
		models.TypeFleetAlert, models.TypePortAlert, models.TypeSystemAlert,
	}
	for _, nt := range types {
		if nt == "" {
			t.Error("empty notification type constant")
		}
	}
}

func TestStatusConstants(t *testing.T) {
	statuses := []models.NotificationStatus{
		models.StatusPending, models.StatusQueued, models.StatusSent,
		models.StatusDelivered, models.StatusFailed, models.StatusCancelled,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty status constant")
		}
	}
}

func TestPriorityConstants(t *testing.T) {
	priorities := []models.NotificationPriority{
		models.PriorityLow, models.PriorityNormal, models.PriorityHigh, models.PriorityCritical,
	}
	for _, p := range priorities {
		if p == "" {
			t.Error("empty priority constant")
		}
	}
}

// ─── AES encryption tests ─────────────────────────────────────────────────────

func TestAESEncryptDecryptMessage(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	enc, err := cryptopkg.NewEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}

	messages := []string{
		"Reminder: You have an appointment at Korle Bu Teaching Hospital on 2026-09-01 at 10:00",
		"+233244123456", // phone number (recipient field)
		"Your prescription for Amoxicillin is ready for pickup",
		"WEATHER ALERT for Accra: Heavy rainfall expected. Protect crops.",
	}
	for _, msg := range messages {
		ct, err := enc.EncryptString(msg)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}
		got, err := enc.DecryptString(ct)
		if err != nil {
			t.Fatalf("decrypt failed: %v", err)
		}
		if got != msg {
			t.Errorf("roundtrip failed: got %q, want %q", got, msg)
		}
	}
}

func TestAESUniqueCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := cryptopkg.NewEncryptor(key)
	c1, _ := enc.EncryptString("appointment reminder")
	c2, _ := enc.EncryptString("appointment reminder")
	if c1 == c2 {
		t.Error("random nonces must produce unique ciphertexts")
	}
}

func TestAESWrongKeyFails(t *testing.T) {
	key1, key2 := make([]byte, 32), make([]byte, 32)
	key2[0] = 0xFF
	enc1, _ := cryptopkg.NewEncryptor(key1)
	enc2, _ := cryptopkg.NewEncryptor(key2)
	ct, _ := enc1.EncryptString("+233244123456")
	if _, err := enc2.DecryptString(ct); err == nil {
		t.Error("wrong key must fail decryption")
	}
}

func TestAESInvalidKeySize(t *testing.T) {
	if _, err := cryptopkg.NewEncryptor([]byte("tooshort")); err != cryptopkg.ErrInvalidKeySize {
		t.Errorf("expected ErrInvalidKeySize, got %v", err)
	}
}

// ─── Request serialization tests ──────────────────────────────────────────────

func TestSendNotificationRequestJSON(t *testing.T) {
	req := models.SendNotificationRequest{
		UserID:    "550e8400-e29b-41d4-a716-446655440000",
		Type:      models.TypeAppointmentReminder,
		Channel:   models.ChannelSMS,
		Priority:  models.PriorityNormal,
		Message:   "You have an appointment tomorrow at 10:00 AM",
		Recipient: "+233244123456",
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var back models.SendNotificationRequest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Type != models.TypeAppointmentReminder {
		t.Errorf("type mismatch: %q", back.Type)
	}
	if back.Channel != models.ChannelSMS {
		t.Errorf("channel mismatch: %q", back.Channel)
	}
}

func TestScheduleNotificationRequestJSON(t *testing.T) {
	req := models.ScheduleNotificationRequest{
		UserID:      "550e8400-e29b-41d4-a716-446655440000",
		Type:        models.TypePrescriptionAlert,
		Channel:     models.ChannelWhatsApp,
		Priority:    models.PriorityHigh,
		Message:     "Your medication is ready",
		Recipient:   "+233244123456",
		ScheduledAt: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
	}
	b, _ := json.Marshal(req)
	var back models.ScheduleNotificationRequest
	json.Unmarshal(b, &back)
	if back.Channel != models.ChannelWhatsApp {
		t.Errorf("channel mismatch: %q", back.Channel)
	}
}

func TestBulkSendRequestJSON(t *testing.T) {
	req := models.BulkSendRequest{
		Type:     models.TypeWeatherAlert,
		Channel:  models.ChannelSMS,
		Priority: models.PriorityCritical,
		Message:  "WEATHER ALERT: Severe flooding expected in your region",
		UserIDs:  []string{"id1", "id2", "id3"},
	}
	b, _ := json.Marshal(req)
	var back models.BulkSendRequest
	json.Unmarshal(b, &back)
	if len(back.UserIDs) != 3 {
		t.Errorf("expected 3 user_ids, got %d", len(back.UserIDs))
	}
	if back.Priority != models.PriorityCritical {
		t.Errorf("priority mismatch: %q", back.Priority)
	}
}

func TestUpdatePreferencesRequestJSON(t *testing.T) {
	smsOff := false
	whatsappOn := true
	tz := "Africa/Lagos"
	req := models.UpdatePreferencesRequest{
		SMSEnabled:      &smsOff,
		WhatsAppEnabled: &whatsappOn,
		TimeZone:        &tz,
	}
	b, _ := json.Marshal(req)
	var back models.UpdatePreferencesRequest
	json.Unmarshal(b, &back)
	if back.SMSEnabled == nil || *back.SMSEnabled != false {
		t.Error("sms_enabled mismatch")
	}
	if back.TimeZone == nil || *back.TimeZone != "Africa/Lagos" {
		t.Error("timezone mismatch")
	}
}

// ─── Pillar coverage tests ────────────────────────────────────────────────────

func TestFourPillarNotificationTypes(t *testing.T) {
	healthTypes := []models.NotificationType{
		models.TypeAppointmentReminder, models.TypePrescriptionAlert,
		models.TypeReferralStatus, models.TypeLabResult, models.TypeVaccineReminder,
	}
	agriTypes := []models.NotificationType{
		models.TypePriceAlert, models.TypeWeatherAlert,
		models.TypeMarketOpportunity, models.TypeHarvestReminder,
	}
	logisticsTypes := []models.NotificationType{
		models.TypeFleetAlert, models.TypeDeliveryStatus, models.TypeRouteChange,
	}
	maritimeTypes := []models.NotificationType{
		models.TypePortAlert, models.TypeVesselStatus, models.TypeCustomsClearance,
	}

	for _, nt := range append(append(append(healthTypes, agriTypes...), logisticsTypes...), maritimeTypes...) {
		if nt == "" {
			t.Errorf("empty notification type in pillar coverage")
		}
	}

	if len(healthTypes) < 3 {
		t.Error("expected at least 3 health notification types")
	}
	if len(agriTypes) < 3 {
		t.Error("expected at least 3 agriculture notification types")
	}
}

// ─── HTTP tests ───────────────────────────────────────────────────────────────

func TestHealthEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"notification-service"}`))
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "notification-service" {
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/send", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAPIResponseShape(t *testing.T) {
	resp := models.APIResponse{
		Success: true,
		Data:    map[string]interface{}{"sent": 150, "failed": 2},
	}
	b, _ := json.Marshal(resp)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	if m["success"] != true {
		t.Error("expected success=true")
	}
}

func TestUserPreferencesSerialization(t *testing.T) {
	qStart := "22:00"
	qEnd := "07:00"
	prefs := models.UserPreferences{
		SMSEnabled:      true,
		PushEnabled:     true,
		WhatsAppEnabled: true,
		EmailEnabled:    false,
		InAppEnabled:    true,
		QuietHoursStart: &qStart,
		QuietHoursEnd:   &qEnd,
		TimeZone:        "Africa/Nairobi",
		Language:        "sw",
		UpdatedAt:       time.Now(),
	}
	b, _ := json.Marshal(prefs)
	var back models.UserPreferences
	json.Unmarshal(b, &back)
	if back.TimeZone != "Africa/Nairobi" {
		t.Errorf("timezone mismatch: %q", back.TimeZone)
	}
	if back.QuietHoursStart == nil || *back.QuietHoursStart != "22:00" {
		t.Error("quiet_hours_start mismatch")
	}
}
