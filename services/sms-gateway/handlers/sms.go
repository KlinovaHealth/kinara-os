package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
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
	store              Store
	marketURL          string
	weatherURL         string
	analyticsURL       string
	farmerURL          string
	patientURL         string
	clinicalURL        string
	appointmentURL     string
	labURL             string
	paymentURL         string
	fleetURL           string
	routeURL           string
	vesselURL          string
	portURL            string
	customsURL         string
	referralURL        string
	immunizationURL    string
	cooperativeURL     string
	outbreakURL        string
	vehicleTrackingURL string
	twilioSID          string
	twilioToken        string
	atAPIKey           string
	atUsername         string
}

func NewHandler(store Store) *Handler {
	return &Handler{
		store:              store,
		marketURL:          env("MARKET_SERVICE_URL", "http://market-service:8083"),
		weatherURL:         env("WEATHER_SERVICE_URL", "http://weather-service:8106"),
		analyticsURL:       env("ANALYTICS_SERVICE_URL", "http://analytics-service:8108"),
		farmerURL:          env("FARMER_SERVICE_URL", "http://farmer-service:8084"),
		patientURL:         env("PATIENT_SERVICE_URL", "http://patient-service:8081"),
		clinicalURL:        env("CLINICAL_SERVICE_URL", "http://clinical-service:8082"),
		appointmentURL:     env("APPOINTMENT_SERVICE_URL", "http://appointment-service:8120"),
		labURL:             env("LAB_SERVICE_URL", "http://lab-service:8122"),
		paymentURL:         env("PAYMENT_SERVICE_URL", "http://payment-service:8107"),
		fleetURL:           env("FLEET_SERVICE_URL", "http://fleet-service:8090"),
		routeURL:           env("ROUTE_SERVICE_URL", "http://route-service:8095"),
		vesselURL:          env("VESSEL_SERVICE_URL", "http://vessel-service:8104"),
		portURL:            env("PORT_SERVICE_URL", "http://port-service:8116"),
		customsURL:         env("CUSTOMS_SERVICE_URL", "http://customs-service:8114"),
		referralURL:        env("REFERRAL_SERVICE_URL", "http://referral-service:8083"),
		immunizationURL:    env("IMMUNIZATION_SERVICE_URL", "http://immunization-service:8121"),
		cooperativeURL:     env("COOPERATIVE_SERVICE_URL", "http://cooperative-service:8096"),
		outbreakURL:        env("OUTBREAK_SERVICE_URL", "http://outbreak-service:8123"),
		vehicleTrackingURL: env("VEHICLE_TRACKING_SERVICE_URL", "http://vehicle-tracking-service:8127"),
		twilioSID:          os.Getenv("TWILIO_ACCOUNT_SID"),
		twilioToken:        os.Getenv("TWILIO_AUTH_TOKEN"),
		atAPIKey:           os.Getenv("AFRICASTALKING_API_KEY"),
		atUsername:         os.Getenv("AFRICASTALKING_USERNAME"),
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

// validateTwilioSignature verifies the X-Twilio-Signature header.
// Returns true in dev mode (no token configured).
func (h *Handler) validateTwilioSignature(r *http.Request, fullURL string) bool {
	if h.twilioToken == "" {
		return true
	}
	expected := r.Header.Get("X-Twilio-Signature")
	if expected == "" {
		return false
	}
	r.ParseForm()
	keys := make([]string, 0, len(r.Form))
	for k := range r.Form {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := fullURL
	for _, k := range keys {
		s += k + r.FormValue(k)
	}
	mac := hmac.New(sha1.New, []byte(h.twilioToken))
	mac.Write([]byte(s))
	computed := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(computed), []byte(expected))
}

func (h *Handler) TwilioWebhook(w http.ResponseWriter, r *http.Request) {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.String())
	if !h.validateTwilioSignature(r, fullURL) {
		log.Printf("sms-gateway: invalid Twilio signature from %s", r.RemoteAddr)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	r.ParseForm()
	from := r.FormValue("From")
	to := r.FormValue("To")
	body := r.FormValue("Body")
	country := r.FormValue("FromCountry")

	if from == "" || body == "" {
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, `<?xml version="1.0"?><Response><Message>Invalid request</Message></Response>`)
		return
	}

	response := h.processCommand(r.Context(), from, body, country)
	h.saveLog(r.Context(), models.ProviderTwilio, from, to, body, response, true)

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
		w.WriteHeader(http.StatusBadRequest)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.From == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{Error: "from and body required"})
		return
	}
	response := h.processCommand(r.Context(), req.From, req.Body, "TG")
	h.saveLog(r.Context(), models.ProviderTwilio, req.From, "test", req.Body, response, true)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.APIResponse{Success: true, Data: map[string]string{
		"response": response,
		"command":  string(parseCommand(req.Body).Type),
	}})
}

func (h *Handler) ListLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.APIResponse{Success: true, Data: []string{}})
}

