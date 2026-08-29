package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/klinova/kinara-os/auth-service/auth"
	cryptopkg "github.com/klinova/kinara-os/auth-service/crypto"
	"github.com/klinova/kinara-os/auth-service/db"
	"github.com/klinova/kinara-os/auth-service/middleware"
	"github.com/klinova/kinara-os/auth-service/models"
	"github.com/redis/go-redis/v9"
)

// Handler bundles all auth service dependencies.
type Handler struct {
	queries    *db.Queries
	enc        *cryptopkg.Encryptor
	issuer     *auth.Issuer
	mtlsCfg    auth.MTLSConfig
	rdb        *redis.Client
	logger     *slog.Logger
}

func New(
	queries *db.Queries,
	enc *cryptopkg.Encryptor,
	issuer *auth.Issuer,
	mtlsCfg auth.MTLSConfig,
	rdb *redis.Client,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		queries: queries, enc: enc, issuer: issuer,
		mtlsCfg: mtlsCfg, rdb: rdb, logger: logger,
	}
}

// Register mounts all auth routes.
func (h *Handler) Register(r *mux.Router, jwtMiddleware func(http.Handler) http.Handler) {
	// Public endpoints (no JWT required)
	r.HandleFunc("/api/v1/auth/register", h.register).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/auth/login", h.login).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/auth/token/refresh", h.refreshToken).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/auth/token/validate", h.validateToken).Methods(http.MethodPost)

	// Protected endpoints (JWT required)
	protected := r.PathPrefix("/api/v1/auth").Subrouter()
	protected.Use(jwtMiddleware)

	protected.HandleFunc("/profile", h.getProfile).Methods(http.MethodGet)
	protected.HandleFunc("/profile", h.updateProfile).Methods(http.MethodPut)
	protected.HandleFunc("/mfa/enroll", h.enrollMFA).Methods(http.MethodPost)
	protected.HandleFunc("/mfa/verify", h.verifyMFA).Methods(http.MethodPost)
	protected.HandleFunc("/apikey/generate", h.generateAPIKey).Methods(http.MethodPost)
	protected.HandleFunc("/roles", h.listRoles).Methods(http.MethodGet)
	protected.HandleFunc("/permissions/check", h.checkPermission).Methods(http.MethodPost)
	protected.HandleFunc("/certs/issue", h.issueCert).Methods(http.MethodPost)

	// Admin-only endpoints
	admin := r.PathPrefix("/api/v1/auth").Subrouter()
	admin.Use(jwtMiddleware)
	admin.Use(middleware.RequireRole("admin"))
	admin.HandleFunc("/access-log", h.getAccessLog).Methods(http.MethodGet)
}

// ─── Registration ─────────────────────────────────────────────────────────────

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" || req.FullName == "" {
		h.badRequest(w, "username, email, password, and full_name are required")
		return
	}
	if len(req.Password) < 8 {
		h.badRequest(w, "password must be at least 8 characters")
		return
	}

	hash, err := cryptopkg.HashPassword(req.Password)
	if err != nil {
		h.internalError(w, err)
		return
	}

	userRow, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
	})
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			h.json(w, http.StatusConflict, models.APIResponse{
				Success: false,
				Error:   &models.APIError{Code: "CONFLICT", Message: "username or email already registered"},
			})
			return
		}
		h.internalError(w, err)
		return
	}

	// Create profile
	fullNameEnc, err := h.enc.EncryptString(req.FullName)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.queries.UpsertProfile(r.Context(), db.UpsertProfileParams{
		UserID:      userRow.ID,
		FullNameEnc: fullNameEnc,
		Country:     req.Country,
	})

	// Assign default role
	roleName := req.Role
	if roleName == "" {
		roleName = "patient"
	}
	if role, err := h.queries.GetRoleByName(r.Context(), roleName); err == nil {
		h.queries.AssignRole(r.Context(), userRow.ID, role.ID, nil)
	}

	h.logAccess(r, &userRow.ID, "register", "users", models.LogSuccess, "")

	user := rowToUser(userRow)
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: user})
}

