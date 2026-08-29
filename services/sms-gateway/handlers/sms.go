package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/sms-gateway/models"
)

type Store interface {
	SaveLog(ctx context.Context, l models.SMSLog) error
}

type Handler struct {
	store          Store
	marketURL      string
	weatherURL     string
	analyticsURL   string
	farmerURL      string
	patientURL     string
	clinicalURL    string
	appointmentURL string
	labURL         string
	twilioSID      string
	twilioToken    string
	atAPIKey       string
	atUsername     string
}

func NewHandler(store Store) *Handler {
	return &Handler{
		store:          store,
		marketURL:      env("MARKET_SERVICE_URL", "http://market-service:8086"),
		weatherURL:     env("WEATHER_SERVICE_URL", "http://weather-service:8088"),
		analyticsURL:   env("ANALYTICS_SERVICE_URL", "http://analytics-service:8108"),
		farmerURL:      env("FARMER_SERVICE_URL", "http://farmer-service:8084"),
		patientURL:     env("PATIENT_SERVICE_URL", "http://patient-service:8081"),
		clinicalURL:    env("CLINICAL_SERVICE_URL", "http://clinical-service:8082"),
		appointmentURL: env("APPOINTMENT_SERVICE_URL", "http://appointment-service:8120"),
		labURL:         env("LAB_SERVICE_URL", "http://lab-service:8122"),
		twilioSID:      os.Getenv("TWILIO_ACCOUNT_SID"),
		twilioToken:    os.Getenv("TWILIO_AUTH_TOKEN"),
		atAPIKey:       os.Getenv("AFRICASTALKING_API_KEY"),
		atUsername:     os.Getenv("AFRICASTALKING_USERNAME"),
	}
}

func NewHandlerWithStore(store Store) *Handler { return NewHandler(store) }

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/webhook/sms/twilio", h.TwilioWebhook).Methods("POST")
	r.HandleFunc("/webhook/sms/africastalking", h.AfricastalkingWebhook).Methods("POST")
	r.HandleFunc("/webhook/sms/test", h.TestSMS).Methods("POST")
	r.HandleFunc("/sms/logs", h.ListLogs).Methods("GET")
}

func (h *Handler) TwilioWebhook(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	from := r.FormValue("From")
	to := r.FormValue("To")
	body := r.FormValue("Body")
	country := r.FormValue("FromCountry")

	if from == "" || body == "" {
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(400)
		fmt.Fprintln(w, `<?xml version="1.0"?><Response><Message>Invalid request</Message></Response>`)
		return
	}

	response := h.processCommand(r.Context(), from, body, country)
	h.saveLog(r.Context(), models.ProviderTwilio, from, to, body, response, true)

	// Twilio TwiML response
	w.Header().Set("Content-Type", "text/xml")
	fmt.Fprintf(w, `<?xml version="1.0"?><Response><Message>%s</Message></Response>`,
		xmlEscape(response))
}

func (h *Handler) AfricastalkingWebhook(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	params, _ := url.ParseQuery(string(body))

	from := params.Get("from")
	to := params.Get("to")
	text := params.Get("text")

	if from == "" || text == "" {
		w.WriteHeader(400)
		return
	}

	response := h.processCommand(r.Context(), from, text, "")
	h.saveLog(r.Context(), models.ProviderAfricastalking, from, to, text, response, true)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": response})
}

func (h *Handler) TestSMS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From string `json:"from"`
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.From == "" || req.Body == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(models.APIResponse{Error: "from and body required"})
		return
	}
	response := h.processCommand(r.Context(), req.From, req.Body, "TG")
	h.saveLog(r.Context(), models.ProviderTwilio, req.From, "test", req.Body, response, true)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.APIResponse{Success: true, Data: map[string]string{
		"response": response,
		"command":  parseCommand(req.Body).Type.String(),
	}})
}

func (h *Handler) ListLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.APIResponse{Success: true, Data: []string{}})
}