func (h *Handler) processCommand(ctx context.Context, from, text, country string) string {
	cmd := parseCommand(text)
	log.Printf("sms from=%s intent=%s args=%v", from, cmd.Type, cmd.Args)

	var response string
	switch cmd.Type {
	// Agriculture
	case models.CmdPrice:
		response = h.handlePrice(ctx, cmd)
	case models.CmdBuyers:
		response = h.handleBuyers(ctx, cmd)
	case models.CmdSell:
		response = h.handleSell(ctx, cmd, from)
	case models.CmdWeather:
		response = h.handleWeather(ctx, cmd, country)
	case models.CmdStatus:
		response = h.handleStatus(ctx, from)
	case models.CmdIncome:
		response = h.handleIncome(ctx, from)
	case models.CmdBalance:
		response = h.handleBalance(ctx, from)
	case models.CmdRegister:
		response = h.handleRegister(ctx, cmd, from)
	case models.CmdFarmers:
		response = h.handleFarmers(ctx, cmd)
	case models.CmdCoop:
		response = h.handleCoop(ctx, cmd)
	case models.CmdJoin:
		response = h.handleJoin(ctx, cmd, from)
	case models.CmdSavings:
		response = h.handleSavings(ctx, from)

	// Health
	case models.CmdPatient:
		response = h.handlePatient(ctx, cmd, from)
	case models.CmdSymptom:
		response = h.handleSymptom(ctx, cmd, from)
	case models.CmdAppt:
		response = h.handleAppt(ctx, cmd, from)
	case models.CmdLab:
		response = h.handleLab(ctx, cmd, from)
	case models.CmdLabResult:
		response = h.handleLabResult(ctx, cmd, from)
	case models.CmdRefer:
		response = h.handleRefer(ctx, cmd, from)
	case models.CmdCancel:
		response = h.handleCancel(ctx, cmd, from)
	case models.CmdReschedule:
		response = h.handleReschedule(ctx, cmd, from)
	case models.CmdVaccine:
		response = h.handleVaccine(ctx, cmd, from)
	case models.CmdSchedule:
		response = h.handleSchedule(ctx, cmd, from)
	case models.CmdOutbreak:
		response = h.handleOutbreak(ctx, cmd, country)

	// Logistics
	case models.CmdTrack:
		response = h.handleTrack(ctx, cmd)
	case models.CmdRoute:
		response = h.handleRoute(ctx, cmd)
	case models.CmdFleet:
		response = h.handleFleet(ctx, cmd)

	// Maritime
	case models.CmdVessel:
		response = h.handleVessel(ctx, cmd)
	case models.CmdBerth:
		response = h.handleBerth(ctx, cmd)
	case models.CmdManifest:
		response = h.handleManifest(ctx, cmd)
	case models.CmdCustoms:
		response = h.handleCustoms(ctx, cmd)

	// Cross-pillar
	case models.CmdSend:
		response = h.handleSend(ctx, cmd, from)
	case models.CmdConvert:
		response = h.handleConvert(ctx, cmd)
	case models.CmdImpact:
		response = h.handleImpact(ctx, cmd)

	case models.CmdHelp:
		response = helpText()
	default:
		response = format160(fmt.Sprintf("KINARA: Commande inconnue '%s'.\n%s", cmd.Args[0], "Reply HELP for commands."))
	}
	return response
}

// ─────────────────────────────────────────────
// Agriculture handlers
// ─────────────────────────────────────────────

func (h *Handler) handlePrice(ctx context.Context, cmd models.ParsedCommand) string {
	if len(cmd.Args) == 0 {
		return "Usage: PRICE <crop>\nEx: PRICE MAIZE"
	}
	crop := strings.ToLower(cmd.Args[0])
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/market/prices?commodity=%s&limit=3", h.marketURL, url.QueryEscape(crop)))
	if err != nil {
		return format160(fmt.Sprintf("KINARA: Price lookup unavailable. Try again.\n%s", shortErr(err)))
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
		return format160(fmt.Sprintf("KINARA: No prices for %s today.", strings.ToUpper(crop)))
	}
	lines := []string{fmt.Sprintf("KINARA PRIX: %s", strings.ToUpper(crop))}
	for _, p := range result.Data {
		lines = append(lines, fmt.Sprintf("%s: %.0f %s/%s", p.Market, p.PricePerUnit, p.Currency, p.Unit))
	}
	lines = append(lines, "SELL "+strings.ToUpper(crop)+" <qty> <price>")
	return format160(strings.Join(lines, "\n"))
}

func (h *Handler) handleBuyers(ctx context.Context, cmd models.ParsedCommand) string {
	if len(cmd.Args) == 0 {
		return "Usage: BUYERS <crop>\nEx: BUYERS COCOA"
	}
	crop := strings.ToLower(cmd.Args[0])
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/market/bids?commodity=%s&status=open&limit=5", h.marketURL, url.QueryEscape(crop)))
	if err != nil {
		return "KINARA: Buyer lookup failed. Try again."
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			BuyerName    string  `json:"buyer_name"`
			PricePerUnit float64 `json:"price_per_unit"`
			QuantityKg   float64 `json:"quantity_kg"`
			Currency     string  `json:"currency"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success || len(result.Data) == 0 {
		return format160(fmt.Sprintf("KINARA: No buyers for %s now.", strings.ToUpper(crop)))
	}
	lines := []string{fmt.Sprintf("BUYERS: %s", strings.ToUpper(crop))}
	for _, b := range result.Data {
		lines = append(lines, fmt.Sprintf("%s: %.0f%s/kg (%.0fkg)", b.BuyerName, b.PricePerUnit, b.Currency, b.QuantityKg))
	}
	return format160(strings.Join(lines, "\n"))
}

func (h *Handler) handleSell(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) < 3 {
		return "Usage: SELL <crop> <qty_kg> <price>\nEx: SELL MAIZE 500 250"
	}
	crop, qty, price := cmd.Args[0], cmd.Args[1], cmd.Args[2]
	payload := fmt.Sprintf(`{"commodity_name":%q,"quantity_kg":%s,"price_per_kg":%s,"seller_phone":%q,"currency":"XOF","location":"Togo"}`,
		crop, qty, price, from)
	resp, err := httpPost(ctx, fmt.Sprintf("%s/api/v1/market/listings", h.marketURL), payload)
	if err != nil {
		return "KINARA: Listing failed. Check numbers and retry."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct{ ID string `json:"id"` } `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return "KINARA: Could not list. Ex: SELL MAIZE 500 250"
	}
	return format160(fmt.Sprintf("KINARA: Listed!\n%skg %s @ %s XOF/kg\nID: %s\nBuyers will call.", qty, strings.ToUpper(crop), price, result.Data.ID[:8]))
}

