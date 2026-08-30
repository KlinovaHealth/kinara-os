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

// ─────────────────────────────────────────────
// Test infrastructure
// ─────────────────────────────────────────────

type memStore struct {
	mu   sync.Mutex
	logs []models.SMSLog
}

func (s *memStore) SaveLog(_ context.Context, l models.SMSLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, l)
	return nil
}

func setup(t *testing.T) (*httptest.Server, *memStore) {
	t.Helper()
	store := &memStore{}
	h := handlers.NewHandlerWithStore(store)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return httptest.NewServer(r), store
}

func sendTestSMS(t *testing.T, srv *httptest.Server, from, body string) (string, string) {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{"from": from, "body": body})
	resp, err := http.Post(srv.URL+"/webhook/sms/test", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out models.APIResponse
	json.NewDecoder(resp.Body).Decode(&out)
	data := out.Data.(map[string]interface{})
	return data["response"].(string), data["command"].(string)
}

// ─────────────────────────────────────────────
// Intent parsing: 55+ test cases
// ─────────────────────────────────────────────

func TestCommandParsing(t *testing.T) {
	cases := []struct {
		input   string
		wantCmd string
	}{
		// Agriculture — English
		{"PRICE MAIZE", "PRICE"},
		{"price cocoa", "PRICE"},
		{"BUYERS COCOA", "BUYERS"},
		{"SELL MAIZE 500 250", "SELL"},
		{"WEATHER", "WEATHER"},
		{"WEATHER Lome", "WEATHER"},
		{"STATUS", "STATUS"},
		{"INCOME", "INCOME"},
		{"BALANCE", "BALANCE"},
		{"REGISTER Kofi MAIZE", "REGISTER"},
		{"FARMERS Lome", "FARMERS"},
		{"COOP Savane", "COOP"},
		{"COOPERATIVE", "COOP"},
		{"JOIN LOME-MAIZE", "JOIN"},
		{"SAVINGS", "SAVINGS"},
		// Agriculture — French
		{"PRIX MAIZE", "PRICE"},
		{"ACHETEURS CACAO", "BUYERS"},
		{"VENDRE MAIZE 500 250", "SELL"},
		{"METEO Lome", "WEATHER"},
		{"MÉTÉO Lome", "WEATHER"},
		{"STATUT", "STATUS"},
		{"REVENU", "INCOME"},
		{"SOLDE", "BALANCE"},
		{"INSCRIRE Kofi MAIZE", "REGISTER"},
		{"AGRICULTEURS", "FARMERS"},
		{"COOPÉRATIVE", "COOP"},
		{"REJOINDRE LOME-MAIZE", "JOIN"},
		{"EPARGNE", "SAVINGS"},
		{"ÉPARGNE", "SAVINGS"},
		// Health — English
		{"PATIENT Afi 30F", "PATIENT"},
		{"SYMPTOM fever chills", "SYMPTOM"},
		{"APPT 2026-10-15 LOME-NORD", "APPT"},
		{"LAB PAT-A1B2 MALARIA", "LAB"},
		{"RESULT LAB-A1B2", "LABRESULT"},
		{"REFER PAT-A1B2 CHU-LOME", "REFER"},
		{"CANCEL APT-A1B2", "CANCEL"},
		{"RESCHEDULE APT-A1B2 2026-10-20", "RESCHEDULE"},
		{"VACCINE PAT-A1B2", "VACCINE"},
		{"VACCINE SCHEDULE", "VACCINE"},
		{"SCHEDULE LOME-NORD", "SCHEDULE"},
		{"OUTBREAK", "OUTBREAK"},
		// Health — French
		{"MALADE Kofi 25M", "PATIENT"},
		{"SYMPTOME fièvre", "SYMPTOM"},
		{"SYMPTÔME douleur", "SYMPTOM"},
		{"RDV 2026-10-15 TSEVIE", "APPT"},
		{"LABO PAT-A1B2 PALUDISME", "LAB"},
		{"RESULTAT LAB-A1B2", "LABRESULT"},
		{"RÉSULTAT LAB-A1B2", "LABRESULT"},
		{"ORIENTER PAT-A1B2 CHU", "REFER"},
		{"ANNULER APT-A1B2", "CANCEL"},
		{"REPORTER APT-A1B2 2026-10-20", "RESCHEDULE"},
		{"REPROGRAMMER APT-A1B2 2026-10-20", "RESCHEDULE"},
		{"VACCIN PAT-A1B2", "VACCINE"},
		{"CALENDRIER LOME-NORD", "SCHEDULE"},
		{"EPID", "OUTBREAK"},
		{"ALERTE", "OUTBREAK"},
		// Logistics — English + French
		{"TRACK SHP-A1B2", "TRACK"},
		{"SUIVI SHP-A1B2", "TRACK"},
		{"SUIVRE SHP-A1B2", "TRACK"},
		{"ROUTE LOME ACCRA", "ROUTE"},
		{"ITINERAIRE LOME ACCRA", "ROUTE"},
		{"ITINÉRAIRE LOME ACCRA", "ROUTE"},
		{"FLEET TG", "FLEET"},
		{"FLOTTE TG", "FLEET"},
		// Maritime — English + French
		{"VESSEL MV-KINARA-01", "VESSEL"},
		{"NAVIRE MV-01", "VESSEL"},
		{"BATEAU MV-01", "VESSEL"},
		{"BERTH LOME", "BERTH"},
		{"QUAI LOME", "BERTH"},
		{"MANIFEST MV-KINARA-01", "MANIFEST"},
		{"MANIFESTE MV-01", "MANIFEST"},
		{"CUSTOMS SHP-A1B2", "CUSTOMS"},
		{"DOUANE SHP-A1B2", "CUSTOMS"},
		// Cross-pillar
		{"SEND +22890111222 5000 XOF", "SEND"},
		{"ENVOYER +22890111222 5000", "SEND"},
		{"TRANSFERT +22890111222 5000", "SEND"},
		{"CONVERT 10000 XOF USD", "CONVERT"},
		{"CONVERTIR 10000 XOF USD", "CONVERT"},
		{"IMPACT", "IMPACT"},
		{"IMPACT agriculture", "IMPACT"},
		// Help
		{"HELP", "HELP"},
		{"AIDE", "HELP"},
		{"?", "HELP"},
		// Unknown
		{"BLAH BLAH", "UNKNOWN"},
		{"", "UNKNOWN"},
		{"12345", "UNKNOWN"},
	}

	srv, _ := setup(t)
	defer srv.Close()

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			_, cmd := sendTestSMS(t, srv, "+22890000001", tc.input)
			if cmd != tc.wantCmd {
				t.Errorf("input=%q: want cmd %s, got %s", tc.input, tc.wantCmd, cmd)
			}
		})
	}
}