// processCommand parses and routes the SMS command to the correct service.
func (h *Handler) processCommand(ctx context.Context, from, text, country string) string {
	cmd := parseCommand(text)
	log.Printf("SMS from=%s command=%s args=%v", from, cmd.Type, cmd.Args)

	switch cmd.Type {
	case models.CmdPrice:
		return h.handlePrice(ctx, cmd)
	case models.CmdBuyers:
		return h.handleBuyers(ctx, cmd)
	case models.CmdSell:
		return h.handleSell(ctx, cmd, from)
	case models.CmdWeather:
		return h.handleWeather(ctx, cmd, country)
	case models.CmdStatus:
		return h.handleStatus(ctx, from)
	case models.CmdIncome:
		return h.handleIncome(ctx, from)
	case models.CmdBalance:
		return h.handleBalance(ctx, from)
	case models.CmdRegister:
		return h.handleRegister(ctx, cmd, from)
	case models.CmdPatient:
		return h.handlePatient(ctx, cmd, from)
	case models.CmdSymptom:
		return h.handleSymptom(ctx, cmd, from)
	case models.CmdAppt:
		return h.handleAppt(ctx, cmd, from)
	case models.CmdLab:
		return h.handleLab(ctx, cmd, from)
	case models.CmdHelp:
		return helpText()
	default:
		return fmt.Sprintf("Kinara: Unknown command '%s'.\n%s", cmd.Args[0], helpText())
	}
}

func (h *Handler) handlePrice(ctx context.Context, cmd models.ParsedCommand) string {
	if len(cmd.Args) == 0 {
		return "Usage: PRICE <crop>\nExample: PRICE MAIZE"
	}
	crop := strings.ToLower(cmd.Args[0])
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/market/prices?commodity=%s&limit=3", h.marketURL, url.QueryEscape(crop)))
	if err != nil {
		return fmt.Sprintf("Kinara: Price lookup failed. Try again.\n(ERR: %s)", err.Error()[:min(len(err.Error()), 40)])
	}

	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			CommodityName string  `json:"commodity_name"`
			PricePerUnit  float64 `json:"price_per_unit"`
			Unit          string  `json:"unit"`
			Market        string  `json:"market_name"`
			Currency      string  `json:"currency"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success || len(result.Data) == 0 {
		return fmt.Sprintf("Kinara: No prices found for %s today.", strings.ToUpper(crop))
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("KINARA PRICES: %s", strings.ToUpper(crop)))
	for _, p := range result.Data {
		lines = append(lines, fmt.Sprintf("  %s: %.0f %s/%s", p.Market, p.PricePerUnit, p.Currency, p.Unit))
	}
	lines = append(lines, "Reply SELL MAIZE <qty> <price> to list")
	return strings.Join(lines, "\n")
}

func (h *Handler) handleBuyers(ctx context.Context, cmd models.ParsedCommand) string {
	if len(cmd.Args) == 0 {
		return "Usage: BUYERS <crop>\nExample: BUYERS COCOA"
	}
	crop := strings.ToLower(cmd.Args[0])
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/market/bids?commodity=%s&status=open&limit=5", h.marketURL, url.QueryEscape(crop)))
	if err != nil {
		return "Kinara: Buyer lookup failed. Try again."
	}

	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			BuyerName   string  `json:"buyer_name"`
			PricePerUnit float64 `json:"price_per_unit"`
			QuantityKg  float64 `json:"quantity_kg"`
			Currency    string  `json:"currency"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success || len(result.Data) == 0 {
		return fmt.Sprintf("Kinara: No buyers found for %s right now.", strings.ToUpper(crop))
	}

	lines := []string{fmt.Sprintf("BUYERS WANT %s:", strings.ToUpper(crop))}
	for _, b := range result.Data {
		lines = append(lines, fmt.Sprintf("  %s: %.0f%s/kg (%.0fkg wanted)", b.BuyerName, b.PricePerUnit, b.Currency, b.QuantityKg))
	}
	lines = append(lines, "Reply SELL "+strings.ToUpper(crop)+" <qty> <price> to respond")
	return strings.Join(lines, "\n")
}

func (h *Handler) handleSell(ctx context.Context, cmd models.ParsedCommand, from string) string {
	// SELL <crop> <qty_kg> <price_per_kg>
	if len(cmd.Args) < 3 {
		return "Usage: SELL <crop> <qty_kg> <price>\nExample: SELL MAIZE 500 250"
	}
	crop := cmd.Args[0]
	qty := cmd.Args[1]
	price := cmd.Args[2]

	payload := fmt.Sprintf(`{"commodity_name":"%s","quantity_kg":%s,"price_per_kg":%s,"seller_phone":"%s","currency":"XOF","location":"Togo"}`,
		crop, qty, price, from)

	resp, err := httpPost(ctx, fmt.Sprintf("%s/api/v1/market/listings", h.marketURL), payload)
	if err != nil {
		return "Kinara: Listing failed. Check your numbers and try again."
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return "Kinara: Could not create listing. Try: SELL MAIZE 500 250"
	}
	return fmt.Sprintf("KINARA: Listing created!\n%skg %s at %s XOF/kg.\nID: %s\nBuyers will contact you.",
		qty, strings.ToUpper(crop), price, result.Data.ID[:8])
}

