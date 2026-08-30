-- Referral Service — Clinic-to-clinic patient referrals and document attachments
-- Database: kinara_referral
-- Manages the full referral lifecycle including urgency triage, status tracking, and attached documents

\c kinara_referral;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS referrals (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id            UUID        NOT NULL,
    from_clinic_id        UUID        NOT NULL,
    to_clinic_id          UUID        NOT NULL,
    from_clinician_id     UUID        NOT NULL,
    to_clinician_id       UUID,
    reason_enc            TEXT        NOT NULL,
    patient_name_enc      TEXT        NOT NULL,
    urgency               TEXT        NOT NULL DEFAULT 'routine'
                          CHECK (urgency IN ('routine','semi_urgent','urgent','emergency')),
    status                TEXT        NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending','accepted','in_progress','completed','rejected','cancelled')),
    follow_up_date        DATE,
    accepted_at           TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    rejected_at           TIMESTAMPTZ,
    rejection_reason_enc  TEXT,
    tenant_id             TEXT        NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  referrals                     IS 'Cross-facility patient referrals — tracks full lifecycle from request to completion';
COMMENT ON COLUMN referrals.reason_enc          IS 'Clinical reason for referral, AES-256-GCM encrypted';
COMMENT ON COLUMN referrals.patient_name_enc    IS 'Patient name copy for receiving facility, encrypted — always use patient_id as authoritative key';
COMMENT ON COLUMN referrals.urgency             IS 'Triage level: routine (days), semi_urgent (hours), urgent (< 2 h), emergency (immediate)';
COMMENT ON COLUMN referrals.rejection_reason_enc IS 'Reason provided by receiving facility if referral was rejected, encrypted';
COMMENT ON COLUMN referrals.to_clinician_id     IS 'Assigned receiving clinician — may be null until accepted';

CREATE INDEX IF NOT EXISTS idx_ref_patient       ON referrals(patient_id);
CREATE INDEX IF NOT EXISTS idx_ref_from_clinic   ON referrals(from_clinic_id);
CREATE INDEX IF NOT EXISTS idx_ref_to_clinic     ON referrals(to_clinic_id);
CREATE INDEX IF NOT EXISTS idx_ref_status        ON referrals(status);
CREATE INDEX IF NOT EXISTS idx_ref_urgency       ON referrals(urgency);
CREATE INDEX IF NOT EXISTS idx_ref_created       ON referrals(created_at DESC);

CREATE TABLE IF NOT EXISTS referral_documents (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id  UUID        NOT NULL REFERENCES referrals(id),
    doc_type     TEXT        NOT NULL,
    file_url     TEXT        NOT NULL,
    uploaded_by  UUID        NOT NULL,
    uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  referral_documents          IS 'Supporting documents attached to a referral — lab results, imaging reports, letters';
COMMENT ON COLUMN referral_documents.doc_type IS 'Document category: lab_result, imaging, letter, discharge_summary, other';
COMMENT ON COLUMN referral_documents.file_url IS 'Signed storage URL — pre-signed with short TTL, not for long-term storage';

CREATE INDEX IF NOT EXISTS idx_refdoc_referral ON referral_documents(referral_id);

-- Immutable audit log
CREATE TABLE IF NOT EXISTS referral_audit_log (
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

COMMENT ON TABLE referral_audit_log IS 'Append-only audit trail for referrals and attached documents';

CREATE RULE no_update_referral_audit AS ON UPDATE TO referral_audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_referral_audit AS ON DELETE TO referral_audit_log DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_referral_audit_entity ON referral_audit_log(entity_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_referral_audit_actor  ON referral_audit_log(actor_id,  occurred_at);

-- DOWN (rollback):
-- DROP TABLE IF EXISTS referral_audit_log CASCADE;
-- DROP TABLE IF EXISTS referral_documents CASCADE;
-- DROP TABLE IF EXISTS referrals CASCADE;