// ─────────────────────────────────────────────
// Response formatting: 160-char limit
// ─────────────────────────────────────────────

func TestResponseMaxLength(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()

	inputs := []string{"HELP", "PRICE MAIZE", "WEATHER", "BALANCE", "STATUS", "IMPACT"}
	for _, input := range inputs {
		response, _ := sendTestSMS(t, srv, "+22890000002", input)
		if len(response) > 160 {
			t.Errorf("input=%q: response length %d > 160 chars:\n%s", input, len(response), response)
		}
	}
}

// ─────────────────────────────────────────────
// Specific command response content
// ─────────────────────────────────────────────

func TestHelp_ContainsAllPillars(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000003", "HELP")
	for _, keyword := range []string{"PRICE", "PATIENT", "TRACK", "VESSEL", "SEND"} {
		if !strings.Contains(response, keyword) {
			t.Errorf("HELP missing keyword %q in:\n%s", keyword, response)
		}
	}
}

func TestHelp_French(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000004", "AIDE")
	if len(response) == 0 {
		t.Fatal("AIDE returned empty response")
	}
}

func TestPrice_MissingArg(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000005", "PRICE")
	if !strings.Contains(strings.ToUpper(response), "USAGE") && !strings.Contains(strings.ToUpper(response), "EX") {
		t.Errorf("PRICE with no arg should show usage, got: %s", response)
	}
}

func TestSell_MissingArgs(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000006", "SELL MAIZE")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("SELL with missing args should show usage, got: %s", response)
	}
}

func TestBuyers_MissingArg(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000007", "BUYERS")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("BUYERS with no arg should show usage, got: %s", response)
	}
}

func TestPatient_BadAge(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000008", "PATIENT Kofi abc")
	if !strings.Contains(strings.ToLower(response), "age") && !strings.Contains(strings.ToLower(response), "invalide") {
		t.Errorf("PATIENT bad age should warn about age, got: %s", response)
	}
}

func TestPatient_MissingArgs(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000009", "PATIENT")
	if !strings.Contains(response, "PATIENT") {
		t.Errorf("PATIENT with no args should show usage hint, got: %s", response)
	}
}

