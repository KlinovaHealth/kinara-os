CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS lab_orders (
    id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_ref    TEXT        NOT NULL UNIQUE,
    patient_id   UUID        NOT NULL,
    ordered_by   UUID        NOT NULL,
    clinic_id    TEXT        NOT NULL,
    test_code    TEXT        NOT NULL,
    test_name    TEXT        NOT NULL,
    priority     TEXT        NOT NULL DEFAULT 'routine',
    status       TEXT        NOT NULL DEFAULT 'pending',
    notes        TEXT,
    tenant_id    TEXT        NOT NULL,
    ordered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_lab_patient  ON lab_orders(patient_id);
CREATE INDEX IF NOT EXISTS idx_lab_clinic   ON lab_orders(clinic_id, ordered_at);

CREATE TABLE IF NOT EXISTS lab_results (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID          NOT NULL REFERENCES lab_orders(id),
    result_value NUMERIC(10,4) NOT NULL,
    unit         TEXT          NOT NULL,
    normal_low   NUMERIC(10,4),
    normal_high  NUMERIC(10,4),
    flag         TEXT          NOT NULL DEFAULT 'normal',
    notes        TEXT,
    recorded_by  UUID          NOT NULL,
    recorded_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS test_catalog (
    test_code  TEXT          PRIMARY KEY,
    test_name  TEXT          NOT NULL,
    normal_low NUMERIC(10,4),
    normal_high NUMERIC(10,4),
    unit       TEXT          NOT NULL
);

INSERT INTO test_catalog VALUES
    ('HGB', 'Hemoglobin',                  11.5, 17.5, 'g/dL'),
    ('WBC', 'White Blood Cell Count',       4.0,  11.0, 'x10³/µL'),
    ('PLT', 'Platelet Count',             150,   400,  'x10³/µL'),
    ('GLU', 'Blood Glucose (Fasting)',     70,   100,  'mg/dL'),
    ('CRE', 'Creatinine',                   0.6,   1.2, 'mg/dL'),
    ('MAL', 'Malaria RDT',                  0,     0,  'positive/negative')
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS lab_audit_log (
    id          BIGSERIAL   PRIMARY KEY,
    order_id    UUID        NOT NULL,
    action      TEXT        NOT NULL,
    actor_id    TEXT        NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_lab_audit AS ON UPDATE TO lab_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_lab_audit  AS ON DELETE TO lab_audit_log DO INSTEAD NOTHING;
