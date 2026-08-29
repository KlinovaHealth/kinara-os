CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS lab_orders (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_ref    TEXT NOT NULL UNIQUE,
    patient_id   UUID NOT NULL,
    ordered_by   UUID NOT NULL,
    clinic_id    TEXT NOT NULL,
    test_code    TEXT NOT NULL,
    test_name    TEXT NOT NULL,
    priority     TEXT NOT NULL DEFAULT 'routine',
    status       TEXT NOT NULL DEFAULT 'pending',
    notes        TEXT,
    tenant_id    TEXT NOT NULL,
    ordered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS lab_results (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id        UUID NOT NULL REFERENCES lab_orders(id),
    patient_id      UUID NOT NULL,
    test_code       TEXT NOT NULL,
    result_value    TEXT NOT NULL,
    unit            TEXT,
    reference_range TEXT,
    flag            TEXT NOT NULL DEFAULT 'normal',
    analyzed_by     UUID NOT NULL,
    result_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_lab_orders_patient ON lab_orders(patient_id);
CREATE INDEX IF NOT EXISTS idx_lab_orders_tenant ON lab_orders(tenant_id);
CREATE INDEX IF NOT EXISTS idx_lab_results_order ON lab_results(order_id);

CREATE RULE no_update_results AS ON UPDATE TO lab_results DO INSTEAD NOTHING;
CREATE RULE no_delete_results AS ON DELETE TO lab_results DO INSTEAD NOTHING;
