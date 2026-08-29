package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/sms-gateway/handlers"
	"github.com/klinova/kinara-os/sms-gateway/models"
)

type memStore struct {
	mu   sync.Mutex
	logs []models.SMSLog
}

func (s *memStore) SaveLog(_ context.Context, l models.SMSLog) error {
	s.mu.Lock(); defer s.mu.Unlock(); s.logs = append(s.logs, l); return nil
}

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := &memStore{}
	h := handlers.NewHandlerWithStore(store)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return httptest.NewServer(r), store
}

func TestTwilioWebhook_Price(t *testing.T) {
	// Use test endpoint instead of Twilio webhook (avoids needing external service)
	srv, store := setup(t)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"from": "+22890123456", "body": "PRICE MAIZE"})
	resp, _ := http.Post(srv.URL+"/webhook/sms/test", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success { t.Fatal("expected success") }
	data := out.Data.(map[string]interface{})
	if data["command"].(string) != "PRICE" { t.Fatalf("expected PRICE command, got %s", data["command"]) }
	if len(store.logs) == 0 { t.Fatal("expected SMS log entry") }
}

func TestTwilioWebhook_Help(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(map[string]string{"from": "+22890999888", "body": "HELP"})
	resp, _ := http.Post(srv.URL+"/webhook/sms/test", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	response := data["response"].(string)
	if !strings.Contains(response, "KINARA COMMANDS") { t.Fatal("help text missing") }
}

func TestAfricastalkingWebhook(t *testing.T) {
	srv, store := setup(t); defer srv.Close()
	form := url.Values{}
	form.Set("from", "+22890111222")
	form.Set("to", "+22891000000")
	form.Set("text", "METEO Lome")
	resp, _ := http.Post(srv.URL+"/webhook/sms/africastalking", "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	if len(store.logs) == 0 { t.Fatal("expected log entry") }
	if store.logs[0].Command != "WEATHER" { t.Fatalf("expected WEATHER, got %s", store.logs[0].Command) }
}

func TestTwilioWebhookForm(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	form := url.Values{}
	form.Set("From", "+22890777666")
	form.Set("To", "+12025551234")
	form.Set("Body", "WEATHER Lome")
	form.Set("FromCountry", "TG")
	resp, _ := http.Post(srv.URL+"/webhook/sms/twilio", "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	respBody := new(strings.Builder)
	if b, err := json.Marshal(resp.Body); err == nil { _ = b }
	_ = respBody
}

func TestCommandParsing(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"PRICE MAIZE", "PRICE"},
		{"price cocoa", "PRICE"},
		{"PRIX MAIZE", "PRICE"},   // French
		{"SELL MAIZE 500 250", "SELL"},
		{"VENDRE MAIZE 500 250", "SELL"}, // French
		{"WEATHER", "WEATHER"},
		{"METEO Lome", "WEATHER"}, // French
		{"STATUS", "STATUS"},
		{"HELP", "HELP"},
		{"AIDE", "HELP"},          // French
		{"REGISTER Kofi MAIZE", "REGISTER"},
		{"BLAH BLAH", "UNKNOWN"},
		{"", "UNKNOWN"},
	}
	for _, tc := range cases {
		srv, _ := setup(t); defer srv.Close()
		body, _ := json.Marshal(map[string]string{"from": "+22890111111", "body": tc.input})
		resp, _ := http.Post(srv.URL+"/webhook/sms/test", "application/json", bytes.NewBuffer(body))
		if resp.StatusCode != 200 { t.Fatalf("input=%q: expected 200, got %d", tc.input, resp.StatusCode) }
		var out models.APIResponse
		json.NewDecoder(resp.Body).Decode(&out)
		data := out.Data.(map[string]interface{})
		if data["command"].(string) != tc.expected {
			t.Errorf("input=%q: expected command %s, got %s", tc.input, tc.expected, data["command"])
		}
	}
}

func TestSellCommand_MissingArgs(t *testing.T) {
	srv, _ := setup(t); defer srv.Close()
	body, _ := json.Marshal(map[string]string{"from": "+22890444333", "body": "SELL MAIZE"})
	resp, _ := http.Post(srv.URL+"/webhook/sms/test", "application/json", bytes.NewBuffer(body))
	if resp.StatusCode != 200 { t.Fatalf("expected 200, got %d", resp.StatusCode) }
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	if !strings.Contains(data["response"].(string), "Usage") { t.Fatal("expected usage hint in response") }
}
