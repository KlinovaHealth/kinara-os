CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE trade_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_ref VARCHAR(20) NOT NULL UNIQUE,
    document_type VARCHAR(50) NOT NULL,
    shipper_name VARCHAR(200) NOT NULL,
    consignee_name VARCHAR(200) NOT NULL,
    booking_ref VARCHAR(30),
    manifest_ref VARCHAR(30),
    issuing_country VARCHAR(3) NOT NULL,
    issuing_authority VARCHAR(200) NOT NULL,
    goods_description TEXT NOT NULL,
    value NUMERIC(14,2) NOT NULL DEFAULT 0,
    currency VARCHAR(5) NOT NULL DEFAULT 'USD',
    weight_kg NUMERIC(10,2) NOT NULL DEFAULT 0,
    net_weight_kg NUMERIC(10,2) NOT NULL DEFAULT 0,
    packages INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    issued_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    file_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE document_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id TEXT NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_doc_audit AS ON UPDATE TO document_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_doc_audit AS ON DELETE TO document_audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_docs_type ON trade_documents(document_type);
CREATE INDEX idx_docs_status ON trade_documents(status);
CREATE INDEX idx_docs_booking_ref ON trade_documents(booking_ref);