// ─── Login ────────────────────────────────────────────────────────────────────

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	// Rate limit: 5 attempts per minute per IP
	allowed, _ := middleware.CheckLoginRateLimit(h.rdb, r)
	if !allowed {
		h.logAccess(r, nil, "login", "auth", models.LogFailure, "rate limited: "+req.Username)
		h.json(w, http.StatusTooManyRequests, models.APIResponse{
			Success: false,
			Error:   &models.APIError{Code: "RATE_LIMITED", Message: "too many login attempts, wait 1 minute"},
		})
		return
	}

	userRow, err := h.queries.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		// Also check by email
		userRow, err = h.queries.GetUserByEmail(r.Context(), req.Username)
		if err != nil {
			middleware.RecordFailedLogin(h.rdb, r)
			h.logAccess(r, nil, "login", "auth", models.LogFailure, "user not found: "+req.Username)
			h.unauthorized(w, "invalid credentials")
			return
		}
	}

	if userRow.Status != models.UserActive {
		h.logAccess(r, &userRow.ID, "login", "auth", models.LogDenied, "account not active")
		h.unauthorized(w, "account is not active")
		return
	}

	if err := cryptopkg.VerifyPassword(userRow.PasswordHash, req.Password); err != nil {
		middleware.RecordFailedLogin(h.rdb, r)
		h.logAccess(r, &userRow.ID, "login", "auth", models.LogFailure, "wrong password")
		h.unauthorized(w, "invalid credentials")
		return
	}

	// Determine primary role
	roles, _ := h.queries.GetUserRoles(r.Context(), userRow.ID)
	primaryRole := "patient"
	if len(roles) > 0 {
		primaryRole = roles[0]
	}

	// MFA check: admin role always requires MFA if a verified device exists
	mfaVerified := false
	device, mfaErr := h.queries.GetVerifiedMFADevice(r.Context(), userRow.ID)
	if mfaErr == nil && device != nil {
		// User has MFA enrolled
		if req.MFACode == "" {
			h.json(w, http.StatusOK, models.APIResponse{
				Success: true,
				Data:    &models.LoginResponse{NeedsMFA: true},
			})
			return
		}
		// Decrypt TOTP secret and verify
		secret, err := h.enc.DecryptString(device.SecretEnc)
		if err != nil || !cryptopkg.VerifyTOTP(secret, req.MFACode) {
			h.logAccess(r, &userRow.ID, "login_mfa", "auth", models.LogFailure, "invalid MFA code")
			h.unauthorized(w, "invalid MFA code")
			return
		}
		mfaVerified = true
	}

	// Issue access token
	accessToken, err := h.issuer.IssueAccessToken(userRow.ID, userRow.Username, primaryRole, roles)
	if err != nil {
		h.internalError(w, err)
		return
	}

	// Generate and store refresh token
	refreshToken, refreshHash, err := cryptopkg.GenerateRefreshToken()
	if err != nil {
		h.internalError(w, err)
		return
	}

	sessionExpiry := time.Now().Add(7 * 24 * time.Hour)
	_, err = h.queries.CreateSession(r.Context(), db.CreateSessionParams{
		UserID:           userRow.ID,
		RefreshTokenHash: refreshHash,
		MFAVerified:      mfaVerified,
		IPAddress:        remoteIP(r),
		UserAgent:        r.UserAgent(),
		ExpiresAt:        sessionExpiry,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.queries.UpdateLastLogin(r.Context(), userRow.ID)
	h.logAccess(r, &userRow.ID, "login", "auth", models.LogSuccess, "")

	user := rowToUser(userRow)
	h.json(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: &models.LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    auth.AccessTokenTTLSeconds(),
			User:         user,
		},
	})
}

// ─── Token endpoints ──────────────────────────────────────────────────────────

