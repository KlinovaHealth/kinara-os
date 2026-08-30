-- =============================================================================
-- Kinara OS — Documentation Service
-- Migration : V202609010036__Documentation_Service__Init.sql
-- Database  : kinara_documentation
-- Description: Initialises the Documentation Service schema: documents,
--              signatories, version history, and an immutable audit log.
-- =============================================================================

\c kinara_documentation;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- documents
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS documents (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    doc_ref          TEXT        UNIQUE NOT NULL,
    doc_type         TEXT        NOT NULL
                     CHECK (doc_type IN (
                         'bill_of_lading','airway_bill','invoice','packing_list',
                         'certificate_of_origin','phytosanitary',
                         'insurance_certificate','other'
                     )),
    title            TEXT        NOT NULL,
    file_url         TEXT,
    file_size_bytes  BIGINT,
    mime_type        TEXT,
    owner_id         UUID        NOT NULL,
    owner_type       TEXT
                     CHECK (owner_type IN (
                         'shipment','cargo','customs','trade_finance','vessel'
                     )),
    entity_id        UUID,
    status           TEXT        NOT NULL DEFAULT 'draft'
                     CHECK (status IN ('draft','submitted','approved','rejected','archived')),
    version          INT         NOT NULL DEFAULT 1,
    tenant_id        TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE documents IS
    'Central document registry for all Kinara OS maritime and logistics documents. '
    'Stores metadata and file references for bills of lading, airway bills, '
    'certificates of origin, phytosanitary certificates, and other trade documents. '
    'Versioning is managed through document_versions; signatories via document_signatories.';

-- ---------------------------------------------------------------------------
-- document_signatories
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS document_signatories (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id      UUID        NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    signatory_id     UUID        NOT NULL,
    signatory_name   TEXT,
    signatory_role   TEXT,
    signature_url    TEXT,
    signed_at        TIMESTAMPTZ,
    signature_hash   TEXT
);

COMMENT ON TABLE document_signatories IS
    'Records each party required to sign or endorse a document. '
    'Signature hashes provide cryptographic proof of the signed content at signing time.';

-- ---------------------------------------------------------------------------
-- document_versions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS document_versions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID        NOT NULL REFERENCES documents(id),
    version_number  INT         NOT NULL,
    file_url        TEXT,
    change_summary  TEXT,
    created_by      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE document_versions IS
    'Immutable version history for each document. Each revision produces a new row; '
    'the current version number is denormalised onto the parent documents row for fast lookup.';

-- ---------------------------------------------------------------------------
-- documentation_audit_log  (immutable)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS documentation_audit_log (
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

COMMENT ON TABLE documentation_audit_log IS
    'Immutable append-only audit trail for all Documentation Service write operations. '
    'UPDATE and DELETE are blocked by rules to preserve document chain-of-custody records.';

CREATE RULE no_update_documentation_audit AS ON UPDATE TO documentation_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_documentation_audit AS ON DELETE TO documentation_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_documentation_audit_entity
    ON documentation_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_documentation_audit_actor
    ON documentation_audit_log(actor_id, occurred_at);

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_documents_owner_type
    ON documents(owner_id, doc_type);

CREATE INDEX IF NOT EXISTS idx_documents_entity
    ON documents(entity_id);

CREATE INDEX IF NOT EXISTS idx_documents_status
    ON documents(status);

CREATE INDEX IF NOT EXISTS idx_document_signatories_document
    ON document_signatories(document_id);

-- =============================================================================
-- DOWN — rollback
-- =============================================================================
-- DROP TABLE IF EXISTS documentation_audit_log CASCADE;
-- DROP TABLE IF EXISTS document_versions CASCADE;
-- DROP TABLE IF EXISTS document_signatories CASCADE;
-- DROP TABLE IF EXISTS documents CASCADE;