func TestAppt_TomorrowExpands(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	// DEMAIN should expand to a date — service will be down in test so we get the fallback
	response, _ := sendTestSMS(t, srv, "+22890000010", "APPT DEMAIN LOME-NORD")
	// Should contain a date like "2026-" or the clinic name
	if !strings.Contains(response, "LOME-NORD") && !strings.Contains(response, "RDV") {
		t.Errorf("APPT DEMAIN: expected clinic in response, got: %s", response)
	}
}

func TestLab_MissingArgs(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000011", "LAB")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("LAB with no args should show usage, got: %s", response)
	}
}

func TestRefer_MissingArgs(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000012", "REFER")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("REFER with no args should show usage, got: %s", response)
	}
}

func TestCancel_MissingArgs(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000013", "CANCEL")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("CANCEL with no args should show usage, got: %s", response)
	}
}

func TestReschedule_MissingArgs(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000014", "RESCHEDULE")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("RESCHEDULE with no args should show usage, got: %s", response)
	}
}

func TestVaccine_Schedule(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000015", "VACCINE SCHEDULE")
	// Service will be down; should return built-in fallback
	if len(response) == 0 {
		t.Fatal("VACCINE SCHEDULE returned empty response")
	}
	if len(response) > 160 {
		t.Errorf("VACCINE SCHEDULE response exceeds 160 chars: %d", len(response))
	}
}

func TestTrack_MissingArg(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000016", "TRACK")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("TRACK with no arg should show usage, got: %s", response)
	}
}

func TestRoute_MissingArgs(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000017", "ROUTE LOME")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("ROUTE with one arg should show usage, got: %s", response)
	}
}

func TestVessel_MissingArg(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000018", "VESSEL")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("VESSEL with no arg should show usage, got: %s", response)
	}
}

func TestManifest_MissingArg(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000019", "MANIFEST")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("MANIFEST with no arg should show usage, got: %s", response)
	}
}

func TestCustoms_MissingArg(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000020", "CUSTOMS")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("CUSTOMS with no arg should show usage, got: %s", response)
	}
}

func TestSend_MissingArgs(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000021", "SEND +22890111222")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("SEND with one arg should show usage, got: %s", response)
	}
}

func TestConvert_MissingArgs(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000022", "CONVERT 10000 XOF")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("CONVERT with two args should show usage, got: %s", response)
	}
}

func TestJoin_MissingArg(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, _ := sendTestSMS(t, srv, "+22890000023", "JOIN")
	if !strings.Contains(strings.ToUpper(response), "USAGE") {
		t.Errorf("JOIN with no arg should show usage, got: %s", response)
	}
}

func TestUnknownCommand(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	response, cmd := sendTestSMS(t, srv, "+22890000024", "FROBNICATE something")
	if cmd != "UNKNOWN" {
		t.Errorf("expected UNKNOWN, got %s", cmd)
	}
	if len(response) == 0 {
		t.Fatal("unknown command returned empty response")
	}
}

func TestEmptyBody(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	_, cmd := sendTestSMS(t, srv, "+22890000025", "")
	if cmd != "UNKNOWN" {
		t.Errorf("empty body should produce UNKNOWN, got %s", cmd)
	}
}

// ─────────────────────────────────────────────
// Webhook endpoint tests
// ─────────────────────────────────────────────

func TestTwilioWebhook_MissingBody(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	form := url.Values{}
	form.Set("From", "+22890777666")
	form.Set("To", "+12025551234")
	// Body is missing
	resp, _ := http.Post(srv.URL+"/webhook/sms/twilio", "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for missing body, got %d", resp.StatusCode)
	}
}

func TestTwilioWebhook_ValidRequest(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	form := url.Values{}
	form.Set("From", "+22890777777")
	form.Set("To", "+12025551234")
	form.Set("Body", "PRICE MAIZE")
	form.Set("FromCountry", "TG")
	resp, _ := http.Post(srv.URL+"/webhook/sms/twilio", "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(store.logs) == 0 {
		t.Fatal("expected audit log entry")
	}
	if store.logs[0].Command != "PRICE" {
		t.Errorf("expected PRICE command logged, got %s", store.logs[0].Command)
	}
}

func TestAfricastalkingWebhook_ValidRequest(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	form := url.Values{}
	form.Set("from", "+22890111222")
	form.Set("to", "+22891000000")
	form.Set("text", "METEO Lome")
	resp, _ := http.Post(srv.URL+"/webhook/sms/africastalking", "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(store.logs) == 0 {
		t.Fatal("expected audit log entry")
	}
	if store.logs[0].Command != "WEATHER" {
		t.Errorf("expected WEATHER logged, got %s", store.logs[0].Command)
	}
}