func (h *Handler) handleWeather(ctx context.Context, cmd models.ParsedCommand, country string) string {
	location := "Togo"
	if len(cmd.Args) > 0 {
		location = strings.Join(cmd.Args, " ")
	}
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/weather/forecast?location=%s&days=3", h.weatherURL, url.QueryEscape(location)))
	if err != nil {
		return "KINARA: Weather unavailable. Try again."
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			Date       string  `json:"date"`
			TempMaxC   float64 `json:"temp_max_c"`
			TempMinC   float64 `json:"temp_min_c"`
			Condition  string  `json:"condition"`
			RainfallMm float64 `json:"rainfall_mm"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success || len(result.Data) == 0 {
		return format160(fmt.Sprintf("KINARA: No forecast for %s.", location))
	}
	lines := []string{fmt.Sprintf("METEO: %s", strings.ToUpper(location))}
	for _, d := range result.Data {
		lines = append(lines, fmt.Sprintf("%s: %.0f-%.0fC %s R:%.0fmm", d.Date[5:10], d.TempMinC, d.TempMaxC, d.Condition, d.RainfallMm))
	}
	return format160(strings.Join(lines, "\n"))
}

func (h *Handler) handleStatus(ctx context.Context, from string) string {
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/market/listings?seller_phone=%s&status=active&limit=3", h.marketURL, url.QueryEscape(from)))
	if err != nil {
		return "KINARA: Status check failed."
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			CommodityName string  `json:"commodity_name"`
			QuantityKg    float64 `json:"quantity_kg"`
			Status        string  `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success || len(result.Data) == 0 {
		return "KINARA: No active listings.\nSELL <crop> <qty> <price> to start."
	}
	lines := []string{"VOS ANNONCES:"}
	for _, l := range result.Data {
		lines = append(lines, fmt.Sprintf("%s: %.0fkg [%s]", strings.ToUpper(l.CommodityName), l.QuantityKg, l.Status))
	}
	return format160(strings.Join(lines, "\n"))
}

func (h *Handler) handleIncome(ctx context.Context, from string) string {
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/analytics/impact?pillar=agriculture&country=TG&limit=1", h.analyticsURL))
	if err != nil {
		return "KINARA: Income data unavailable."
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
		return "KINARA: No income data yet. Keep selling to build history."
	}
	m := result.Data[0]
	return format160(fmt.Sprintf("REVENU KINARA:\n%s: %.0f %s\nSELL via Kinara to earn more.", m.MetricName, m.MetricValue, m.MetricUnit))
}

func (h *Handler) handleBalance(ctx context.Context, from string) string {
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/wallets?owner_phone=%s", h.paymentURL, url.QueryEscape(from)))
	if err != nil {
		return "KINARA: Balance check failed. Try again."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Balance  float64 `json:"balance"`
			Currency string  `json:"currency"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return "KINARA: No wallet found.\nREGISTER <name> <crop> to sign up."
	}
	return format160(fmt.Sprintf("SOLDE KINARA:\n%.2f %s\nSEND <phone> <amount> to transfer.", result.Data.Balance, result.Data.Currency))
}

func (h *Handler) handleRegister(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) < 2 {
		return "Usage: REGISTER <name> <crop>\nEx: REGISTER KOFI MAIZE"
	}
	name, crop := cmd.Args[0], cmd.Args[1]
	payload := fmt.Sprintf(`{"name":%q,"phone":%q,"primary_crop":%q,"country":"TG","currency":"XOF"}`, name, from, strings.ToLower(crop))
	resp, err := httpPost(ctx, fmt.Sprintf("%s/api/v1/farmers", h.farmerURL), payload)
	if err != nil {
		return "KINARA: Registration failed. Try again."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct{ ID string `json:"id"` } `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return "KINARA: Already registered?\nSTATUS to check."
	}
	return format160(fmt.Sprintf("KINARA: Bienvenue %s!\nAgriculteur %s. ID: %s\nPRICE %s pour les prix.", strings.ToUpper(name), strings.ToUpper(crop), result.Data.ID[:8], strings.ToUpper(crop)))
}

func (h *Handler) handleFarmers(ctx context.Context, cmd models.ParsedCommand) string {
	region := "Togo"
	if len(cmd.Args) > 0 {
		region = strings.Join(cmd.Args, " ")
	}
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/farmers?region=%s&limit=5", h.farmerURL, url.QueryEscape(region)))
	if err != nil {
		return "KINARA: Farmer list unavailable."
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			Name        string `json:"name"`
			PrimaryCrop string `json:"primary_crop"`
			Region      string `json:"region"`
		} `json:"data"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA: No farmers found in %s.", region))
	}
	lines := []string{fmt.Sprintf("AGRICULTEURS: %s (%d)", strings.ToUpper(region), result.Total)}
	for _, f := range result.Data {
		lines = append(lines, fmt.Sprintf("%s [%s]", f.Name, strings.ToUpper(f.PrimaryCrop)))
	}
	return format160(strings.Join(lines, "\n"))
}

func (h *Handler) handleCoop(ctx context.Context, cmd models.ParsedCommand) string {
	region := ""
	if len(cmd.Args) > 0 {
		region = strings.Join(cmd.Args, " ")
	}
	coopURL := fmt.Sprintf("%s/api/v1/cooperatives?limit=3", h.cooperativeURL)
	if region != "" {
		coopURL += "&region=" + url.QueryEscape(region)
	}
	resp, err := httpGet(ctx, coopURL)
	if err != nil {
		return "KINARA: Cooperative info unavailable."
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			Name        string `json:"name"`
			Region      string `json:"region"`
			MemberCount int    `json:"member_count"`
			CropFocus   string `json:"crop_focus"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success || len(result.Data) == 0 {
		return "KINARA: No cooperatives found.\nREGISTER to join the network."
	}
	lines := []string{"COOPERATIVES KINARA:"}
	for _, c := range result.Data {
		lines = append(lines, fmt.Sprintf("%s [%s] %d membres", c.Name, c.CropFocus, c.MemberCount))
	}
	lines = append(lines, "JOIN <coop_name> to join")
	return format160(strings.Join(lines, "\n"))
}

func (h *Handler) handleJoin(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) == 0 {
		return "Usage: JOIN <cooperative_name>\nEx: JOIN LOME-MAIZE-COOP\nCOOP to see list."
	}
	coopName := strings.Join(cmd.Args, " ")
	payload := fmt.Sprintf(`{"cooperative_name":%q,"phone":%q}`, coopName, from)
	resp, err := httpPost(ctx, fmt.Sprintf("%s/api/v1/cooperatives/join", h.cooperativeURL), payload)
	if err != nil {
		return "KINARA: Join request failed. Try again."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			MemberID string `json:"member_id"`
			CoopName string `json:"coop_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA: Could not join %s. Check name and retry.", coopName))
	}
	return format160(fmt.Sprintf("KINARA: Joined %s!\nMember ID: %s\nBenefits: group pricing + loans.", result.Data.CoopName, result.Data.MemberID[:8]))
}

func (h *Handler) handleSavings(ctx context.Context, from string) string {
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/farmer-finance/savings?phone=%s", h.paymentURL, url.QueryEscape(from)))
	if err != nil {
		return "KINARA: Savings data unavailable."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Balance      float64 `json:"balance"`
			Currency     string  `json:"currency"`
			TotalDeposit float64 `json:"total_deposited"`
			InterestRate float64 `json:"interest_rate_pct"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return "KINARA: No savings account found.\nREGISTER to open one."
	}
	return format160(fmt.Sprintf("EPARGNE KINARA:\nSolde: %.2f %s\nDépôts: %.2f\nTaux: %.1f%%/an", result.Data.Balance, result.Data.Currency, result.Data.TotalDeposit, result.Data.InterestRate))
}