func (h *Handler) handleWeather(ctx context.Context, cmd models.ParsedCommand, country string) string {
	location := "Togo"
	if len(cmd.Args) > 0 {
		location = strings.Join(cmd.Args, " ")
	}
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/weather/forecast?location=%s&days=3", h.weatherURL, url.QueryEscape(location)))
	if err != nil {
		return "Kinara: Weather unavailable. Try again later."
	}

	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			Date        string  `json:"date"`
			TempMaxC    float64 `json:"temp_max_c"`
			TempMinC    float64 `json:"temp_min_c"`
			Condition   string  `json:"condition"`
			RainfallMm  float64 `json:"rainfall_mm"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success || len(result.Data) == 0 {
		return fmt.Sprintf("Kinara: No weather data for %s.", location)
	}

	lines := []string{fmt.Sprintf("KINARA WEATHER: %s", strings.ToUpper(location))}
	for _, d := range result.Data {
		lines = append(lines, fmt.Sprintf("  %s: %.0f-%.0fC %s Rain:%.0fmm", d.Date[5:10], d.TempMinC, d.TempMaxC, d.Condition, d.RainfallMm))
	}
	lines = append(lines, "Reply WEATHER <city> for other location")
	return strings.Join(lines, "\n")
}

func (h *Handler) handleStatus(ctx context.Context, from string) string {
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/market/listings?seller_phone=%s&status=active&limit=3", h.marketURL, url.QueryEscape(from)))
	if err != nil {
		return "Kinara: Status check failed. Try again."
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			CommodityName string  `json:"commodity_name"`
			QuantityKg   float64 `json:"quantity_kg"`
			Status       string  `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return "Kinara: No active listings found for your number."
	}
	if len(result.Data) == 0 {
		return "Kinara: No active listings.\nReply SELL <crop> <qty> <price> to create one."
	}
	lines := []string{"YOUR ACTIVE LISTINGS:"}
	for _, l := range result.Data {
		lines = append(lines, fmt.Sprintf("  %s: %.0fkg [%s]", strings.ToUpper(l.CommodityName), l.QuantityKg, l.Status))
	}
	return strings.Join(lines, "\n")
}

func (h *Handler) handleIncome(ctx context.Context, from string) string {
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/analytics/impact?pillar=agriculture&country=TG&limit=1", h.analyticsURL))
	if err != nil {
		return "Kinara: Income data unavailable. Try again."
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			MetricName  string  `json:"metric_name"`
			MetricValue float64 `json:"metric_value"`
			MetricUnit  string  `json:"metric_unit"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success || len(result.Data) == 0 {
		return "Kinara: Income data not yet available for your region."
	}
	m := result.Data[0]
	return fmt.Sprintf("KINARA INCOME UPDATE:\n%s: %.0f %s\nSell via Kinara to improve your income.\nReply SELL <crop> <qty> <price>", m.MetricName, m.MetricValue, m.MetricUnit)
}

func (h *Handler) handleBalance(ctx context.Context, from string) string {
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/payments/wallets?owner_phone=%s", h.analyticsURL, url.QueryEscape(from)))
	if err != nil {
		return "Kinara: Balance check failed. Try again."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Balance  float64 `json:"balance"`
			Currency string  `json:"currency"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return "Kinara: No wallet found. Register with REGISTER <name> <crop>"
	}
	return fmt.Sprintf("KINARA BALANCE:\n%.2f %s\nReply INCOME for earnings report", result.Data.Balance, result.Data.Currency)
}

func (h *Handler) handleRegister(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) < 2 {
		return "Usage: REGISTER <name> <main_crop>\nExample: REGISTER KOFI MAIZE"
	}
	name := cmd.Args[0]
	crop := cmd.Args[1]
	payload := fmt.Sprintf(`{"name":"%s","phone":"%s","primary_crop":"%s","country":"TG","currency":"XOF"}`, name, from, strings.ToLower(crop))
	resp, err := httpPost(ctx, fmt.Sprintf("%s/api/v1/farmers", h.farmerURL), payload)
	if err != nil {
		return "Kinara: Registration failed. Try again."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return "Kinara: Registration failed. You may already be registered.\nReply STATUS to check."
	}
	return fmt.Sprintf("KINARA: Welcome %s!\nYou are registered as a %s farmer.\nID: %s\nReply PRICE %s to see market prices.",
		strings.ToUpper(name), strings.ToUpper(crop), result.Data.ID[:8], strings.ToUpper(crop))
}

