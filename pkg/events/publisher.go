// Package events provides the Kinara OS event bus publish path.
//
// Scope (tenant_id, entity_type, clinic_id, origin_user_id) is always derived
// from the JWT claims injected into the context by the service's JWT middleware,
// or copied verbatim from a parent Event. Callers have no parameter that accepts
// these fields — forgery is structurally impossible at the call site.
//
// Usage:
//
//	pub := events.NewPublisher(pool, "health-worker-service")
//	ev, err := pub.Publish(ctx, "health.visit.completed", payload)
//	derived, err := pub.PublishDerived(ctx, ev, "health.alert.triggered", alertPayload)
package events

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
)

// Publisher writes events to the kinara_events database.
// Construct one per service at startup via NewPublisher.
type Publisher struct {
	pool        *pgxpool.Pool
	serviceName string
}

// NewPublisher returns a Publisher backed by pool.
// serviceName is written to the published_by column on every event.
func NewPublisher(pool *pgxpool.Pool, serviceName string) *Publisher {
	return &Publisher{pool: pool, serviceName: serviceName}
}

// Publish inserts a root event (hop=0) whose scope is derived entirely from the
// JWT claims in ctx. Returns ErrNoClaims if ctx has no claims.
//
// The caller supplies only the event type and payload. tenant_id, entity_type,
// clinic_id, and origin_user_id are taken from the validated claims and cannot
// be overridden by the caller.
func (p *Publisher) Publish(ctx context.Context, eventType string, payload json.RawMessage) (*Event, error) {
	claims := pkgauth.ClaimsFromContext(ctx)
	if claims == nil {
		return nil, ErrNoClaims
	}

	// Lift origin_user_id from claims. UserID is a value type in Claims, so take
	// its address only after confirming claims is non-nil.
	uid := claims.UserID
	e := &Event{
		Type:         eventType,
		TenantID:     claims.TenantID,
		EntityType:   claims.EntityType,
		ClinicID:     claims.ClinicID, // nil for non-device sessions
		OriginUserID: &uid,
		Hop:          0,
		Payload:      payload,
		PublishedBy:  p.serviceName,
	}

	return p.insert(ctx, e)
}

// PublishDerived inserts a derived event whose scope is copied verbatim from
// parent and whose hop is parent.Hop+1. Returns ErrHopLimitExceeded if
// parent.Hop == MaxHop.
//
// Scope (tenant_id, entity_type, clinic_id, origin_user_id) is inherited from
// the parent event — not from the context — so derivation preserves the original
// scope boundary even when called from a background goroutine without user claims.
func (p *Publisher) PublishDerived(ctx context.Context, parent *Event, eventType string, payload json.RawMessage) (*Event, error) {
	if parent.Hop >= MaxHop {
		return nil, ErrHopLimitExceeded
	}

	parentID := parent.ID
	e := &Event{
		Type:          eventType,
		TenantID:      parent.TenantID,   // copied exactly
		EntityType:    parent.EntityType, // copied exactly
		ClinicID:      parent.ClinicID,   // copied exactly
		OriginUserID:  parent.OriginUserID,
		OriginEventID: &parentID,
		Hop:           parent.Hop + 1,
		Payload:       payload,
		PublishedBy:   p.serviceName,
	}

	return p.insert(ctx, e)
}

// insert writes e to the events table and populates e.ID and e.CreatedAt from
// the RETURNING clause. It is the only path that touches the database.
func (p *Publisher) insert(ctx context.Context, e *Event) (*Event, error) {
	err := p.pool.QueryRow(ctx, `
		INSERT INTO events
		    (type, tenant_id, entity_type, clinic_id, origin_user_id, origin_event_id, hop, payload, published_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`,
		e.Type,
		e.TenantID,
		e.EntityType,
		e.ClinicID,
		e.OriginUserID,
		e.OriginEventID,
		e.Hop,
		e.Payload,
		e.PublishedBy,
	).Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}