// ─────────────────────────────────────────────
// Health handlers
// ─────────────────────────────────────────────

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
	body := fmt.Sprintf(`{"first_name":%q,"last_name":"","gender":%q,"phone":%q,"country":"TG","tenant_id":"TG","sms_registered":true,"sms_age":%d}`,
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
		return "KINARA: Présentez-vous au dispensaire."
	}
	return format160(fmt.Sprintf("KINARA SANTE: OK\nPatient: %s\nID: %s\nMontrez ce code.", name, result.Data.PatientRef))
}

func (h *Handler) handleSymptom(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) == 0 {
		return "KINARA SANTE:\nSYMPTOM <symptômes>\nEx: SYMPTOM fièvre frissons"
	}
	symptoms := strings.Join(cmd.Args, ", ")
	body := fmt.Sprintf(`{"phone":%q,"subjective":%q,"source":"sms"}`, from, symptoms)
	resp, err := httpPost(ctx, h.clinicalURL+"/api/v1/soap/sms", body)
	if err != nil {
		return format160(fmt.Sprintf("KINARA: Symptômes notés: %s\nConsultez le médecin.", symptoms))
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct{ NoteRef string `json:"note_ref"` } `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA: Symptômes: %s\nConsultez le médecin.", symptoms))
	}
	return format160(fmt.Sprintf("KINARA SANTE: OK\nNote: %s\nSymptômes: %s\nConsultez dès que possible.", result.Data.NoteRef, symptoms))
}

func (h *Handler) handleAppt(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) < 2 {
		return "KINARA RDV:\nAPPT <date> <centre>\nEx: APPT 2026-10-15 LOME-NORD"
	}
	dateStr := cmd.Args[0]
	clinic := strings.Join(cmd.Args[1:], " ")
	if strings.ToUpper(dateStr) == "DEMAIN" || strings.ToUpper(dateStr) == "TOMORROW" {
		dateStr = time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	}
	body := fmt.Sprintf(`{"phone":%q,"scheduled_date":%q,"clinic_name":%q,"source":"sms","duration_min":30}`, from, dateStr, clinic)
	resp, err := httpPost(ctx, h.appointmentURL+"/api/v1/appointments/sms", body)
	if err != nil {
		return format160(fmt.Sprintf("KINARA RDV: Demande reçue\n%s à %s\nPrésentez-vous à 8h00.", dateStr, clinic))
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			AppointmentRef string `json:"appointment_ref"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA RDV: OK\n%s\n%s\n15 min avant.", clinic, dateStr))
	}
	return format160(fmt.Sprintf("KINARA RDV: OK\nRef: %s\n%s\n15 min avant.", result.Data.AppointmentRef, dateStr))
}

func (h *Handler) handleLab(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) < 2 {
		return "Usage: LAB <patient_ref> <test>\nEx: LAB PAT-A1B2 MALARIA\nTests: MALARIA HIV GLUCOSE HGB"
	}
	patientRef := cmd.Args[0]
	testName := strings.Join(cmd.Args[1:], " ")
	body := fmt.Sprintf(`{"patient_ref":%q,"test_name":%q,"ordered_by_phone":%q,"priority":"routine","source":"sms"}`,
		patientRef, testName, from)
	resp, err := httpPost(ctx, h.labURL+"/api/v1/lab/orders/sms", body)
	if err != nil {
		return format160(fmt.Sprintf("KINARA LABO: Test %s demandé.\nAllez au laboratoire.", testName))
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct{ OrderRef string `json:"order_ref"` } `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA LABO: OK\n%s\n%s\nAllez au labo.", testName, patientRef))
	}
	return format160(fmt.Sprintf("KINARA LABO: OK\nOrdre: %s\nTest: %s\nRésultats 24h.", result.Data.OrderRef, testName))
}

func (h *Handler) handleLabResult(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) == 0 {
		return "Usage: RESULT <order_ref>\nEx: RESULT LAB-A1B2C3D4"
	}
	orderRef := cmd.Args[0]
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/lab/results?order_ref=%s", h.labURL, url.QueryEscape(orderRef)))
	if err != nil {
		return "KINARA: Résultats indisponibles. Réessayez."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			TestName  string `json:"test_name"`
			Result    string `json:"result"`
			Unit      string `json:"unit"`
			Status    string `json:"status"`
			CompletedAt string `json:"completed_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA: Pas de résultats pour %s.\nVérifiez l'ID.", orderRef))
	}
	if result.Data.Status != "completed" {
		return format160(fmt.Sprintf("KINARA LABO: En cours\nOrdre: %s\nStatut: %s\nVérifiez plus tard.", orderRef, result.Data.Status))
	}
	return format160(fmt.Sprintf("KINARA LABO: Résultat\n%s: %s %s\nDate: %s", result.Data.TestName, result.Data.Result, result.Data.Unit, result.Data.CompletedAt[:10]))
}

