-- Referral Service Schema
-- Manages clinic-to-clinic patient referrals with full audit trail

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── referrals ───────────────────────────────────────────────────────────────
CREATE TABLE referrals (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id          UUID        NOT NULL,
    from_clinic_id      UUID        NOT NULL,
    to_clinic_id        UUID        NOT NULL,
    from_clinician_id   UUID        NOT NULL,
    to_clinician_id     UUID,
    reason_enc          TEXT        NOT NULL,
    patient_name_enc    TEXT        NOT NULL,
    urgency             TEXT        NOT NULL DEFAULT 'routine'
                            CHECK (urgency IN ('routine','semi_urgent','urgent','emergency')),
    status              TEXT        NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending','accepted','in_progress','completed','rejected','cancelled')),
    follow_up_date      TIMESTAMPTZ,
    follow_up_notes_enc TEXT,
    accepted_at         TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    rejected_at         TIMESTAMPTZ,
    rejection_reason_enc TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_referrals_patient   ON referrals(patient_id);
CREATE INDEX idx_referrals_from_clinic ON referrals(from_clinic_id);
CREATE INDEX idx_referrals_to_clinic ON referrals(to_clinic_id);
CREATE INDEX idx_referrals_status    ON referrals(status);
CREATE INDEX idx_referrals_urgency   ON referrals(urgency);

-- ─── referral_notes ──────────────────────────────────────────────────────────
-- Clinical notes attached to a referral. Immutable once written.
CREATE TABLE referral_notes (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id         UUID        NOT NULL REFERENCES referrals(id) ON DELETE RESTRICT,
    note_enc            TEXT        NOT NULL,
    created_by_user_id  UUID        NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_referral_notes_referral ON referral_notes(referral_id);

CREATE RULE referral_notes_no_update AS
    ON UPDATE TO referral_notes DO INSTEAD NOTHING;

CREATE RULE referral_notes_no_delete AS
    ON DELETE TO referral_notes DO INSTEAD NOTHING;

-- ─── referral_history ────────────────────────────────────────────────────────
-- Immutable audit trail of every status transition.
CREATE TABLE referral_history (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id         UUID        NOT NULL REFERENCES referrals(id) ON DELETE RESTRICT,
    status_before       TEXT,
    status_after        TEXT        NOT NULL,
    changed_by_user_id  UUID        NOT NULL,
    changed_by_role     TEXT        NOT NULL,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_referral_history_referral ON referral_history(referral_id);

CREATE RULE referral_history_no_update AS
    ON UPDATE TO referral_history DO INSTEAD NOTHING;

CREATE RULE referral_history_no_delete AS
    ON DELETE TO referral_history DO INSTEAD NOTHING;

-- ─── referral_audit_log ──────────────────────────────────────────────────────
-- Immutable record of every API access.
CREATE TABLE referral_audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id UUID,
    user_id     UUID        NOT NULL,
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    ip_address  TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_referral_audit_user     ON referral_audit_log(user_id);
CREATE INDEX idx_referral_audit_referral ON referral_audit_log(referral_id);

CREATE RULE referral_audit_log_no_update AS
    ON UPDATE TO referral_audit_log DO INSTEAD NOTHING;

CREATE RULE referral_audit_log_no_delete AS
    ON DELETE TO referral_audit_log DO INSTEAD NOTHING;
