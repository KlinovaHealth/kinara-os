package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/klinova/kinara-os/offline-sync-service/db"
)

// DBQuerier is the subset of db.Queries methods used by handlers.
// Defined here to allow test doubles without a live database.
type DBQuerier interface {
	GetDeviceStatus(ctx context.Context, deviceID uuid.UUID) (*db.DeviceStatus, error)
	PullPatients(ctx context.Context, clinicID uuid.UUID) ([]db.PatientRecord, error)
	PatientInClinic(ctx context.Context, patientID, clinicID uuid.UUID) (bool, error)
	PushExists(ctx context.Context, deviceID uuid.UUID, key string) (bool, error)
	InsertSyncRecord(ctx context.Context, rec db.SyncRecord) error
	MarkApplied(ctx context.Context, id uuid.UUID, now time.Time) error
	MarkRejected(ctx context.Context, id uuid.UUID, reason string, now time.Time) error
	GetSyncStatus(ctx context.Context, deviceID uuid.UUID) (pending, applied, rejected int, err error)
	UpdateLastSeen(ctx context.Context, deviceID uuid.UUID, now time.Time) error
}