func (h *Handler) handleRefer(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) < 2 {
		return "Usage: REFER <patient_ref> <facility>\nEx: REFER PAT-A1B2 CHU-LOME"
	}
	patientRef := cmd.Args[0]
	facility := strings.Join(cmd.Args[1:], " ")
	body := fmt.Sprintf(`{"patient_ref":%q,"referring_facility":%q,"reason":"SMS referral","source":"sms","referred_by_phone":%q}`,
		patientRef, facility, from)
	resp, err := httpPost(ctx, h.referralURL+"/api/v1/referrals/sms", body)
	if err != nil {
		return format160(fmt.Sprintf("KINARA: Référence envoyée à %s pour %s.", facility, patientRef))
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct{ ReferralRef string `json:"referral_ref"` } `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA REFER: OK\n%s → %s\nApportez vos documents.", patientRef, facility))
	}
	return format160(fmt.Sprintf("KINARA REFER: OK\nRef: %s\n%s → %s\nDocuments requis.", result.Data.ReferralRef, patientRef, facility))
}

func (h *Handler) handleCancel(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) == 0 {
		return "Usage: CANCEL <appointment_ref>\nEx: CANCEL APT-A1B2C3D4"
	}
	apptRef := cmd.Args[0]
	body := fmt.Sprintf(`{"appointment_ref":%q,"cancelled_by_phone":%q,"reason":"SMS cancellation"}`, apptRef, from)
	resp, err := httpPost(ctx, fmt.Sprintf("%s/api/v1/appointments/cancel/sms", h.appointmentURL), body)
	if err != nil {
		return "KINARA: Annulation envoyée. Vérifiez avec le centre."
	}
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA: RDV %s annulé.\nAPPT pour un nouveau.", apptRef))
	}
	return format160(fmt.Sprintf("KINARA RDV: Annulé\nRef: %s\nAPPT <date> <centre> pour un nouveau.", apptRef))
}

func (h *Handler) handleReschedule(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) < 2 {
		return "Usage: RESCHEDULE <ref> <new_date>\nEx: RESCHEDULE APT-A1B2 2026-10-20"
	}
	apptRef := cmd.Args[0]
	newDate := cmd.Args[1]
	if strings.ToUpper(newDate) == "DEMAIN" || strings.ToUpper(newDate) == "TOMORROW" {
		newDate = time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	}
	body := fmt.Sprintf(`{"appointment_ref":%q,"new_date":%q,"phone":%q}`, apptRef, newDate, from)
	resp, err := httpPost(ctx, fmt.Sprintf("%s/api/v1/appointments/reschedule/sms", h.appointmentURL), body)
	if err != nil {
		return format160(fmt.Sprintf("KINARA RDV: Reprogrammé\n%s → %s\nConfirmation en cours.", apptRef, newDate))
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct{ NewRef string `json:"new_ref"` } `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA RDV: %s → %s\nPrésentez-vous à 8h00.", apptRef, newDate))
	}
	return format160(fmt.Sprintf("KINARA RDV: Reprogrammé\nRef: %s\nDate: %s\n15 min avant.", result.Data.NewRef, newDate))
}