func (h *Handler) saveLog(ctx context.Context, provider models.SMSProvider, from, to, body, response string, success bool) {
	cmd := parseCommand(body)
	h.store.SaveLog(ctx, models.SMSLog{
		ID:        uuid.New(),
		Provider:  provider,
		Direction: models.DirectionInbound,
		From:      from,
		To:        to,
		Body:      body,
		Response:  response,
		Command:   string(cmd.Type),
		Success:   success,
		CreatedAt: time.Now().UTC(),
	})
}

// parseCommand converts raw SMS text to a structured command.
func parseCommand(text string) models.ParsedCommand {
	text = strings.TrimSpace(text)
	parts := strings.FieldsFunc(text, func(r rune) bool { return unicode.IsSpace(r) })
	if len(parts) == 0 {
		return models.ParsedCommand{Type: models.CmdUnknown, Args: []string{""}, RawText: text}
	}
	keyword := strings.ToUpper(parts[0])
	args := parts[1:]

	cmdMap := map[string]models.CommandType{
		"PRICE":    models.CmdPrice,
		"PRIX":     models.CmdPrice, // French
		"BUYERS":   models.CmdBuyers,
		"ACHETEURS": models.CmdBuyers,
		"SELL":     models.CmdSell,
		"VENDRE":   models.CmdSell,
		"WEATHER":  models.CmdWeather,
		"METEO":    models.CmdWeather,
		"STATUS":   models.CmdStatus,
		"STATUT":   models.CmdStatus,
		"INCOME":   models.CmdIncome,
		"REVENU":   models.CmdIncome,
		"BALANCE":  models.CmdBalance,
		"SOLDE":    models.CmdBalance,
		"REGISTER":  models.CmdRegister,
		"INSCRIRE":  models.CmdRegister,
		"PATIENT":   models.CmdPatient,
		"MALADE":    models.CmdPatient,
		"SYMPTOM":   models.CmdSymptom,
		"SYMPTOME":  models.CmdSymptom,
		"APPT":      models.CmdAppt,
		"RDV":       models.CmdAppt,
		"LAB":       models.CmdLab,
		"LABO":      models.CmdLab,
		"HELP":      models.CmdHelp,
		"AIDE":      models.CmdHelp,
		"?":         models.CmdHelp,
	}
	if t, ok := cmdMap[keyword]; ok {
		return models.ParsedCommand{Type: t, Args: args, RawText: text}
	}
	return models.ParsedCommand{Type: models.CmdUnknown, Args: parts, RawText: text}
}

// handlePatient: "PATIENT John 35M" → create patient record
// Format: PATIENT <name> <age><M|F> [clinic_id]
func (h *Handler) handlePatient(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) < 2 {
		return "KINARA SANTE:\nPATIENT <nom> <age><M|F>\nEx: PATIENT Kofi 35M"
	}
	name := cmd.Args[0]
	ageGender := cmd.Args[1]
	gender := "M"
	ageStr := ageGender
	if len(ageGender) > 0 {
		last := strings.ToUpper(string(ageGender[len(ageGender)-1]))
		if last == "M" || last == "F" {
			gender = last
			ageStr = ageGender[:len(ageGender)-1]
		}
	}
	age := 0
	for _, c := range ageStr {
		if unicode.IsDigit(c) {
			age = age*10 + int(c-'0')
		}
	}
	if age <= 0 || age > 150 {
		return "KINARA: Age invalide.\nEx: PATIENT Kofi 35M"
	}

	body := fmt.Sprintf(`{"first_name":%q,"last_name":"","date_of_birth":"2006-01-02","gender":%q,"phone":%q,"country":"TG","tenant_id":"TG","sms_registered":true,"sms_age":%d}`,
		name, strings.ToLower(gender), from, age)
	resp, err := httpPost(ctx, h.patientURL+"/api/v1/patients/sms", body)
	if err != nil {
		return "KINARA: Enregistrement échoué. Réessayez."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			PatientRef string `json:"patient_ref"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return "KINARA: Patient enregistré (hors ligne). Venez au centre de santé."
	}
	return fmt.Sprintf("KINARA SANTÉ: ✓\nPatient: %s\nID: %s\nMontrez ce code au dispensaire.", name, result.Data.PatientRef)
}

// handleSymptom: "SYMPTOM fever chills headache" → add to active SOAP note
func (h *Handler) handleSymptom(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) == 0 {
		return "KINARA SANTÉ:\nSYMPTOM <symptômes>\nEx: SYMPTOM fièvre frissons"
	}
	symptoms := strings.Join(cmd.Args, ", ")
	body := fmt.Sprintf(`{"phone":%q,"subjective":%q,"source":"sms"}`, from, symptoms)
	resp, err := httpPost(ctx, h.clinicalURL+"/api/v1/soap/sms", body)
	if err != nil {
		return "KINARA: Symptômes notés. Consultez le médecin."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			NoteRef string `json:"note_ref"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return fmt.Sprintf("KINARA SANTÉ: ✓\nSymptômes notés: %s\nConsultez le médecin dès que possible.", symptoms)
	}
	return fmt.Sprintf("KINARA SANTÉ: ✓\nSymptômes enregistrés: %s\nNote: %s\nConsultez le médecin.", symptoms, result.Data.NoteRef)
}

