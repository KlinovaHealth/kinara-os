package models

import (
	"time"

	"github.com/google/uuid"
)

// ─── Enumerations ─────────────────────────────────────────────────────────────

type UserStatus string

const (
	UserActive    UserStatus = "active"
	UserInactive  UserStatus = "inactive"
	UserSuspended UserStatus = "suspended"
)

type MFAType string

const (
	MFATOTPApp MFAType = "totp"
)

type AccessLogStatus string

const (
	LogSuccess AccessLogStatus = "success"
	LogFailure AccessLogStatus = "failure"
	LogDenied  AccessLogStatus = "denied"
)

// Predefined roles in Kinara Governance OS — cross-pillar.
var SystemRoles = []string{
	"admin",
	"clinician",
	"nurse",
	"doctor",
	"patient",
	"frontdesk",
	"analyst",
	"government",
	"ministry_official",
	"facility_admin",
	"farmer",
	"cooperative_manager",
	"logistics",
	"fleet_operator",
	"port_operator",
	"customs_officer",
	"pharmacist",
	"system", // inter-service machine identity
}

// ─── Domain models ────────────────────────────────────────────────────────────

// User is the core identity record. Password is always stored hashed.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	Status       UserStatus `json:"status"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// UserRow is the database row including password_hash (never exposed in API).
type UserRow struct {
	ID            uuid.UUID  `db:"id"`
	Username      string     `db:"username"`
	Email         string     `db:"email"`
	PasswordHash  string     `db:"password_hash"`
	Status        UserStatus `db:"status"`
	EmailVerified bool       `db:"email_verified"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
	LastLoginAt   *time.Time `db:"last_login_at"`
}

// UserProfile holds extended user data. Sensitive fields are AES-256-GCM encrypted.
type UserProfile struct {
	UserID     uuid.UUID `json:"user_id"`
	FullName   string    `json:"full_name"`
	Department string    `json:"department,omitempty"`
	Phone      string    `json:"phone,omitempty"`
	Country    string    `json:"country,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// UserProfileRow has encrypted personal data fields.
type UserProfileRow struct {
	UserID          uuid.UUID `db:"user_id"`
	FullNameEnc     string    `db:"full_name_enc"`
	DepartmentEnc   *string   `db:"department_enc"`
	PhoneEnc        *string   `db:"phone_enc"`
	Country         string    `db:"country"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// Role is a named set of permissions.
type Role struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Permission grants access to a specific resource action.
type Permission struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	CreatedAt   time.Time `json:"created_at"`
}

// APIKey is a long-lived credential for service-to-service calls.
// The plaintext key is returned exactly once at creation; only the hash is stored.
type APIKey struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Name        string     `json:"name"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

// APIKeyRow stores the SHA-256 hash of the key, never the plaintext.
type APIKeyRow struct {
	ID          uuid.UUID  `db:"id"`
	UserID      uuid.UUID  `db:"user_id"`
	Name        string     `db:"name"`
	KeyHash     string     `db:"key_hash"`
	Permissions []string   `db:"permissions"`
	CreatedAt   time.Time  `db:"created_at"`
	ExpiresAt   *time.Time `db:"expires_at"`
	LastUsedAt  *time.Time `db:"last_used_at"`
}

// Session represents an active user session. Token hash stored, never plaintext.
type Session struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	MFAVerified bool      `json:"mfa_verified"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// SessionRow includes the refresh token hash.
type SessionRow struct {
	ID               uuid.UUID `db:"id"`
	UserID           uuid.UUID `db:"user_id"`
	RefreshTokenHash string    `db:"refresh_token_hash"`
	MFAVerified      bool      `db:"mfa_verified"`
	IPAddress        string    `db:"ip_address"`
	UserAgent        string    `db:"user_agent"`
	ExpiresAt        time.Time `db:"expires_at"`
	CreatedAt        time.Time `db:"created_at"`
}

// MFADevice is a TOTP authenticator app registered by a user.
type MFADevice struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Type      MFAType   `json:"type"`
	Name      string    `json:"name"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
}

// MFADeviceRow stores the TOTP secret encrypted with AES-256-GCM.
type MFADeviceRow struct {
	ID            uuid.UUID `db:"id"`
	UserID        uuid.UUID `db:"user_id"`
	Type          MFAType   `db:"type"`
	Name          string    `db:"name"`
	SecretEnc     string    `db:"secret_enc"`
	Verified      bool      `db:"verified"`
	CreatedAt     time.Time `db:"created_at"`
}

// AccessLog is an immutable record of every authentication event.
type AccessLog struct {
	ID        uuid.UUID       `json:"id"`
	UserID    *uuid.UUID      `json:"user_id,omitempty"`
	Action    string          `json:"action"`
	Resource  string          `json:"resource"`
	Status    AccessLogStatus `json:"status"`
	IPAddress string          `json:"ip_address"`
	UserAgent string          `json:"user_agent"`
	Details   string          `json:"details,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// ─── Request / Response types ─────────────────────────────────────────────────

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Country  string `json:"country"`
	Role     string `json:"role,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	MFACode  string `json:"mfa_code,omitempty"`
}

type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"` // seconds
	User         *User     `json:"user"`
	NeedsMFA     bool      `json:"needs_mfa,omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ValidateTokenRequest struct {
	Token string `json:"token"`
}

type ValidateTokenResponse struct {
	Valid    bool     `json:"valid"`
	UserID   string   `json:"user_id,omitempty"`
	Username string   `json:"username,omitempty"`
	Role     string   `json:"role,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
}

type UpdateProfileRequest struct {
	FullName   string `json:"full_name,omitempty"`
	Department string `json:"department,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Country    string `json:"country,omitempty"`
}

type EnrollMFAResponse struct {
	DeviceID string `json:"device_id"`
	Secret   string `json:"secret"`
	OTPAuth  string `json:"otp_auth_uri"`
}

type VerifyMFARequest struct {
	DeviceID string `json:"device_id"`
	Code     string `json:"code"`
}

type GenerateAPIKeyRequest struct {
	Name        string     `json:"name"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type GenerateAPIKeyResponse struct {
	Key    string `json:"key"` // plaintext, returned once
	APIKey *APIKey `json:"api_key"`
}

type CheckPermissionRequest struct {
	UserID   string `json:"user_id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type CheckPermissionResponse struct {
	Allowed bool   `json:"allowed"`
	Role    string `json:"role,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *PageMeta   `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PageMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}
