package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/auth-service/models"
)

// Queries wraps a pgxpool.Pool and exposes typed query methods.
type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

// ─── Users ────────────────────────────────────────────────────────────────────

type CreateUserParams struct {
	Username     string
	Email        string
	PasswordHash string
}

func (q *Queries) CreateUser(ctx context.Context, p CreateUserParams) (*models.UserRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, username, email, password_hash, status, email_verified,
		          created_at, updated_at, last_login_at`,
		p.Username, p.Email, p.PasswordHash,
	)
	return scanUserRow(row)
}

func (q *Queries) GetUserByID(ctx context.Context, id uuid.UUID) (*models.UserRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, status, email_verified,
		       created_at, updated_at, last_login_at
		FROM users WHERE id = $1`, id)
	return scanUserRow(row)
}

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (*models.UserRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, status, email_verified,
		       created_at, updated_at, last_login_at
		FROM users WHERE username = $1`, username)
	return scanUserRow(row)
}

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (*models.UserRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, status, email_verified,
		       created_at, updated_at, last_login_at
		FROM users WHERE email = $1`, email)
	return scanUserRow(row)
}

func (q *Queries) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, userID)
	return err
}

func (q *Queries) UpdateUserStatus(ctx context.Context, userID uuid.UUID, status models.UserStatus) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1`, userID, status)
	return err
}

func scanUserRow(scanner interface{ Scan(...any) error }) (*models.UserRow, error) {
	var u models.UserRow
	err := scanner.Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Status,
		&u.EmailVerified, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &u, err
}

// ─── User Profiles ────────────────────────────────────────────────────────────

type UpsertProfileParams struct {
	UserID        uuid.UUID
	FullNameEnc   string
	DepartmentEnc *string
	PhoneEnc      *string
	Country       string
}

func (q *Queries) UpsertProfile(ctx context.Context, p UpsertProfileParams) (*models.UserProfileRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO user_profiles (user_id, full_name_enc, department_enc, phone_enc, country)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			full_name_enc  = EXCLUDED.full_name_enc,
			department_enc = COALESCE(EXCLUDED.department_enc, user_profiles.department_enc),
			phone_enc      = COALESCE(EXCLUDED.phone_enc, user_profiles.phone_enc),
			country        = CASE WHEN EXCLUDED.country = '' THEN user_profiles.country ELSE EXCLUDED.country END,
			updated_at     = NOW()
		RETURNING user_id, full_name_enc, department_enc, phone_enc, country, updated_at`,
		p.UserID, p.FullNameEnc, p.DepartmentEnc, p.PhoneEnc, p.Country,
	)
	return scanProfileRow(row)
}

func (q *Queries) GetProfile(ctx context.Context, userID uuid.UUID) (*models.UserProfileRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT user_id, full_name_enc, department_enc, phone_enc, country, updated_at
		FROM user_profiles WHERE user_id = $1`, userID)
	return scanProfileRow(row)
}

func scanProfileRow(scanner interface{ Scan(...any) error }) (*models.UserProfileRow, error) {
	var p models.UserProfileRow
	err := scanner.Scan(&p.UserID, &p.FullNameEnc, &p.DepartmentEnc, &p.PhoneEnc, &p.Country, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("profile not found")
	}
	return &p, err
}

// ─── Roles ────────────────────────────────────────────────────────────────────

func (q *Queries) ListRoles(ctx context.Context) ([]*models.Role, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, name, description, created_at FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.Role
	for rows.Next() {
		var r models.Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, &r)
	}
	return result, rows.Err()
}

func (q *Queries) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	var r models.Role
	err := q.pool.QueryRow(ctx,
		`SELECT id, name, description, created_at FROM roles WHERE name = $1`, name,
	).Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("role not found: %s", name)
	}
	return &r, err
}

func (q *Queries) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT r.name FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		roles = append(roles, name)
	}
	return roles, rows.Err()
}

func (q *Queries) AssignRole(ctx context.Context, userID, roleID uuid.UUID, grantedBy *uuid.UUID) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id, granted_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role_id) DO NOTHING`,
		userID, roleID, grantedBy,
	)
	return err
}

// ─── Permissions ──────────────────────────────────────────────────────────────

func (q *Queries) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, string, error) {
	var role string
	err := q.pool.QueryRow(ctx, `
		SELECT r.name FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		JOIN role_permissions rp ON rp.role_id = r.id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE ur.user_id = $1 AND p.resource = $2 AND p.action = $3
		LIMIT 1`,
		userID, resource, action,
	).Scan(&role)
	if err == pgx.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, role, nil
}

// ─── API Keys ─────────────────────────────────────────────────────────────────

type CreateAPIKeyParams struct {
	UserID      uuid.UUID
	Name        string
	KeyHash     string
	Permissions []string
	ExpiresAt   *time.Time
}

func (q *Queries) CreateAPIKey(ctx context.Context, p CreateAPIKeyParams) (*models.APIKeyRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO api_keys (user_id, name, key_hash, permissions, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, name, key_hash, permissions, created_at, expires_at, last_used_at`,
		p.UserID, p.Name, p.KeyHash, p.Permissions, p.ExpiresAt,
	)
	return scanAPIKeyRow(row)
}

func (q *Queries) GetAPIKeyByHash(ctx context.Context, keyHash string) (*models.APIKeyRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, user_id, name, key_hash, permissions, created_at, expires_at, last_used_at
		FROM api_keys WHERE key_hash = $1`, keyHash)
	return scanAPIKeyRow(row)
}

func (q *Queries) TouchAPIKey(ctx context.Context, id uuid.UUID) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}