func (h *Handler) refreshToken(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}
	if req.RefreshToken == "" {
		h.badRequest(w, "refresh_token is required")
		return
	}

	tokenHash := cryptopkg.HashRefreshToken(req.RefreshToken)
	session, err := h.queries.GetSessionByRefreshHash(r.Context(), tokenHash)
	if err != nil {
		h.unauthorized(w, "invalid or expired refresh token")
		return
	}

	userRow, err := h.queries.GetUserByID(r.Context(), session.UserID)
	if err != nil || userRow.Status != models.UserActive {
		h.unauthorized(w, "account not active")
		return
	}

	roles, _ := h.queries.GetUserRoles(r.Context(), userRow.ID)
	primaryRole := "patient"
	if len(roles) > 0 {
		primaryRole = roles[0]
	}

	accessToken, err := h.issuer.IssueAccessToken(userRow.ID, userRow.Username, primaryRole, roles)
	if err != nil {
		h.internalError(w, err)
		return
	}

	// Rotate refresh token (invalidate old, issue new)
	h.queries.DeleteSession(r.Context(), session.ID)

	newRefreshToken, newHash, err := cryptopkg.GenerateRefreshToken()
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.queries.CreateSession(r.Context(), db.CreateSessionParams{
		UserID:           userRow.ID,
		RefreshTokenHash: newHash,
		MFAVerified:      session.MFAVerified,
		IPAddress:        remoteIP(r),
		UserAgent:        r.UserAgent(),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	})

	h.logAccess(r, &userRow.ID, "token_refresh", "auth", models.LogSuccess, "")

	h.json(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: &models.LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    auth.AccessTokenTTLSeconds(),
		},
	})
}

func (h *Handler) validateToken(w http.ResponseWriter, r *http.Request) {
	var req models.ValidateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	claims, err := h.issuer.Validate(req.Token)
	if err != nil {
		h.json(w, http.StatusOK, models.APIResponse{
			Success: true,
			Data:    &models.ValidateTokenResponse{Valid: false},
		})
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: &models.ValidateTokenResponse{
			Valid:    true,
			UserID:   claims.UserID.String(),
			Username: claims.Username,
			Role:     claims.Role,
			Scopes:   claims.Scopes,
		},
	})
}

// ─── Profile endpoints ────────────────────────────────────────────────────────

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	userRow, err := h.queries.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		h.notFound(w, "user not found")
		return
	}

	profileRow, err := h.queries.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		h.json(w, http.StatusOK, models.APIResponse{
			Success: true,
			Data:    rowToUser(userRow),
		})
		return
	}

	profile, err := h.decryptProfile(profileRow)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"user":    rowToUser(userRow),
			"profile": profile,
		},
	})
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	// Re-encrypt updated fields
	params := db.UpsertProfileParams{
		UserID:  claims.UserID,
		Country: req.Country,
	}

	if req.FullName != "" {
		enc, err := h.enc.EncryptString(req.FullName)
		if err != nil {
			h.internalError(w, err)
			return
		}
		params.FullNameEnc = enc
	} else {
		// Keep existing — fetch first
		existing, err := h.queries.GetProfile(r.Context(), claims.UserID)
		if err == nil {
			params.FullNameEnc = existing.FullNameEnc
		}
	}

	deptEnc, _ := h.enc.EncryptOptional(req.Department)
	params.DepartmentEnc = deptEnc

	phoneEnc, _ := h.enc.EncryptOptional(req.Phone)
	params.PhoneEnc = phoneEnc

	row, err := h.queries.UpsertProfile(r.Context(), params)
	if err != nil {
		h.internalError(w, err)
		return
	}

	profile, err := h.decryptProfile(row)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: profile})
}

// ─── MFA endpoints ────────────────────────────────────────────────────────────

