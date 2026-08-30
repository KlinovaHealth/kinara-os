package models

import (
	"time"

	"github.com/google/uuid"
)

type SMSProvider string
type SMSDirection string
type CommandType string

const (
	ProviderTwilio         SMSProvider = "twilio"
	ProviderAfricastalking SMSProvider = "africastalking"

	DirectionInbound  SMSDirection = "inbound"
	DirectionOutbound SMSDirection = "outbound"

	// Agriculture
	CmdPrice    CommandType = "PRICE"
	CmdBuyers   CommandType = "BUYERS"
	CmdSell     CommandType = "SELL"
	CmdWeather  CommandType = "WEATHER"
	CmdStatus   CommandType = "STATUS"
	CmdIncome   CommandType = "INCOME"
	CmdBalance  CommandType = "BALANCE"
	CmdRegister CommandType = "REGISTER"
	CmdFarmers  CommandType = "FARMERS"
	CmdCoop     CommandType = "COOP"
	CmdJoin     CommandType = "JOIN"
	CmdSavings  CommandType = "SAVINGS"

	// Health
	CmdPatient    CommandType = "PATIENT"
	CmdSymptom    CommandType = "SYMPTOM"
	CmdAppt       CommandType = "APPT"
	CmdLab        CommandType = "LAB"
	CmdLabResult  CommandType = "LABRESULT"
	CmdRefer      CommandType = "REFER"
	CmdCancel     CommandType = "CANCEL"
	CmdReschedule CommandType = "RESCHEDULE"
	CmdVaccine    CommandType = "VACCINE"
	CmdSchedule   CommandType = "SCHEDULE"
	CmdOutbreak   CommandType = "OUTBREAK"

	// Logistics
	CmdTrack CommandType = "TRACK"
	CmdRoute CommandType = "ROUTE"
	CmdFleet CommandType = "FLEET"

	// Maritime
	CmdVessel   CommandType = "VESSEL"
	CmdBerth    CommandType = "BERTH"
	CmdManifest CommandType = "MANIFEST"
	CmdCustoms  CommandType = "CUSTOMS"

	// Cross-pillar
	CmdSend    CommandType = "SEND"
	CmdConvert CommandType = "CONVERT"
	CmdImpact  CommandType = "IMPACT"

	CmdHelp    CommandType = "HELP"
	CmdUnknown CommandType = "UNKNOWN"
)

type InboundSMS struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Body    string `json:"body"`
	Country string `json:"country"`
}

type TwilioWebhook struct {
	MessageSid string `form:"MessageSid"`
	From       string `form:"From"`
	To         string `form:"To"`
	Body       string `form:"Body"`
	NumMedia   string `form:"NumMedia"`
	Country    string `form:"FromCountry"`
}

type AfricastalkingWebhook struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
	Date string `json:"date"`
	ID   string `json:"id"`
}

type ParsedCommand struct {
	Type    CommandType
	Args    []string
	RawText string
	From    string
}

type SMSLog struct {
	ID        uuid.UUID    `json:"id"`
	Provider  SMSProvider  `json:"provider"`
	Direction SMSDirection `json:"direction"`
	From      string       `json:"from"`
	To        string       `json:"to"`
	Body      string       `json:"body"`
	Response  string       `json:"response,omitempty"`
	Command   string       `json:"command,omitempty"`
	Success   bool         `json:"success"`
	CreatedAt time.Time    `json:"created_at"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