func (q *Queries) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]*models.APIKeyRow, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, user_id, name, key_hash, permissions, created_at, expires_at, last_used_at
		FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.APIKeyRow
	for rows.Next() {
		r, err := scanAPIKeyRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func scanAPIKeyRow(scanner interface{ Scan(...any) error }) (*models.APIKeyRow, error) {
	var k models.APIKeyRow
	err := scanner.Scan(
		&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.Permissions,
		&k.CreatedAt, &k.ExpiresAt, &k.LastUsedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("api key not found")
	}
	return &k, err
}

// ─── Sessions ─────────────────────────────────────────────────────────────────

type CreateSessionParams struct {
	UserID           uuid.UUID
	RefreshTokenHash string
	MFAVerified      bool
	IPAddress        string
	UserAgent        string
	ExpiresAt        time.Time
}

func (q *Queries) CreateSession(ctx context.Context, p CreateSessionParams) (*models.SessionRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, refresh_token_hash, mfa_verified, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, refresh_token_hash, mfa_verified, ip_address, user_agent, expires_at, created_at`,
		p.UserID, p.RefreshTokenHash, p.MFAVerified, p.IPAddress, p.UserAgent, p.ExpiresAt,
	)
	return scanSessionRow(row)
}

func (q *Queries) GetSessionByRefreshHash(ctx context.Context, hash string) (*models.SessionRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, user_id, refresh_token_hash, mfa_verified, ip_address, user_agent, expires_at, created_at
		FROM sessions WHERE refresh_token_hash = $1 AND expires_at > NOW()`, hash)
	return scanSessionRow(row)
}

func (q *Queries) DeleteSession(ctx context.Context, id uuid.UUID) error {
	_, err := q.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

func (q *Queries) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := q.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (q *Queries) DeleteExpiredSessions(ctx context.Context) error {
	_, err := q.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`)
	return err
}

func scanSessionRow(scanner interface{ Scan(...any) error }) (*models.SessionRow, error) {
	var s models.SessionRow
	err := scanner.Scan(
		&s.ID, &s.UserID, &s.RefreshTokenHash, &s.MFAVerified,
		&s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("session not found or expired")
	}
	return &s, err
}

// ─── MFA Devices ──────────────────────────────────────────────────────────────

type CreateMFADeviceParams struct {
	UserID    uuid.UUID
	Type      models.MFAType
	Name      string
	SecretEnc string
}

func (q *Queries) CreateMFADevice(ctx context.Context, p CreateMFADeviceParams) (*models.MFADeviceRow, error) {
	row := q.pool.QueryRow(ctx, `
		INSERT INTO mfa_devices (user_id, type, name, secret_enc)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, type, name, secret_enc, verified, created_at`,
		p.UserID, p.Type, p.Name, p.SecretEnc,
	)
	return scanMFADeviceRow(row)
}

func (q *Queries) GetMFADeviceByID(ctx context.Context, id uuid.UUID) (*models.MFADeviceRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, user_id, type, name, secret_enc, verified, created_at
		FROM mfa_devices WHERE id = $1`, id)
	return scanMFADeviceRow(row)
}

func (q *Queries) VerifyMFADevice(ctx context.Context, id uuid.UUID) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE mfa_devices SET verified = TRUE WHERE id = $1`, id)
	return err
}

func (q *Queries) GetVerifiedMFADevice(ctx context.Context, userID uuid.UUID) (*models.MFADeviceRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, user_id, type, name, secret_enc, verified, created_at
		FROM mfa_devices WHERE user_id = $1 AND verified = TRUE
		LIMIT 1`, userID)
	return scanMFADeviceRow(row)
}

func scanMFADeviceRow(scanner interface{ Scan(...any) error }) (*models.MFADeviceRow, error) {
	var d models.MFADeviceRow
	err := scanner.Scan(
		&d.ID, &d.UserID, &d.Type, &d.Name, &d.SecretEnc, &d.Verified, &d.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("mfa device not found")
	}
	return &d, err
}

// ─── Access Log ───────────────────────────────────────────────────────────────

type InsertAccessLogParams struct {
	UserID    *uuid.UUID
	Action    string
	Resource  string
	Status    models.AccessLogStatus
	IPAddress string
	UserAgent string
	Details   string
}

func (q *Queries) InsertAccessLog(ctx context.Context, p InsertAccessLogParams) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO access_log (user_id, action, resource, status, ip_address, user_agent, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.UserID, p.Action, p.Resource, p.Status, p.IPAddress, p.UserAgent, p.Details,
	)
	return err
}

type ListAccessLogParams struct {
	UserID *uuid.UUID
	Status *models.AccessLogStatus
	Page   int
	Limit  int
}

func (q *Queries) ListAccessLog(ctx context.Context, p ListAccessLogParams) ([]*models.AccessLog, error) {
	offset := (p.Page - 1) * p.Limit
	rows, err := q.pool.Query(ctx, `
		SELECT id, user_id, action, resource, status, ip_address, user_agent, details, created_at
		FROM access_log
		WHERE ($1::UUID IS NULL OR user_id = $1)
		  AND ($2::TEXT IS NULL OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`,
		p.UserID, p.Status, p.Limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.AccessLog
	for rows.Next() {
		var a models.AccessLog
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Action, &a.Resource, &a.Status,
			&a.IPAddress, &a.UserAgent, &a.Details, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, &a)
	}
	return result, rows.Err()
}

func (q *Queries) CountAccessLog(ctx context.Context, p ListAccessLogParams) (int64, error) {
	var count int64
	err := q.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM access_log
		WHERE ($1::UUID IS NULL OR user_id = $1)
		  AND ($2::TEXT IS NULL OR status = $2)`,
		p.UserID, p.Status,
	).Scan(&count)
	return count, err
}
