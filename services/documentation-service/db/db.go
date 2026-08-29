package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/documentation-service/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) CreateDocument(ctx context.Context, d models.TradeDocument) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO trade_documents (id,document_ref,document_type,shipper_name,consignee_name,booking_ref,manifest_ref,issuing_country,issuing_authority,goods_description,value,currency,weight_kg,net_weight_kg,packages,status,expires_at,created_at,updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		d.ID, d.DocumentRef, d.DocType, d.ShipperName, d.ConsigneeName, d.BookingRef, d.ManifestRef,
		d.IssuingCountry, d.IssuingAuthority, d.GoodsDescription, d.Value, d.Currency,
		d.WeightKg, d.NetWeightKg, d.Packages, d.Status, d.ExpiresAt, d.CreatedAt, d.UpdatedAt)
	return err
}

func (q *Queries) GetDocument(ctx context.Context, id uuid.UUID) (*models.TradeDocument, error) {
	d := &models.TradeDocument{}
	err := q.pool.QueryRow(ctx,
		`SELECT id,document_ref,document_type,shipper_name,consignee_name,booking_ref,manifest_ref,issuing_country,issuing_authority,goods_description,value,currency,weight_kg,net_weight_kg,packages,status,issued_at,expires_at,file_url,created_at,updated_at
         FROM trade_documents WHERE id=$1`, id).
		Scan(&d.ID, &d.DocumentRef, &d.DocType, &d.ShipperName, &d.ConsigneeName, &d.BookingRef, &d.ManifestRef,
			&d.IssuingCountry, &d.IssuingAuthority, &d.GoodsDescription, &d.Value, &d.Currency,
			&d.WeightKg, &d.NetWeightKg, &d.Packages, &d.Status, &d.IssuedAt, &d.ExpiresAt, &d.FileURL, &d.CreatedAt, &d.UpdatedAt)
	if err != nil { return nil, err }
	return d, nil
}

func (q *Queries) ListDocuments(ctx context.Context, docType *models.DocType, bookingRef *string) ([]models.TradeDocument, error) {
	query := `SELECT id,document_ref,document_type,shipper_name,consignee_name,booking_ref,manifest_ref,issuing_country,issuing_authority,goods_description,value,currency,weight_kg,net_weight_kg,packages,status,issued_at,expires_at,file_url,created_at,updated_at FROM trade_documents WHERE 1=1`
	args := []interface{}{}
	i := 1
	if docType != nil { query += " AND document_type=$1"; args = append(args, *docType); i++ }
	if bookingRef != nil {
		if i == 1 { query += " AND booking_ref=$1" } else { query += " AND booking_ref=$2" }
		args = append(args, *bookingRef)
	}
	query += " ORDER BY created_at DESC LIMIT 100"
	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var docs []models.TradeDocument
	for rows.Next() {
		var d models.TradeDocument
		if err := rows.Scan(&d.ID, &d.DocumentRef, &d.DocType, &d.ShipperName, &d.ConsigneeName, &d.BookingRef, &d.ManifestRef,
			&d.IssuingCountry, &d.IssuingAuthority, &d.GoodsDescription, &d.Value, &d.Currency,
			&d.WeightKg, &d.NetWeightKg, &d.Packages, &d.Status, &d.IssuedAt, &d.ExpiresAt, &d.FileURL, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func (q *Queries) IssueDocument(ctx context.Context, id uuid.UUID, fileURL string, now time.Time) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE trade_documents SET status='issued',issued_at=COALESCE(issued_at,$1),file_url=COALESCE(NULLIF($2,''),file_url),updated_at=$3 WHERE id=$4`,
		now, fileURL, now, id)
	return err
}

func (q *Queries) RevokeDocument(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := q.pool.Exec(ctx, `UPDATE trade_documents SET status='revoked',updated_at=$1 WHERE id=$2`, now, id)
	return err
}

func (q *Queries) InsertAuditLog(ctx context.Context, l models.DocumentAuditLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO document_audit_log (id,actor_id,action,entity_type,entity_id,created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		l.ID, l.ActorID, l.Action, l.EntityType, l.EntityID, l.CreatedAt)
	return err
}