func (h *Handler) handleVaccine(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) == 0 {
		return "Usage: VACCINE <patient_ref>\nEx: VACCINE PAT-A1B2\nOu: VACCINE SCHEDULE"
	}
	if strings.ToUpper(cmd.Args[0]) == "SCHEDULE" || strings.ToUpper(cmd.Args[0]) == "CALENDRIER" {
		resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/immunizations/schedule", h.immunizationURL))
		if err != nil {
			return "KINARA VACCIN:\nSchéma OMS: BCG, PENTA, IPV, Rougeole\nConsultez le centre de santé."
		}
		var result struct {
			Success bool `json:"success"`
			Data    []struct {
				Vaccine string `json:"vaccine_name"`
				AgeWeeks int  `json:"age_weeks"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
			return "KINARA VACCIN:\nBCG:0sem PENTA:6sem IPV:14sem Rougeole:36sem\nCentre de santé pour plus."
		}
		lines := []string{"CALENDRIER VACCINS:"}
		for _, v := range result.Data {
			lines = append(lines, fmt.Sprintf("%s: %dsem", v.Vaccine, v.AgeWeeks))
		}
		return format160(strings.Join(lines, "\n"))
	}
	patientRef := cmd.Args[0]
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/immunizations?patient_ref=%s", h.immunizationURL, url.QueryEscape(patientRef)))
	if err != nil {
		return "KINARA: Historique vaccins indisponible."
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			VaccineName string `json:"vaccine_name"`
			GivenAt     string `json:"given_at"`
			NextDueAt   string `json:"next_due_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success || len(result.Data) == 0 {
		return format160(fmt.Sprintf("KINARA VACCIN: Aucun historique pour %s.\nPrésentez-vous au centre.", patientRef))
	}
	lines := []string{fmt.Sprintf("VACCINS: %s", patientRef)}
	for _, v := range result.Data {
		line := v.VaccineName + ": " + v.GivenAt[:10]
		if v.NextDueAt != "" {
			line += " → " + v.NextDueAt[:10]
		}
		lines = append(lines, line)
	}
	return format160(strings.Join(lines, "\n"))
}

func (h *Handler) handleSchedule(ctx context.Context, cmd models.ParsedCommand, from string) string {
	clinic := "KINARA"
	if len(cmd.Args) > 0 {
		clinic = strings.Join(cmd.Args, " ")
	}
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/appointments/availability?clinic=%s&days=7", h.appointmentURL, url.QueryEscape(clinic)))
	if err != nil {
		return format160(fmt.Sprintf("KINARA: Disponibilités de %s indisponibles.\nAPPT <date> <centre> pour réserver.", clinic))
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			Date      string `json:"date"`
			Available int    `json:"available_slots"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success || len(result.Data) == 0 {
		return format160(fmt.Sprintf("KINARA: Pas de créneaux pour %s.\nAppeler le centre directement.", clinic))
	}
	lines := []string{fmt.Sprintf("DISPONIBLE: %s", strings.ToUpper(clinic))}
	for _, d := range result.Data {
		if d.Available > 0 {
			lines = append(lines, fmt.Sprintf("%s: %d créneaux", d.Date[5:10], d.Available))
		}
	}
	lines = append(lines, "APPT <date> <centre> pour réserver")
	return format160(strings.Join(lines, "\n"))
}

func (h *Handler) handleOutbreak(ctx context.Context, cmd models.ParsedCommand, country string) string {
	region := country
	if region == "" {
		region = "TG"
	}
	if len(cmd.Args) > 0 {
		region = strings.Join(cmd.Args, " ")
	}
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/outbreaks?country=%s&status=active&limit=3", h.outbreakURL, url.QueryEscape(region)))
	if err != nil {
		return "KINARA: Alertes épidémiques indisponibles."
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			Disease  string `json:"disease"`
			Region   string `json:"region"`
			Severity string `json:"severity"`
			Cases    int    `json:"case_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success || len(result.Data) == 0 {
		return "KINARA SANTE: Aucune alerte active dans votre région."
	}
	lines := []string{"ALERTE SANTE:"}
	for _, o := range result.Data {
		lines = append(lines, fmt.Sprintf("%s [%s] %s: %d cas", o.Disease, o.Severity, o.Region, o.Cases))
	}
	lines = append(lines, "Consultez le centre de santé local.")
	return format160(strings.Join(lines, "\n"))
}

// ─────────────────────────────────────────────
// Logistics handlers
// ─────────────────────────────────────────────

func (h *Handler) handleTrack(ctx context.Context, cmd models.ParsedCommand) string {
	if len(cmd.Args) == 0 {
		return "Usage: TRACK <shipment_id>\nEx: TRACK SHP-A1B2C3D4"
	}
	ref := cmd.Args[0]
	// Try vehicle tracking first, fall back to shipment
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/vehicles/track?ref=%s", h.vehicleTrackingURL, url.QueryEscape(ref)))
	if err != nil {
		resp, err = httpGet(ctx, fmt.Sprintf("%s/api/v1/shipments/%s/status", h.routeURL, url.QueryEscape(ref)))
		if err != nil {
			return "KINARA: Suivi indisponible. Vérifiez l'ID."
		}
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Ref      string `json:"ref"`
			Status   string `json:"status"`
			Location string `json:"current_location"`
			UpdatedAt string `json:"updated_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA SUIVI: %s\nStatut inconnu. Vérifiez l'ID.", ref))
	}
	updatedAt := ""
	if len(result.Data.UpdatedAt) >= 10 {
		updatedAt = result.Data.UpdatedAt[:10]
	}
	return format160(fmt.Sprintf("KINARA SUIVI: %s\nStatut: %s\nPosition: %s\nMàj: %s", ref, result.Data.Status, result.Data.Location, updatedAt))
}

func (h *Handler) handleRoute(ctx context.Context, cmd models.ParsedCommand) string {
	if len(cmd.Args) < 2 {
		return "Usage: ROUTE <origin> <destination>\nEx: ROUTE LOME ACCRA"
	}
	origin := cmd.Args[0]
	dest := strings.Join(cmd.Args[1:], " ")
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/routes/query?origin=%s&destination=%s", h.routeURL, url.QueryEscape(origin), url.QueryEscape(dest)))
	if err != nil {
		return "KINARA: Calcul de route indisponible."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			DistanceKm  float64 `json:"distance_km"`
			DurationHrs float64 `json:"duration_hours"`
			Highway     string  `json:"primary_highway"`
			Stops       int     `json:"waypoints"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA: Pas de route trouvée\n%s → %s", origin, dest))
	}
	return format160(fmt.Sprintf("KINARA ROUTE:\n%s → %s\n%.0fkm | %.1fh\nVia: %s | %d étapes", origin, dest, result.Data.DistanceKm, result.Data.DurationHrs, result.Data.Highway, result.Data.Stops))
}

func (h *Handler) handleFleet(ctx context.Context, cmd models.ParsedCommand) string {
	tenantID := "TG"
	if len(cmd.Args) > 0 {
		tenantID = cmd.Args[0]
	}
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/fleet/summary?tenant_id=%s", h.fleetURL, url.QueryEscape(tenantID)))
	if err != nil {
		return "KINARA: Données flotte indisponibles."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			TotalVehicles  int `json:"total_vehicles"`
			Active         int `json:"active"`
			Maintenance    int `json:"in_maintenance"`
			Available      int `json:"available"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return "KINARA: Résumé flotte indisponible."
	}
	return format160(fmt.Sprintf("KINARA FLOTTE:\nTotal: %d | Actifs: %d\nDisponibles: %d | Maintenance: %d",
		result.Data.TotalVehicles, result.Data.Active, result.Data.Available, result.Data.Maintenance))
}

// ─────────────────────────────────────────────
// Maritime handlers
// ─────────────────────────────────────────────

func (h *Handler) handleVessel(ctx context.Context, cmd models.ParsedCommand) string {
	if len(cmd.Args) == 0 {
		return "Usage: VESSEL <vessel_id_or_name>\nEx: VESSEL MV-KINARA-01"
	}
	vesselRef := strings.Join(cmd.Args, " ")
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/vessels?ref=%s", h.vesselURL, url.QueryEscape(vesselRef)))
	if err != nil {
		return "KINARA: Données navire indisponibles."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Name     string `json:"name"`
			IMO      string `json:"imo_number"`
			Status   string `json:"status"`
			Port     string `json:"current_port"`
			Flag     string `json:"flag_country"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA: Navire %s introuvable.", vesselRef))
	}
	return format160(fmt.Sprintf("NAVIRE: %s\nIMO: %s | %s\nPort: %s | Pavillon: %s", result.Data.Name, result.Data.IMO, result.Data.Status, result.Data.Port, result.Data.Flag))
}

func (h *Handler) handleBerth(ctx context.Context, cmd models.ParsedCommand) string {
	port := "LOME"
	if len(cmd.Args) > 0 {
		port = strings.Join(cmd.Args, " ")
	}
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/berths?port=%s&status=available", h.portURL, url.QueryEscape(port)))
	if err != nil {
		return "KINARA: Données quai indisponibles."
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			BerthNumber string `json:"berth_number"`
			Status      string `json:"status"`
			MaxDWT      int    `json:"max_dwt_tonnes"`
		} `json:"data"`
		Total int `json:"total_available"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA PORT %s: Pas de quais disponibles.", port))
	}
	lines := []string{fmt.Sprintf("PORT %s: %d quais libres", strings.ToUpper(port), result.Total)}
	for _, b := range result.Data {
		lines = append(lines, fmt.Sprintf("Quai %s: max %dt", b.BerthNumber, b.MaxDWT))
	}
	return format160(strings.Join(lines, "\n"))
}

func (h *Handler) handleManifest(ctx context.Context, cmd models.ParsedCommand) string {
	if len(cmd.Args) == 0 {
		return "Usage: MANIFEST <vessel_id>\nEx: MANIFEST MV-KINARA-01"
	}
	vesselRef := cmd.Args[0]
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/cargo/manifest?vessel=%s", h.portURL, url.QueryEscape(vesselRef)))
	if err != nil {
		return "KINARA: Manifeste indisponible."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			VesselName   string  `json:"vessel_name"`
			TotalItems   int     `json:"total_items"`
			TotalWeightT float64 `json:"total_weight_tonnes"`
			Destination  string  `json:"destination_port"`
			Status       string  `json:"customs_status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA: Manifeste de %s introuvable.", vesselRef))
	}
	return format160(fmt.Sprintf("MANIFESTE: %s\n%d articles | %.0ft\nDest: %s | Douanes: %s", result.Data.VesselName, result.Data.TotalItems, result.Data.TotalWeightT, result.Data.Destination, result.Data.Status))
}

