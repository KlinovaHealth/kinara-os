package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/notification-service/models"
)

type Queries struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

// ─── Notifications ────────────────────────────────────────────────────────────

func (q *Queries) CreateNotification(ctx context.Context, row models.NotificationRow) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO notifications
			(id, user_id, type, channel, priority, message_enc, subject_enc, recipient_enc,
			 template_id, status, scheduled_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		row.ID, row.UserID, row.Type, row.Channel, row.Priority,
		row.MessageEnc, row.SubjectEnc, row.RecipientEnc,
		row.TemplateID, row.Status, row.ScheduledAt, row.CreatedAt, row.UpdatedAt,
	)
	return err
}

func (q *Queries) GetNotification(ctx context.Context, id uuid.UUID) (*models.NotificationRow, error) {
	row := q.pool.QueryRow(ctx, `
		SELECT id, user_id, type, channel, priority, message_enc, subject_enc, recipient_enc,
		       template_id, status, retry_count, scheduled_at, sent_at, delivered_at,
		       failure_reason, external_id, created_at, updated_at
		FROM notifications WHERE id = $1`, id)
	return scanNotificationRow(row)
}

type ListNotificationsParams struct {
	UserID   *uuid.UUID
	Type     *models.NotificationType
	Channel  *models.NotificationChannel
	Status   *models.NotificationStatus
	Page     int
	Limit    int
}

func (q *Queries) ListNotifications(ctx context.Context, p ListNotificationsParams) ([]models.NotificationRow, error) {
	where, args := buildNotificationWhere(p)
	offset := (p.Page - 1) * p.Limit
	n := len(args) + 1
	args = append(args, p.Limit, offset)
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, user_id, type, channel, priority, message_enc, subject_enc, recipient_enc,
		       template_id, status, retry_count, scheduled_at, sent_at, delivered_at,
		       failure_reason, external_id, created_at, updated_at
		FROM notifications %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, n, n+1), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.NotificationRow
	for rows.Next() {
		r, err := scanNotificationRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *r)
	}
	return result, rows.Err()
}

func (q *Queries) CountNotifications(ctx context.Context, p ListNotificationsParams) (int, error) {
	where, args := buildNotificationWhere(p)
	var total int
	err := q.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM notifications %s`, where), args...).Scan(&total)
	return total, err
}

func buildNotificationWhere(p ListNotificationsParams) (string, []interface{}) {
	where := "WHERE 1=1"
	var args []interface{}
	n := 1
	if p.UserID != nil {
		where += fmt.Sprintf(" AND user_id = $%d", n)
		args = append(args, *p.UserID)
		n++
	}
	if p.Type != nil {
		where += fmt.Sprintf(" AND type = $%d", n)
		args = append(args, *p.Type)
		n++
	}
	if p.Channel != nil {
		where += fmt.Sprintf(" AND channel = $%d", n)
		args = append(args, *p.Channel)
		n++
	}
	if p.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, *p.Status)
	}
	return where, args
}

func (q *Queries) UpdateNotificationStatus(ctx context.Context, id uuid.UUID, status models.NotificationStatus, sentAt, deliveredAt *time.Time, failureReason, externalID string, now time.Time) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE notifications SET
			status         = $1,
			sent_at        = COALESCE($2, sent_at),
			delivered_at   = COALESCE($3, delivered_at),
			failure_reason = CASE WHEN $4 != '' THEN $4 ELSE failure_reason END,
			external_id    = CASE WHEN $5 != '' THEN $5 ELSE external_id END,
			retry_count    = CASE WHEN $1 = 'failed' THEN retry_count + 1 ELSE retry_count END,
			updated_at     = $6
		WHERE id = $7`,
		status, sentAt, deliveredAt, failureReason, externalID, now, id,
	)
	return err
}