func (h *Handler) enrollMFA(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	secret, err := cryptopkg.GenerateTOTPSecret()
	if err != nil {
		h.internalError(w, err)
		return
	}

	secretEnc, err := h.enc.EncryptString(secret)
	if err != nil {
		h.internalError(w, err)
		return
	}

	device, err := h.queries.CreateMFADevice(r.Context(), db.CreateMFADeviceParams{
		UserID:    claims.UserID,
		Type:      models.MFATOTPApp,
		Name:      "Authenticator App",
		SecretEnc: secretEnc,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.logAccess(r, &claims.UserID, "mfa_enroll", "mfa_devices", models.LogSuccess, "")

	h.json(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data: &models.EnrollMFAResponse{
			DeviceID: device.ID.String(),
			Secret:   secret,
			OTPAuth:  cryptopkg.TOTPAuthURI(secret, claims.Username),
		},
	})
}

func (h *Handler) verifyMFA(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req models.VerifyMFARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		h.badRequest(w, "invalid device_id")
		return
	}

	device, err := h.queries.GetMFADeviceByID(r.Context(), deviceID)
	if err != nil || device.UserID != claims.UserID {
		h.notFound(w, "MFA device not found")
		return
	}

	secret, err := h.enc.DecryptString(device.SecretEnc)
	if err != nil {
		h.internalError(w, err)
		return
	}

	if !cryptopkg.VerifyTOTP(secret, req.Code) {
		h.logAccess(r, &claims.UserID, "mfa_verify", "mfa_devices", models.LogFailure, "invalid code")
		h.json(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error:   &models.APIError{Code: "INVALID_MFA", Message: "invalid MFA code"},
		})
		return
	}

	if err := h.queries.VerifyMFADevice(r.Context(), deviceID); err != nil {
		h.internalError(w, err)
		return
	}

	h.logAccess(r, &claims.UserID, "mfa_verify", "mfa_devices", models.LogSuccess, "")
	h.json(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    map[string]string{"status": "mfa_enrolled"},
	})
}

// ─── API Key endpoints ────────────────────────────────────────────────────────

func (h *Handler) generateAPIKey(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req models.GenerateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}
	if req.Name == "" {
		h.badRequest(w, "name is required")
		return
	}

	plaintextKey, keyHash, err := cryptopkg.GenerateAPIKey()
	if err != nil {
		h.internalError(w, err)
		return
	}

	row, err := h.queries.CreateAPIKey(r.Context(), db.CreateAPIKeyParams{
		UserID:      claims.UserID,
		Name:        req.Name,
		KeyHash:     keyHash,
		Permissions: req.Permissions,
		ExpiresAt:   req.ExpiresAt,
	})
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.logAccess(r, &claims.UserID, "apikey_create", "api_keys", models.LogSuccess, req.Name)

	h.json(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data: &models.GenerateAPIKeyResponse{
			Key: plaintextKey,
			APIKey: &models.APIKey{
				ID:          row.ID,
				UserID:      row.UserID,
				Name:        row.Name,
				Permissions: row.Permissions,
				CreatedAt:   row.CreatedAt,
				ExpiresAt:   row.ExpiresAt,
			},
		},
	})
}

// ─── RBAC endpoints ───────────────────────────────────────────────────────────

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.queries.ListRoles(r.Context())
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.json(w, http.StatusOK, models.APIResponse{Success: true, Data: roles})
}

func (h *Handler) checkPermission(w http.ResponseWriter, r *http.Request) {
	var req models.CheckPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "invalid JSON body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.badRequest(w, "invalid user_id")
		return
	}

	allowed, role, err := h.queries.CheckPermission(r.Context(), userID, req.Resource, req.Action)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.json(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    &models.CheckPermissionResponse{Allowed: allowed, Role: role},
	})
}

// ─── Certificate issuance ─────────────────────────────────────────────────────

