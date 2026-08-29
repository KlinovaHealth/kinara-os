package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klinova/kinara-os/sms-gateway/models"
)

type Queries struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Queries { return &Queries{pool: pool} }

func (q *Queries) SaveLog(ctx context.Context, l models.SMSLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO sms_logs (id,provider,direction,from_number,to_number,body,response,command,success,created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		l.ID, l.Provider, l.Direction, l.From, l.To, l.Body, l.Response, l.Command, l.Success, l.CreatedAt)
	return err
}