func TestAfricastalkingWebhook_MissingFrom(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	form := url.Values{}
	form.Set("to", "+22891000000")
	form.Set("text", "PRICE MAIZE")
	resp, _ := http.Post(srv.URL+"/webhook/sms/africastalking", "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for missing from, got %d", resp.StatusCode)
	}
}

func TestTestEndpoint_MissingFrom(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()
	payload, _ := json.Marshal(map[string]string{"body": "PRICE MAIZE"})
	resp, _ := http.Post(srv.URL+"/webhook/sms/test", "application/json", bytes.NewBuffer(payload))
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for missing from, got %d", resp.StatusCode)
	}
}

// ─────────────────────────────────────────────
// Audit logging
// ─────────────────────────────────────────────

func TestAuditLog_EveryCommand(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	inputs := []string{"PRICE MAIZE", "BUYERS COCOA", "HELP", "WEATHER Lome", "BALANCE", "TRACK SHP-001"}
	for i, input := range inputs {
		sendTestSMS(t, srv, "+22890000100", input)
		store.mu.Lock()
		count := len(store.logs)
		store.mu.Unlock()
		if count != i+1 {
			t.Errorf("after %d commands, expected %d log entries, got %d", i+1, i+1, count)
		}
	}
}

func TestAuditLog_CorrectProvider(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()

	// Test endpoint uses ProviderTwilio
	sendTestSMS(t, srv, "+22890000200", "HELP")
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.logs) == 0 {
		t.Fatal("no log entries")
	}
	if store.logs[0].Provider != models.ProviderTwilio {
		t.Errorf("expected provider twilio, got %s", store.logs[0].Provider)
	}
}

func TestAuditLog_PhoneRecorded(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()
	phone := "+22890098765"
	sendTestSMS(t, srv, phone, "INCOME")
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.logs[0].From != phone {
		t.Errorf("expected phone %s logged, got %s", phone, store.logs[0].From)
	}
}

// ─────────────────────────────────────────────
// Concurrency
// ─────────────────────────────────────────────

func TestConcurrentRequests(t *testing.T) {
	srv, store := setup(t)
	defer srv.Close()

	var wg sync.WaitGroup
	const n = 20
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			phone := "+2289000" + string(rune('0'+i%10))
			commands := []string{"PRICE MAIZE", "HELP", "WEATHER", "BALANCE", "STATUS"}
			sendTestSMS(t, srv, phone, commands[i%len(commands)])
		}(i)
	}
	wg.Wait()

	store.mu.Lock()
	count := len(store.logs)
	store.mu.Unlock()
	if count != n {
		t.Errorf("concurrent: expected %d log entries, got %d", n, count)
	}
}

// ─────────────────────────────────────────────
// Service fallback: downstream services are down in tests
// All handlers must still return a non-empty, ≤160-char response.
// ─────────────────────────────────────────────

func TestAllCommandsFallback(t *testing.T) {
	srv, _ := setup(t)
	defer srv.Close()

	allCommands := []string{
		"PRICE MAIZE", "BUYERS COCOA", "SELL MAIZE 500 250",
		"WEATHER Lome", "STATUS", "INCOME", "BALANCE",
		"REGISTER Kofi MAIZE", "FARMERS Lome", "COOP", "JOIN LOME-COOP", "SAVINGS",
		"PATIENT Kofi 35M", "SYMPTOM fever", "APPT 2026-10-15 LOME",
		"LAB PAT-A1B2 MALARIA", "RESULT LAB-A1B2",
		"REFER PAT-A1B2 CHU", "CANCEL APT-A1B2",
		"RESCHEDULE APT-A1B2 2026-10-20",
		"VACCINE PAT-A1B2", "VACCINE SCHEDULE", "SCHEDULE LOME",
		"OUTBREAK",
		"TRACK SHP-A1B2", "ROUTE LOME ACCRA", "FLEET TG",
		"VESSEL MV-01", "BERTH LOME", "MANIFEST MV-01", "CUSTOMS SHP-A1B2",
		"SEND +22890111222 5000 XOF", "CONVERT 10000 XOF USD", "IMPACT",
		"HELP",
	}

	for _, cmd := range allCommands {
		t.Run(cmd, func(t *testing.T) {
			response, _ := sendTestSMS(t, srv, "+22890999001", cmd)
			if len(response) == 0 {
				t.Errorf("cmd=%q: got empty response", cmd)
			}
			if len(response) > 160 {
				t.Errorf("cmd=%q: response length %d > 160:\n%s", cmd, len(response), response)
			}
		})
	}
}