func (h *Handler) issueCert(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if !claims.IsAllowedRole("admin", "system") {
		h.forbidden(w)
		return
	}

	var req struct {
		ServiceName string   `json:"service_name"`
		DNSNames    []string `json:"dns_names"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ServiceName == "" {
		h.badRequest(w, "service_name is required")
		return
	}

	// CAKeyPath is optional — skip if not configured
	if h.mtlsCfg.CAKeyPath == "" {
		h.json(w, http.StatusServiceUnavailable, models.APIResponse{
			Success: false,
			Error:   &models.APIError{Code: "NOT_CONFIGURED", Message: "CA key not configured for cert issuance"},
		})
		return
	}

	bundle, err := auth.IssueCert(h.mtlsCfg, req.ServiceName, req.DNSNames)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.logAccess(r, &claims.UserID, "cert_issue", req.ServiceName, models.LogSuccess, "")
	h.json(w, http.StatusCreated, models.APIResponse{Success: true, Data: bundle})
}

// ─── Access Log handler ───────────────────────────────────────────────────────

func (h *Handler) getAccessLog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}

	params := db.ListAccessLogParams{Page: page, Limit: limit}
	if uid := q.Get("user_id"); uid != "" {
		if id, err := uuid.Parse(uid); err == nil {
			params.UserID = &id
		}
	}
	if s := q.Get("status"); s != "" {
		status := models.AccessLogStatus(s)
		params.Status = &status
	}

	logs, err := h.queries.ListAccessLog(r.Context(), params)
	if err != nil {
		h.internalError(w, err)
		return
	}
	total, _ := h.queries.CountAccessLog(r.Context(), params)
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	h.json(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    logs,
		Meta:    &models.PageMeta{Page: page, Limit: limit, Total: total, TotalPages: totalPages},
	})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func rowToUser(row *models.UserRow) *models.User {
	return &models.User{
		ID:            row.ID,
		Username:      row.Username,
		Email:         row.Email,
		Status:        row.Status,
		EmailVerified: row.EmailVerified,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		LastLoginAt:   row.LastLoginAt,
	}
}

func (h *Handler) decryptProfile(row *models.UserProfileRow) (*models.UserProfile, error) {
	fullName, err := h.enc.DecryptString(row.FullNameEnc)
	if err != nil {
		return nil, err
	}
	dept, _ := h.enc.DecryptOptional(row.DepartmentEnc)
	phone, _ := h.enc.DecryptOptional(row.PhoneEnc)
	return &models.UserProfile{
		UserID:     row.UserID,
		FullName:   fullName,
		Department: dept,
		Phone:      phone,
		Country:    row.Country,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

func (h *Handler) logAccess(r *http.Request, userID *uuid.UUID, action, resource string, status models.AccessLogStatus, details string) {
	h.queries.InsertAccessLog(r.Context(), db.InsertAccessLogParams{
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Status:    status,
		IPAddress: remoteIP(r),
		UserAgent: r.UserAgent(),
		Details:   details,
	})
}

func (h *Handler) json(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func (h *Handler) badRequest(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusBadRequest, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "BAD_REQUEST", Message: msg},
	})
}

func (h *Handler) unauthorized(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusUnauthorized, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "UNAUTHORIZED", Message: msg},
	})
}

func (h *Handler) forbidden(w http.ResponseWriter) {
	h.json(w, http.StatusForbidden, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "FORBIDDEN", Message: "insufficient role"},
	})
}

func (h *Handler) notFound(w http.ResponseWriter, msg string) {
	h.json(w, http.StatusNotFound, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "NOT_FOUND", Message: msg},
	})
}

func (h *Handler) internalError(w http.ResponseWriter, err error) {
	h.logger.Error("internal error", "error", err)
	h.json(w, http.StatusInternalServerError, models.APIResponse{
		Success: false, Error: &models.APIError{Code: "INTERNAL_ERROR", Message: "an internal error occurred"},
	})
}

func remoteIP(r *http.Request) string {
	// Check X-Forwarded-For first (set by load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx]
	}
	return addr
}

// ensure fmt is used
var _ = fmt.Sprintf