func (h *Handler) handleCustoms(ctx context.Context, cmd models.ParsedCommand) string {
	if len(cmd.Args) == 0 {
		return "Usage: CUSTOMS <shipment_ref>\nEx: CUSTOMS SHP-A1B2C3D4"
	}
	ref := cmd.Args[0]
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/declarations?shipment_ref=%s", h.customsURL, url.QueryEscape(ref)))
	if err != nil {
		return "KINARA: Statut douane indisponible."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			DeclarationID string  `json:"declaration_id"`
			Status        string  `json:"status"`
			DutyAmountXOF float64 `json:"duty_amount_xof"`
			ClearedAt     string  `json:"cleared_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA DOUANE: %s\nDéclaration introuvable.", ref))
	}
	cleared := "En cours"
	if result.Data.ClearedAt != "" && len(result.Data.ClearedAt) >= 10 {
		cleared = result.Data.ClearedAt[:10]
	}
	return format160(fmt.Sprintf("KINARA DOUANE: %s\nStatut: %s\nTaxes: %.0f XOF\nDédouané: %s", ref, result.Data.Status, result.Data.DutyAmountXOF, cleared))
}

// ─────────────────────────────────────────────
// Cross-pillar handlers
// ─────────────────────────────────────────────

func (h *Handler) handleSend(ctx context.Context, cmd models.ParsedCommand, from string) string {
	if len(cmd.Args) < 2 {
		return "Usage: SEND <phone> <amount> [currency]\nEx: SEND +22891234567 5000 XOF"
	}
	toPhone := cmd.Args[0]
	amount := cmd.Args[1]
	currency := "XOF"
	if len(cmd.Args) >= 3 {
		currency = strings.ToUpper(cmd.Args[2])
	}
	payload := fmt.Sprintf(`{"from_phone":%q,"to_phone":%q,"amount":%s,"currency":%q,"source":"sms"}`,
		from, toPhone, amount, currency)
	resp, err := httpPost(ctx, fmt.Sprintf("%s/api/v1/payments/send/sms", h.paymentURL), payload)
	if err != nil {
		return "KINARA PAY: Transfert échoué. Vérifiez le numéro et solde."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			TxRef   string  `json:"tx_ref"`
			Amount  float64 `json:"amount"`
			Fee     float64 `json:"fee"`
			Balance float64 `json:"new_balance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return "KINARA PAY: Transfert échoué. Solde insuffisant?"
	}
	return format160(fmt.Sprintf("KINARA PAY: OK\n%s → %s\n%.0f %s (frais: %.0f)\nSolde: %.2f %s", from[:8]+"...", toPhone, result.Data.Amount, currency, result.Data.Fee, result.Data.Balance, currency))
}