// GetPendingScheduled returns notifications scheduled to be sent by the given time.
func (q *Queries) GetPendingScheduled(ctx context.Context, before time.Time) ([]models.NotificationRow, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, user_id, type, channel, priority, message_enc, subject_enc, recipient_enc,
		       template_id, status, retry_count, scheduled_at, sent_at, delivered_at,
		       failure_reason, external_id, created_at, updated_at
		FROM notifications
		WHERE status = 'pending' AND scheduled_at IS NOT NULL AND scheduled_at <= $1
		ORDER BY
			CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'normal' THEN 3 ELSE 4 END,
			scheduled_at ASC
		LIMIT 100`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.NotificationRow
	for rows.Next() {
		r, err := scanNotificationRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *r)
	}
	return result, rows.Err()
}

// ─── Templates ────────────────────────────────────────────────────────────────

func (q *Queries) GetTemplate(ctx context.Context, id uuid.UUID) (*models.NotificationTemplate, error) {
	var t models.NotificationTemplate
	err := q.pool.QueryRow(ctx, `
		SELECT id, type, channel, name, subject_template, body_template, variables, language, is_active, created_at
		FROM notification_templates WHERE id = $1`, id,
	).Scan(&t.ID, &t.Type, &t.Channel, &t.Name, &t.SubjectTpl, &t.BodyTpl, &t.Variables, &t.Language, &t.IsActive, &t.CreatedAt)
	return &t, err
}

func (q *Queries) ListTemplates(ctx context.Context, notifType *models.NotificationType, channel *models.NotificationChannel) ([]models.NotificationTemplate, error) {
	where := "WHERE is_active = TRUE"
	var args []interface{}
	n := 1
	if notifType != nil {
		where += fmt.Sprintf(" AND type = $%d", n)
		args = append(args, *notifType)
		n++
	}
	if channel != nil {
		where += fmt.Sprintf(" AND channel = $%d", n)
		args = append(args, *channel)
	}
	rows, err := q.pool.Query(ctx,
		fmt.Sprintf(`SELECT id, type, channel, name, subject_template, body_template, variables, language, is_active, created_at
		             FROM notification_templates %s ORDER BY type, channel`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.NotificationTemplate
	for rows.Next() {
		var t models.NotificationTemplate
		if err := rows.Scan(&t.ID, &t.Type, &t.Channel, &t.Name, &t.SubjectTpl, &t.BodyTpl,
			&t.Variables, &t.Language, &t.IsActive, &t.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

// ─── User preferences ─────────────────────────────────────────────────────────

func (q *Queries) GetOrCreatePreferences(ctx context.Context, userID uuid.UUID) (*models.UserPreferences, error) {
	var p models.UserPreferences
	err := q.pool.QueryRow(ctx, `
		INSERT INTO user_preferences (id, user_id, updated_at)
		VALUES (gen_random_uuid(), $1, NOW())
		ON CONFLICT (user_id) DO UPDATE SET updated_at = user_preferences.updated_at
		RETURNING id, user_id, sms_enabled, push_enabled, whatsapp_enabled,
		          email_enabled, in_app_enabled, quiet_hours_start, quiet_hours_end,
		          timezone, language, updated_at`, userID,
	).Scan(&p.ID, &p.UserID, &p.SMSEnabled, &p.PushEnabled, &p.WhatsAppEnabled,
		&p.EmailEnabled, &p.InAppEnabled, &p.QuietHoursStart, &p.QuietHoursEnd,
		&p.TimeZone, &p.Language, &p.UpdatedAt)
	return &p, err
}

type UpdatePreferencesParams struct {
	UserID          uuid.UUID
	SMSEnabled      *bool
	PushEnabled     *bool
	WhatsAppEnabled *bool
	EmailEnabled    *bool
	InAppEnabled    *bool
	QuietHoursStart *string
	QuietHoursEnd   *string
	TimeZone        *string
	Language        *string
	Now             time.Time
}

func (q *Queries) UpdatePreferences(ctx context.Context, p UpdatePreferencesParams) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE user_preferences SET
			sms_enabled      = COALESCE($1, sms_enabled),
			push_enabled     = COALESCE($2, push_enabled),
			whatsapp_enabled = COALESCE($3, whatsapp_enabled),
			email_enabled    = COALESCE($4, email_enabled),
			in_app_enabled   = COALESCE($5, in_app_enabled),
			quiet_hours_start = COALESCE($6, quiet_hours_start),
			quiet_hours_end   = COALESCE($7, quiet_hours_end),
			timezone         = COALESCE($8, timezone),
			language         = COALESCE($9, language),
			updated_at       = $10
		WHERE user_id = $11`,
		p.SMSEnabled, p.PushEnabled, p.WhatsAppEnabled, p.EmailEnabled, p.InAppEnabled,
		p.QuietHoursStart, p.QuietHoursEnd, p.TimeZone, p.Language, p.Now, p.UserID,
	)
	return err
}

// ─── Audit log ────────────────────────────────────────────────────────────────

func (q *Queries) InsertAuditLog(ctx context.Context, log models.NotificationAuditLog) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO notification_audit_log (id, notification_id, user_id, action, resource, ip_address, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		log.ID, log.NotificationID, log.UserID, log.Action, log.Resource, log.IPAddress, log.CreatedAt,
	)
	return err
}

// ─── Row scanner ─────────────────────────────────────────────────────────────

type scannable interface {
	Scan(dest ...any) error
}

func scanNotificationRow(row scannable) (*models.NotificationRow, error) {
	var n models.NotificationRow
	err := row.Scan(
		&n.ID, &n.UserID, &n.Type, &n.Channel, &n.Priority,
		&n.MessageEnc, &n.SubjectEnc, &n.RecipientEnc,
		&n.TemplateID, &n.Status, &n.RetryCount, &n.ScheduledAt, &n.SentAt, &n.DeliveredAt,
		&n.FailureReason, &n.ExternalID, &n.CreatedAt, &n.UpdatedAt,
	)
	return &n, err
}