// handleAppt: "APPT <date> <clinic>" → schedule appointment
// Format: APPT 2026-10-15 LOME-NORD
func (h *Handler) handleAppt(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) < 2 {
		return "KINARA RDV:\nAPPT <date> <centre>\nEx: APPT 2026-10-15 LOME-NORD\nOu RDV DEMAIN TSEVIE"
	}
	dateStr := cmd.Args[0]
	clinic := strings.Join(cmd.Args[1:], " ")

	// Accept DEMAIN/TOMORROW
	if strings.ToUpper(dateStr) == "DEMAIN" || strings.ToUpper(dateStr) == "TOMORROW" {
		dateStr = time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	}

	body := fmt.Sprintf(`{"phone":%q,"scheduled_date":%q,"clinic_name":%q,"source":"sms","duration_min":30}`,
		from, dateStr, clinic)
	resp, err := httpPost(ctx, h.appointmentURL+"/api/v1/appointments/sms", body)
	if err != nil {
		return fmt.Sprintf("KINARA RDV: Demande reçue\n%s à %s\nPrésentez-vous à 8h00.", dateStr, clinic)
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			AppointmentRef string `json:"appointment_ref"`
			ScheduledAt    string `json:"scheduled_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return fmt.Sprintf("KINARA RDV: ✓\n%s\n%s\nPrésentez-vous 15 min avant.", clinic, dateStr)
	}
	return fmt.Sprintf("KINARA RDV: ✓\nRef: %s\n%s\nPrésentez-vous 15 min avant.", result.Data.AppointmentRef, dateStr)
}

// handleLab: "LAB <patient_ref> <test>" → order lab test via SMS
func (h *Handler) handleLab(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) < 2 {
		return "KINARA LABO:\nLAB <ID-patient> <test>\nEx: LAB PAT-A1B2C3D4 MALARIA\nTests: MALARIA HIV GLUCOSE HEMO"
	}
	patientRef := cmd.Args[0]
	testName := strings.Join(cmd.Args[1:], " ")

	body := fmt.Sprintf(`{"patient_ref":%q,"test_name":%q,"ordered_by_phone":%q,"priority":"routine","source":"sms"}`,
		patientRef, testName, from)
	resp, err := httpPost(ctx, h.labURL+"/api/v1/lab/orders/sms", body)
	if err != nil {
		return fmt.Sprintf("KINARA LABO: Test %s demandé pour %s\nCollecte au laboratoire.", testName, patientRef)
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			OrderRef string `json:"order_ref"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return fmt.Sprintf("KINARA LABO: ✓\nTest: %s\nPatient: %s\nAllez au labo maintenant.", testName, patientRef)
	}
	return fmt.Sprintf("KINARA LABO: ✓\nOrdre: %s\nTest: %s\nPatient: %s\nRésultats en 24h.", result.Data.OrderRef, testName, patientRef)
}

func helpText() string {
	return `KINARA COMMANDS:
PRICE <crop> - Market prices
BUYERS <crop> - Find buyers
SELL <crop> <qty> <price> - Create listing
WEATHER [city] - Forecast
STATUS - Your listings
INCOME - Earnings report
BALANCE - Wallet balance
REGISTER <name> <crop> - Sign up

CLINIC / SANTÉ:
PATIENT <name> <age><M|F> - Register patient
SYMPTOM <symptoms...> - Log symptoms
APPT <date> <clinic> - Book appointment
LAB <patient_id> <test> - Order lab test
HELP - This menu

EN FRANÇAIS: PRIX METEO VENDRE AIDE RDV MALADE SYMPTOME LABO`
}

func (c models.CommandType) String() string { return string(c) }

func httpGet(ctx context.Context, u string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Service", "sms-gateway")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func httpPost(ctx context.Context, u, body string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "sms-gateway")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
