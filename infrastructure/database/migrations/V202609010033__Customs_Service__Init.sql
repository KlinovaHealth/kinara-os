-- =============================================================================
-- Kinara OS — Customs Service
-- Migration : V202609010033__Customs_Service__Init.sql
-- Database  : kinara_customs
-- Description: Initialises the Customs Service schema: declarations, supporting
--              documents, physical inspections, and an immutable audit log.
-- =============================================================================

\c kinara_customs;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- customs_declarations
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS customs_declarations (
    id                      UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    declaration_ref         TEXT           UNIQUE NOT NULL,
    declarant_id            UUID           NOT NULL,
    shipment_id             UUID,
    entry_type              TEXT
                            CHECK (entry_type IN ('import','export','transit','re-export')),
    port_of_entry           TEXT           NOT NULL,
    country_of_origin       TEXT,
    country_of_destination  TEXT,
    commodity_code          TEXT,
    commodity_description   TEXT,
    gross_weight_kg         NUMERIC(12,3),
    net_weight_kg           NUMERIC(12,3),
    declared_value          NUMERIC(14,4)  NOT NULL,
    currency                TEXT           NOT NULL DEFAULT 'USD',
    duty_rate_pct           NUMERIC(5,2)   NOT NULL DEFAULT 0,
    duty_amount             NUMERIC(14,4)  NOT NULL DEFAULT 0,
    vat_amount              NUMERIC(14,4)  NOT NULL DEFAULT 0,
    total_taxes             NUMERIC(14,4)  NOT NULL DEFAULT 0,
    status                  TEXT           NOT NULL DEFAULT 'pending'
                            CHECK (status IN (
                                'pending','under_review','approved',
                                'rejected','queried','cleared'
                            )),
    submitted_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    cleared_at              TIMESTAMPTZ,
    reviewed_by             UUID,
    tenant_id               TEXT,
    created_at              TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE customs_declarations IS
    'Formal customs entry declarations for goods crossing borders tracked by Kinara OS. '
    'Records declared value, duty calculations, entry type, and clearance status for '
    'compliance with African Union customs regulations.';

-- ---------------------------------------------------------------------------
-- customs_documents
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS customs_documents (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    declaration_id  UUID        NOT NULL REFERENCES customs_declarations(id) ON DELETE CASCADE,
    doc_type        TEXT
                    CHECK (doc_type IN (
                        'bill_of_lading','invoice','packing_list','certificate_of_origin',
                        'phytosanitary','veterinary','dangerous_goods'
                    )),
    file_url        TEXT        NOT NULL,
    doc_number      TEXT,
    doc_date        DATE,
    uploaded_by     UUID,
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE customs_documents IS
    'Supporting documents attached to a customs declaration. '
    'Covers all document types required by customs authorities including bills of lading, '
    'phytosanitary certificates, and dangerous goods declarations.';

-- ---------------------------------------------------------------------------
-- customs_inspections
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS customs_inspections (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    declaration_id  UUID        NOT NULL REFERENCES customs_declarations(id),
    inspector_id    UUID,
    inspection_type TEXT
                    CHECK (inspection_type IN ('documentary','physical','x-ray','scanning')),
    outcome         TEXT
                    CHECK (outcome IN ('pass','fail','query')),
    notes           TEXT,
    inspected_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE customs_inspections IS
    'Inspection records tied to a customs declaration. '
    'Supports documentary, physical, X-ray, and scanning inspection types with '
    'structured outcome recording for regulatory audit trails.';

-- ---------------------------------------------------------------------------
-- customs_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS customs_audit_log (
    id              BIGSERIAL   PRIMARY KEY,
    entity_id       UUID,
    action          TEXT        NOT NULL,
    actor_id        TEXT        NOT NULL,
    old_data        JSONB,
    new_data        JSONB,
    signature_hash  TEXT,
    ip_address      INET,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE customs_audit_log IS
    'Immutable append-only audit trail for all Customs Service write operations. '
    'UPDATE and DELETE are blocked by rules to meet regulatory retention requirements.';

CREATE RULE no_update_customs_audit AS ON UPDATE TO customs_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_customs_audit AS ON DELETE TO customs_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_customs_audit_entity
    ON customs_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_customs_audit_actor
    ON customs_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_customs_decl_declarant
    ON customs_declarations(declarant_id);

CREATE INDEX IF NOT EXISTS idx_customs_decl_status_submitted
    ON customs_declarations(status, submitted_at);

CREATE INDEX IF NOT EXISTS idx_customs_decl_port
    ON customs_declarations(port_of_entry);

CREATE INDEX IF NOT EXISTS idx_customs_docs_declaration
    ON customs_documents(declaration_id);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS customs_audit_log CASCADE;
-- DROP TABLE IF EXISTS customs_inspections CASCADE;
-- DROP TABLE IF EXISTS customs_documents CASCADE;
-- DROP TABLE IF EXISTS customs_declarations CASCADE;
