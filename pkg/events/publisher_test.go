package events_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"

	"github.com/klinova/kinara-os/pkg/events"
)

// TestPublish_NoClaims verifies that Publish returns ErrNoClaims when the context
// has no JWT claims. No DB is needed — the check happens before insert.
func TestPublish_NoClaims(t *testing.T) {
	pub := events.NewPublisher(nil, "test-service")
	_, err := pub.Publish(context.Background(), "test.event", json.RawMessage(`{}`))
	if !errors.Is(err, events.ErrNoClaims) {
		t.Fatalf("got %v, want ErrNoClaims", err)
	}
}

// TestPublish_NoForgeableScope verifies by method signature that Publish and
// PublishDerived accept no scope parameters. The only inputs are ctx, event type,
// and payload. Scope (tenant_id, entity_type, clinic_id, origin_user_id) is derived
// from ctx claims or copied from a parent Event — both paths are outside caller control.
//
// If someone adds a scope parameter to either function, this test breaks: it is the
// code-review canary for forgeability regressions.
func TestPublish_NoForgeableScope(t *testing.T) {
	pubType := reflect.TypeOf((*events.Publisher)(nil))

	publishMethod, ok := pubType.MethodByName("Publish")
	if !ok {
		t.Fatal("Publisher.Publish not found")
	}
	// Expect: func(*Publisher, context.Context, string, json.RawMessage) (*Event, error)
	// NumIn = 4 (receiver, ctx, eventType, payload)
	if n := publishMethod.Type.NumIn(); n != 4 {
		t.Errorf("Publish: got %d params, want 4 (receiver+ctx+type+payload) — scope param added?", n)
	}

	derivedMethod, ok := pubType.MethodByName("PublishDerived")
	if !ok {
		t.Fatal("Publisher.PublishDerived not found")
	}
	// Expect: func(*Publisher, context.Context, *Event, string, json.RawMessage) (*Event, error)
	// NumIn = 5 (receiver, ctx, parent, eventType, payload)
	if n := derivedMethod.Type.NumIn(); n != 5 {
		t.Errorf("PublishDerived: got %d params, want 5 (receiver+ctx+parent+type+payload) — scope param added?", n)
	}
}

// TestPublishDerived_CopiesScopeExactly inserts a real root event then derives from
// it, verifying that the derived event carries the parent's scope verbatim and that
// hop is incremented by exactly one. Requires EVENTS_TEST_DB_URL pointing at a live
// kinara_events database; skipped if unset.
func TestPublishDerived_CopiesScopeExactly(t *testing.T) {
	dbURL := os.Getenv("EVENTS_TEST_DB_URL")
	if dbURL == "" {
		t.Skip("EVENTS_TEST_DB_URL not set — skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	pub := events.NewPublisher(pool, "events-test")

	tenantID := uuid.New()
	clinicID := uuid.New()
	userID := uuid.New()
	claims := &pkgauth.Claims{
		UserID:     userID,
		EntityType: "klinova",
		TenantID:   tenantID,
		ClinicID:   &clinicID,
	}
	authCtx := pkgauth.InjectClaims(ctx, claims)

	parent, err := pub.Publish(authCtx, "test.root.event", json.RawMessage(`{"step":1}`))
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if parent.ID == uuid.Nil {
		t.Fatal("parent.ID is nil — DB did not return generated ID")
	}

	derived, err := pub.PublishDerived(authCtx, parent, "test.derived.event", json.RawMessage(`{"step":2}`))
	if err != nil {
		t.Fatalf("PublishDerived: %v", err)
	}

	if derived.TenantID != parent.TenantID {
		t.Errorf("TenantID: got %v, want %v", derived.TenantID, parent.TenantID)
	}
	if derived.EntityType != parent.EntityType {
		t.Errorf("EntityType: got %q, want %q", derived.EntityType, parent.EntityType)
	}
	switch {
	case derived.ClinicID == nil && parent.ClinicID != nil:
		t.Errorf("ClinicID: got nil, want %v", *parent.ClinicID)
	case derived.ClinicID != nil && parent.ClinicID == nil:
		t.Errorf("ClinicID: got %v, want nil", *derived.ClinicID)
	case derived.ClinicID != nil && *derived.ClinicID != *parent.ClinicID:
		t.Errorf("ClinicID: got %v, want %v", *derived.ClinicID, *parent.ClinicID)
	}
	if derived.Hop != parent.Hop+1 {
		t.Errorf("Hop: got %d, want %d", derived.Hop, parent.Hop+1)
	}
	if derived.OriginEventID == nil || *derived.OriginEventID != parent.ID {
		t.Errorf("OriginEventID: got %v, want %v", derived.OriginEventID, parent.ID)
	}
}

// TestPublishDerived_HopExceeded verifies that PublishDerived refuses to write when
// parent.Hop == MaxHop. No DB is needed — the check happens before insert.
func TestPublishDerived_HopExceeded(t *testing.T) {
	pub := events.NewPublisher(nil, "test-service")
	parent := &events.Event{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		EntityType: "klinova",
		Hop:        events.MaxHop,
	}
	_, err := pub.PublishDerived(context.Background(), parent, "test.event", json.RawMessage(`{}`))
	if !errors.Is(err, events.ErrHopLimitExceeded) {
		t.Fatalf("got %v, want ErrHopLimitExceeded", err)
	}
}
