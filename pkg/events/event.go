package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event is the in-memory representation of a row in the events table.
// It is returned by Publish and PublishDerived after the row is written.
type Event struct {
	ID            uuid.UUID       `db:"id"`
	Type          string          `db:"type"`
	TenantID      uuid.UUID       `db:"tenant_id"`
	EntityType    string          `db:"entity_type"`
	ClinicID      *uuid.UUID      `db:"clinic_id"`
	OriginUserID  *uuid.UUID      `db:"origin_user_id"`
	OriginEventID *uuid.UUID      `db:"origin_event_id"`
	Hop           int             `db:"hop"`
	Payload       json.RawMessage `db:"payload"`
	PublishedBy   string          `db:"published_by"`
	CreatedAt     time.Time       `db:"created_at"`
}
