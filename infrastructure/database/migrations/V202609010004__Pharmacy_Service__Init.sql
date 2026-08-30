-- Pharmacy Service — Drug inventory, prescription dispensing, and stock movements
-- Database: kinara_pharmacy
-- Manages medication catalog, incoming prescriptions, stock levels, and movement history

\c kinara_pharmacy;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS medications (
    id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT          NOT NULL,
    generic_name     TEXT,
    unit_price       NUMERIC(10,2) NOT NULL DEFAULT 0,
    currency         TEXT          NOT NULL DEFAULT 'XOF',
    stock_level      INT           NOT NULL DEFAULT 0 CHECK (stock_level >= 0),
    reorder_point    INT           NOT NULL DEFAULT 10,
    reorder_qty      INT           NOT NULL DEFAULT 50,
    unit             TEXT          NOT NULL DEFAULT 'tablet',
    supplier_id      UUID,
    expiration_date  TIMESTAMPTZ,
    batch_number     TEXT,
    requires_cold    BOOLEAN       NOT NULL DEFAULT false,
    is_active        BOOLEAN       NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  medications                  IS 'Drug catalog and inventory — stock levels updated via stock_movements';
COMMENT ON COLUMN medications.stock_level      IS 'Current on-hand quantity; enforced >= 0 to prevent phantom stock';
COMMENT ON COLUMN medications.reorder_point    IS 'Minimum stock level that triggers a reorder alert';
COMMENT ON COLUMN medications.requires_cold    IS 'True if cold-chain storage (2–8°C) is required';
COMMENT ON COLUMN medications.batch_number     IS 'Manufacturer batch/lot number for recall traceability';
COMMENT ON COLUMN medications.expiration_date  IS 'Earliest expiry date for current batch; alert when within 30 days';

CREATE INDEX IF NOT EXISTS idx_med_name        ON medications(name);
CREATE INDEX IF NOT EXISTS idx_med_stock       ON medications(stock_level);
CREATE INDEX IF NOT EXISTS idx_med_expiry      ON medications(expiration_date);

CREATE TABLE IF NOT EXISTS prescriptions (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id           UUID        NOT NULL,
    doctor_id            UUID        NOT NULL,
    consultation_id      UUID,
    medication_id        UUID        NOT NULL REFERENCES medications(id),
    dosage               TEXT        NOT NULL,
    frequency            TEXT        NOT NULL,
    duration_days        INT,
    status               TEXT        NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending','dispensed','partial','cancelled')),
    dispensed_at         TIMESTAMPTZ,
    dispensed_by         UUID,
    quantity_dispensed   INT,
    notes                TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  prescriptions                   IS 'Incoming prescriptions to be filled — sourced from clinical service';
COMMENT ON COLUMN prescriptions.medication_id     IS 'References medications(id) — ensures dispensed drug exists in catalog';
COMMENT ON COLUMN prescriptions.quantity_dispensed IS 'Actual quantity dispensed; may differ from prescribed if partial fill';
COMMENT ON COLUMN prescriptions.status            IS 'Dispensing status: pending, dispensed, partial, cancelled';

CREATE INDEX IF NOT EXISTS idx_rx_patient ON prescriptions(patient_id);
CREATE INDEX IF NOT EXISTS idx_rx_status  ON prescriptions(status);
CREATE INDEX IF NOT EXISTS idx_rx_med     ON prescriptions(medication_id);

CREATE TABLE IF NOT EXISTS stock_movements (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    medication_id  UUID        NOT NULL REFERENCES medications(id),
    movement_type  TEXT        NOT NULL
                   CHECK (movement_type IN ('in','out','adjustment','expiry')),
    quantity       INT         NOT NULL,
    reference_id   UUID,
    reason         TEXT,
    performed_by   UUID        NOT NULL,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  stock_movements               IS 'Full inventory movement ledger — source of truth for stock reconciliation';
COMMENT ON COLUMN stock_movements.movement_type IS 'Direction: in (receipt), out (dispensing), adjustment (count correction), expiry';
COMMENT ON COLUMN stock_movements.reference_id  IS 'FK to prescriptions.id or purchase_order.id depending on movement_type';

CREATE INDEX IF NOT EXISTS idx_stock_med     ON stock_movements(medication_id);
CREATE INDEX IF NOT EXISTS idx_stock_type    ON stock_movements(movement_type);
CREATE INDEX IF NOT EXISTS idx_stock_date    ON stock_movements(occurred_at DESC);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS pharmacy_audit_log (
    id             BIGSERIAL   PRIMARY KEY,
    entity_id      UUID        NOT NULL,
    action         TEXT        NOT NULL,  -- 'create','update','delete','read'
    actor_id       TEXT        NOT NULL,
    old_data       JSONB,
    new_data       JSONB,
    signature_hash TEXT,
    ip_address     INET,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE pharmacy_audit_log IS 'Append-only audit trail for pharmacy entities — includes stock adjustments';

CREATE RULE no_update_pharmacy_audit AS ON UPDATE TO pharmacy_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_pharmacy_audit AS ON DELETE TO pharmacy_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_pharmacy_audit_entity ON pharmacy_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_pharmacy_audit_actor  ON pharmacy_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS pharmacy_audit_log CASCADE;
-- DROP TABLE IF EXISTS stock_movements CASCADE;
-- DROP TABLE IF EXISTS prescriptions CASCADE;
-- DROP TABLE IF EXISTS medications CASCADE;
