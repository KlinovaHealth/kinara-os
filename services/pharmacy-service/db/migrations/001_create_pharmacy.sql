-- Pharmacy Service Schema
-- Prescription fulfillment, medication inventory, supply chain

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── medications ─────────────────────────────────────────────────────────────
CREATE TABLE medications (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT        NOT NULL,
    generic_name    TEXT        NOT NULL DEFAULT '',
    description     TEXT        NOT NULL DEFAULT '',
    unit_price      NUMERIC(12,4) NOT NULL DEFAULT 0,
    currency        TEXT        NOT NULL DEFAULT 'USD',
    stock_level     INTEGER     NOT NULL DEFAULT 0 CHECK (stock_level >= 0),
    reorder_point   INTEGER     NOT NULL DEFAULT 10,
    reorder_qty     INTEGER     NOT NULL DEFAULT 100,
    unit            TEXT        NOT NULL DEFAULT 'tablet',
    supplier_id     UUID,
    expiration_date TIMESTAMPTZ,
    batch_number    TEXT        NOT NULL DEFAULT '',
    requires_cold   BOOLEAN     NOT NULL DEFAULT FALSE,
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_medications_name       ON medications(name);
CREATE INDEX idx_medications_active     ON medications(is_active);
CREATE INDEX idx_medications_stock      ON medications(stock_level);
CREATE INDEX idx_medications_expiration ON medications(expiration_date);

-- ─── prescriptions ───────────────────────────────────────────────────────────
-- Linked to clinical-service prescription by clinical_id.
-- Patient data encrypted; dosage encrypted to protect treatment details.
CREATE TABLE prescriptions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    clinical_id     UUID        NOT NULL UNIQUE, -- FK to clinical-service
    patient_id      UUID        NOT NULL,
    clinic_id       UUID        NOT NULL,
    medication_id   UUID        NOT NULL REFERENCES medications(id),
    patient_name_enc TEXT       NOT NULL,
    dosage_enc      TEXT        NOT NULL,
    quantity        INTEGER     NOT NULL CHECK (quantity > 0),
    quantity_unit   TEXT        NOT NULL DEFAULT 'tablet',
    instructions    TEXT        NOT NULL DEFAULT '',
    status          TEXT        NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','dispensed','partial','cancelled','expired')),
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prescriptions_patient   ON prescriptions(patient_id);
CREATE INDEX idx_prescriptions_clinic    ON prescriptions(clinic_id);
CREATE INDEX idx_prescriptions_status    ON prescriptions(status);
CREATE INDEX idx_prescriptions_expires   ON prescriptions(expires_at);
CREATE INDEX idx_prescriptions_clinical  ON prescriptions(clinical_id);

-- ─── dispensing ──────────────────────────────────────────────────────────────
-- Immutable record each time medication is given to a patient.
CREATE TABLE dispensing (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    prescription_id     UUID        NOT NULL REFERENCES prescriptions(id),
    medication_id       UUID        NOT NULL REFERENCES medications(id),
    dispensed_by_user_id UUID       NOT NULL,
    quantity_dispensed  INTEGER     NOT NULL CHECK (quantity_dispensed > 0),
    batch_number        TEXT        NOT NULL DEFAULT '',
    cost_amount         NUMERIC(12,4) NOT NULL DEFAULT 0,
    currency            TEXT        NOT NULL DEFAULT 'USD',
    patient_cost_share  NUMERIC(12,4) NOT NULL DEFAULT 0,
    notes               TEXT        NOT NULL DEFAULT '',
    dispensed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dispensing_prescription ON dispensing(prescription_id);
CREATE INDEX idx_dispensing_medication   ON dispensing(medication_id);
CREATE INDEX idx_dispensing_dispensed_at ON dispensing(dispensed_at);

CREATE RULE dispensing_no_update AS
    ON UPDATE TO dispensing DO INSTEAD NOTHING;

CREATE RULE dispensing_no_delete AS
    ON DELETE TO dispensing DO INSTEAD NOTHING;

-- ─── supply_orders ────────────────────────────────────────────────────────────
CREATE TABLE supply_orders (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    supplier_id       UUID        NOT NULL,
    medication_id     UUID        NOT NULL REFERENCES medications(id),
    quantity_ordered  INTEGER     NOT NULL CHECK (quantity_ordered > 0),
    quantity_received INTEGER     NOT NULL DEFAULT 0,
    unit_cost         NUMERIC(12,4) NOT NULL DEFAULT 0,
    currency          TEXT        NOT NULL DEFAULT 'USD',
    status            TEXT        NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending','approved','shipped','received','cancelled')),
    ordered_by_id     UUID        NOT NULL,
    expected_at       TIMESTAMPTZ,
    received_at       TIMESTAMPTZ,
    notes             TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_medication ON supply_orders(medication_id);
CREATE INDEX idx_orders_status     ON supply_orders(status);
CREATE INDEX idx_orders_supplier   ON supply_orders(supplier_id);

-- ─── pharmacy_audit_log ───────────────────────────────────────────────────────
CREATE TABLE pharmacy_audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id   UUID,
    user_id     UUID        NOT NULL,
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    ip_address  TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pharmacy_audit_user   ON pharmacy_audit_log(user_id);
CREATE INDEX idx_pharmacy_audit_entity ON pharmacy_audit_log(entity_id);

CREATE RULE pharmacy_audit_log_no_update AS
    ON UPDATE TO pharmacy_audit_log DO INSTEAD NOTHING;

CREATE RULE pharmacy_audit_log_no_delete AS
    ON DELETE TO pharmacy_audit_log DO INSTEAD NOTHING;
