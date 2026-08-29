CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS immunization_records (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    record_ref       TEXT NOT NULL UNIQUE,
    patient_id       UUID NOT NULL,
    vaccine_code     TEXT NOT NULL,
    vaccine_name     TEXT NOT NULL,
    dose_number      INT NOT NULL DEFAULT 1,
    administered_by  UUID NOT NULL,
    administered_at  TIMESTAMPTZ NOT NULL,
    lot_number       TEXT NOT NULL,
    expiry_date      DATE,
    site_of_injection TEXT,
    clinic_id        TEXT NOT NULL,
    next_dose_date   DATE,
    status           TEXT NOT NULL DEFAULT 'administered',
    tenant_id        TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_immunizations_patient ON immunization_records(patient_id);
CREATE INDEX IF NOT EXISTS idx_immunizations_vaccine ON immunization_records(vaccine_code);
CREATE INDEX IF NOT EXISTS idx_immunizations_tenant ON immunization_records(tenant_id);

CREATE RULE no_delete_immunization AS ON DELETE TO immunization_records DO INSTEAD NOTHING;
