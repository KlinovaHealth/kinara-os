-- Lab Service — Test orders, results, reference catalog, and critical flag detection
-- Database: kinara_lab
-- Manages the full lab workflow: order → processing → result entry with normal range flagging

\c kinara_lab;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS test_catalog (
    test_code   TEXT          PRIMARY KEY,
    test_name   TEXT          NOT NULL,
    normal_low  NUMERIC(10,4),
    normal_high NUMERIC(10,4),
    unit        TEXT
);

COMMENT ON TABLE  test_catalog            IS 'Reference table of lab tests with standard normal ranges — used for automatic flag assignment';
COMMENT ON COLUMN test_catalog.test_code  IS 'Short mnemonic code, e.g. HGB, WBC, GLU — primary key';
COMMENT ON COLUMN test_catalog.normal_low IS 'Lower bound of normal range (inclusive); null if no lower bound';
COMMENT ON COLUMN test_catalog.normal_high IS 'Upper bound of normal range (inclusive); null if no upper bound';
COMMENT ON COLUMN test_catalog.unit        IS 'Measurement unit, e.g. g/dL, 10^3/µL, mg/dL';

INSERT INTO test_catalog (test_code, test_name, normal_low, normal_high, unit) VALUES
    ('HGB', 'Hemoglobin',           12.0,  17.5,  'g/dL'),
    ('WBC', 'White Blood Cells',    4.5,   11.0,  '10^3/µL'),
    ('PLT', 'Platelets',            150.0, 400.0, '10^3/µL'),
    ('GLU', 'Fasting Blood Glucose',70.0,  99.0,  'mg/dL'),
    ('CRE', 'Creatinine',           0.6,   1.2,   'mg/dL'),
    ('MAL', 'Malaria RDT',          NULL,  NULL,  'qualitative')
ON CONFLICT (test_code) DO NOTHING;

CREATE TABLE IF NOT EXISTS lab_orders (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_ref     TEXT        NOT NULL UNIQUE,
    patient_id    UUID        NOT NULL,
    ordered_by    UUID        NOT NULL,
    clinic_id     TEXT        NOT NULL,
    test_code     TEXT        NOT NULL REFERENCES test_catalog(test_code),
    test_name     TEXT        NOT NULL,
    priority      TEXT        NOT NULL DEFAULT 'routine'
                  CHECK (priority IN ('stat','urgent','routine')),
    status        TEXT        NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','processing','completed','failed')),
    notes         TEXT,
    tenant_id     TEXT        NOT NULL,
    ordered_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ
);

COMMENT ON TABLE  lab_orders            IS 'Lab test orders — one order per test per patient encounter';
COMMENT ON COLUMN lab_orders.order_ref  IS 'Human-readable unique reference, e.g. LAB-20260901-0001';
COMMENT ON COLUMN lab_orders.priority   IS 'Turnaround priority: stat (< 1 h), urgent (< 4 h), routine (next cycle)';
COMMENT ON COLUMN lab_orders.status     IS 'Workflow state: pending → processing → completed | failed';

CREATE INDEX IF NOT EXISTS idx_lab_patient   ON lab_orders(patient_id);
CREATE INDEX IF NOT EXISTS idx_lab_clinic    ON lab_orders(clinic_id);
CREATE INDEX IF NOT EXISTS idx_lab_status    ON lab_orders(status);
CREATE INDEX IF NOT EXISTS idx_lab_priority  ON lab_orders(priority) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_lab_ordered   ON lab_orders(ordered_at DESC);

CREATE TABLE IF NOT EXISTS lab_results (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID          NOT NULL REFERENCES lab_orders(id),
    result_value  NUMERIC(10,4),
    unit          TEXT,
    normal_low    NUMERIC(10,4),
    normal_high   NUMERIC(10,4),
    flag          TEXT          NOT NULL DEFAULT 'normal'
                  CHECK (flag IN ('normal','abnormal','critical')),
    notes         TEXT,
    recorded_by   UUID          NOT NULL,
    recorded_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  lab_results              IS 'Lab test results — linked 1:1 to lab_orders; stores denormalized normal ranges at time of result';
COMMENT ON COLUMN lab_results.result_value IS 'Numeric result; null for qualitative tests (e.g. MAL RDT — capture in notes)';
COMMENT ON COLUMN lab_results.flag         IS 'Automated flag: normal (within range), abnormal (outside range), critical (life-threatening)';
COMMENT ON COLUMN lab_results.normal_low   IS 'Normal range lower bound at time of recording — denormalized for historical accuracy';
COMMENT ON COLUMN lab_results.normal_high  IS 'Normal range upper bound at time of recording — denormalized for historical accuracy';

CREATE INDEX IF NOT EXISTS idx_result_order ON lab_results(order_id);
CREATE INDEX IF NOT EXISTS idx_result_flag  ON lab_results(flag) WHERE flag IN ('abnormal','critical');

-- Immutable audit log
CREATE TABLE IF NOT EXISTS lab_audit_log (
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

COMMENT ON TABLE lab_audit_log IS 'Append-only audit trail for lab orders and results — critical for clinical liability';

CREATE RULE no_update_lab_audit AS ON UPDATE TO lab_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_lab_audit AS ON DELETE TO lab_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_lab_audit_entity ON lab_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_lab_audit_actor  ON lab_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS lab_audit_log CASCADE;
-- DROP TABLE IF EXISTS lab_results CASCADE;
-- DROP TABLE IF EXISTS lab_orders CASCADE;
-- DROP TABLE IF EXISTS test_catalog CASCADE;