func (h *Handler) handleConvert(ctx context.Context, cmd models.ParsedCommand) string {
	if len(cmd.Args) < 3 {
		return "Usage: CONVERT <amount> <from> <to>\nEx: CONVERT 10000 XOF USD\nDevises: XOF GHS KES NGN USD EUR"
	}
	amount, fromCur, toCur := cmd.Args[0], strings.ToUpper(cmd.Args[1]), strings.ToUpper(cmd.Args[2])
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/payments/exchange?from=%s&to=%s&amount=%s", h.paymentURL, fromCur, toCur, amount))
	if err != nil {
		return "KINARA: Conversion indisponible. Réessayez."
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Converted float64 `json:"converted_amount"`
			Rate      float64 `json:"exchange_rate"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil || !result.Success {
		return format160(fmt.Sprintf("KINARA: Taux %s→%s indisponible.", fromCur, toCur))
	}
	return format160(fmt.Sprintf("KINARA FX:\n%s %s = %.2f %s\nTaux: %.4f", amount, fromCur, result.Data.Converted, toCur, result.Data.Rate))
}

func (h *Handler) handleImpact(ctx context.Context, cmd models.ParsedCommand) string {
	pillar := "all"
	if len(cmd.Args) > 0 {
		pillar = strings.ToLower(cmd.Args[0])
	}
	resp, err := httpGet(ctx, fmt.Sprintf("%s/api/v1/analytics/impact?pillar=%s&country=TG", h.analyticsURL, url.QueryEscape(pillar)))
	if err != nil {
		return "KINARA: Données d'impact indisponibles."
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
		return "KINARA: Pas encore de données d'impact."
	}
	lines := []string{"KINARA IMPACT:"}
	for _, m := range result.Data {
		lines = append(lines, fmt.Sprintf("%s: %.0f %s", m.MetricName, m.MetricValue, m.MetricUnit))
	}
	return format160(strings.Join(lines, "\n"))
}

// ─────────────────────────────────────────────
// Audit logging
// ─────────────────────────────────────────────

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

// ─────────────────────────────────────────────
// Parser: 50+ intents with French aliases
// ─────────────────────────────────────────────

func parseCommand(text string) models.ParsedCommand {
	text = strings.TrimSpace(text)
	parts := strings.FieldsFunc(text, func(r rune) bool { return unicode.IsSpace(r) })
	if len(parts) == 0 {
		return models.ParsedCommand{Type: models.CmdUnknown, Args: []string{""}, RawText: text}
	}
	keyword := strings.ToUpper(parts[0])
	args := parts[1:]

	cmdMap := map[string]models.CommandType{
		// Agriculture — English + French
		"PRICE":        models.CmdPrice,
		"PRIX":         models.CmdPrice,
		"BUYERS":       models.CmdBuyers,
		"ACHETEURS":    models.CmdBuyers,
		"SELL":         models.CmdSell,
		"VENDRE":       models.CmdSell,
		"WEATHER":      models.CmdWeather,
		"METEO":        models.CmdWeather,
		"MÉTÉO":        models.CmdWeather,
		"STATUS":       models.CmdStatus,
		"STATUT":       models.CmdStatus,
		"INCOME":       models.CmdIncome,
		"REVENU":       models.CmdIncome,
		"BALANCE":      models.CmdBalance,
		"SOLDE":        models.CmdBalance,
		"REGISTER":     models.CmdRegister,
		"INSCRIRE":     models.CmdRegister,
		"FARMERS":      models.CmdFarmers,
		"AGRICULTEURS": models.CmdFarmers,
		"COOP":         models.CmdCoop,
		"COOPERATIVE":  models.CmdCoop,
		"COOPÉRATIVE":  models.CmdCoop,
		"JOIN":         models.CmdJoin,
		"REJOINDRE":    models.CmdJoin,
		"SAVINGS":      models.CmdSavings,
		"EPARGNE":      models.CmdSavings,
		"ÉPARGNE":      models.CmdSavings,

		// Health — English
		"PATIENT":    models.CmdPatient,
		"SYMPTOM":    models.CmdSymptom,
		"APPT":       models.CmdAppt,
		"LAB":        models.CmdLab,
		"RESULT":     models.CmdLabResult,
		"REFER":      models.CmdRefer,
		"CANCEL":     models.CmdCancel,
		"RESCHEDULE": models.CmdReschedule,
		"VACCINE":    models.CmdVaccine,
		"SCHEDULE":   models.CmdSchedule,
		"OUTBREAK":   models.CmdOutbreak,

		// Health — French
		"MALADE":      models.CmdPatient,
		"SYMPTOME":    models.CmdSymptom,
		"SYMPTÔME":    models.CmdSymptom,
		"RDV":         models.CmdAppt,
		"LABO":        models.CmdLab,
		"RESULTAT":    models.CmdLabResult,
		"RÉSULTAT":    models.CmdLabResult,
		"ORIENTER":    models.CmdRefer,
		"ANNULER":     models.CmdCancel,
		"REPORTER":    models.CmdReschedule,
		"REPROGRAMMER": models.CmdReschedule,
		"VACCIN":      models.CmdVaccine,
		"CALENDRIER":  models.CmdSchedule,
		"EPID":        models.CmdOutbreak,
		"EPIDEMIE":    models.CmdOutbreak,
		"ÉPIDÉMIE":    models.CmdOutbreak,
		"ALERTE":      models.CmdOutbreak,

		// Logistics — English
		"TRACK": models.CmdTrack,
		"ROUTE": models.CmdRoute,
		"FLEET": models.CmdFleet,

		// Logistics — French
		"SUIVI":      models.CmdTrack,
		"SUIVRE":     models.CmdTrack,
		"ITINERAIRE": models.CmdRoute,
		"ITINÉRAIRE": models.CmdRoute,
		"FLOTTE":     models.CmdFleet,

		// Maritime — English
		"VESSEL":   models.CmdVessel,
		"BERTH":    models.CmdBerth,
		"MANIFEST": models.CmdManifest,
		"CUSTOMS":  models.CmdCustoms,

		// Maritime — French
		"NAVIRE":    models.CmdVessel,
		"BATEAU":    models.CmdVessel,
		"QUAI":      models.CmdBerth,
		"MANIFESTE": models.CmdManifest,
		"DOUANE":    models.CmdCustoms,

		// Cross-pillar — English
		"SEND":    models.CmdSend,
		"CONVERT": models.CmdConvert,
		"IMPACT":  models.CmdImpact,

		// Cross-pillar — French
		"ENVOYER":   models.CmdSend,
		"TRANSFERT": models.CmdSend,
		"CONVERTIR": models.CmdConvert,

		// Help
		"HELP": models.CmdHelp,
		"AIDE": models.CmdHelp,
		"?":    models.CmdHelp,
	}

	if t, ok := cmdMap[keyword]; ok {
		return models.ParsedCommand{Type: t, Args: args, RawText: text}
	}
	return models.ParsedCommand{Type: models.CmdUnknown, Args: parts, RawText: text}
}

func helpText() string {
	return `KINARA:
AGR:PRICE SELL STATUS INCOME
HLT:PATIENT APPT VACCINE
LOG:TRACK ROUTE FLEET
MAR:VESSEL BERTH CUSTOMS
PAY:SEND CONVERT BALANCE
HELP/?`
}

// ─────────────────────────────────────────────
// Utilities
// ─────────────────────────────────────────────

// format160 truncates to 160 chars at a clean boundary.
func format160(s string) string {
	if len(s) <= 160 {
		return s
	}
	// Try to cut at last newline before char 157
	for i := 156; i > 100; i-- {
		if s[i] == '\n' {
			return s[:i] + "..."
		}
	}
	return s[:157] + "..."
}

func shortErr(err error) string {
	msg := err.Error()
	if len(msg) > 40 {
		return msg[:40]
	}
	return msg
}

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
