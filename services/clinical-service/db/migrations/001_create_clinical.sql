-- Kinara OS — Clinical Service schema
-- All PHI fields encrypted (AES-256-GCM) before storage.
-- Audit log is immutable via PostgreSQL rules.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── Consultations ────────────────────────────────────────────────────────────

CREATE TABLE consultations (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id           UUID         NOT NULL,
    doctor_id            UUID         NOT NULL,
    status               TEXT         NOT NULL DEFAULT 'scheduled'
                                      CHECK (status IN ('scheduled','in_progress','completed','cancelled','no_show')),
    consultation_type    TEXT         NOT NULL
                                      CHECK (consultation_type IN ('video','audio','chat','in_person','whatsapp')),
    chief_complaint_enc  TEXT         NOT NULL,
    scheduled_at         TIMESTAMPTZ,
    started_at           TIMESTAMPTZ,
    ended_at             TIMESTAMPTZ,
    country              TEXT         NOT NULL,
    region               TEXT,
    facility_id          UUID,
    created_by           UUID         NOT NULL,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consultations_patient_id  ON consultations(patient_id);
CREATE INDEX idx_consultations_doctor_id   ON consultations(doctor_id);
CREATE INDEX idx_consultations_status      ON consultations(status);
CREATE INDEX idx_consultations_country     ON consultations(country);
CREATE INDEX idx_consultations_scheduled   ON consultations(scheduled_at);

-- ─── Diagnoses ────────────────────────────────────────────────────────────────

CREATE TABLE diagnoses (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id     UUID         NOT NULL REFERENCES consultations(id),
    patient_id          UUID         NOT NULL,
    doctor_id           UUID         NOT NULL,
    icd10_code          TEXT         NOT NULL,
    icd10_description   TEXT         NOT NULL,
    clinical_notes_enc  TEXT         NOT NULL,
    severity            TEXT         NOT NULL
                                     CHECK (severity IN ('mild','moderate','severe','critical')),
    is_primary          BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_diagnoses_consultation_id ON diagnoses(consultation_id);
CREATE INDEX idx_diagnoses_patient_id      ON diagnoses(patient_id);
CREATE INDEX idx_diagnoses_icd10_code      ON diagnoses(icd10_code);

-- ─── Treatments ───────────────────────────────────────────────────────────────

CREATE TABLE treatments (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id   UUID         NOT NULL REFERENCES consultations(id),
    patient_id        UUID         NOT NULL,
    doctor_id         UUID         NOT NULL,
    treatment_type    TEXT         NOT NULL
                                   CHECK (treatment_type IN ('medication','procedure','referral','lifestyle','monitoring')),
    instructions_enc  TEXT         NOT NULL,
    duration_days     INT          NOT NULL DEFAULT 0,
    follow_up_date    TIMESTAMPTZ,
    status            TEXT         NOT NULL DEFAULT 'active'
                                   CHECK (status IN ('active','completed','discontinued')),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_treatments_consultation_id ON treatments(consultation_id);
CREATE INDEX idx_treatments_patient_id      ON treatments(patient_id);
CREATE INDEX idx_treatments_status          ON treatments(status);

-- ─── Clinical Notes ───────────────────────────────────────────────────────────
-- Notes are write-once; the DELETE rule enforces immutability.

CREATE TABLE clinical_notes (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id  UUID         NOT NULL REFERENCES consultations(id),
    patient_id       UUID         NOT NULL,
    author_id        UUID         NOT NULL,
    note_type        TEXT         NOT NULL
                                  CHECK (note_type IN ('subjective','objective','assessment','plan','general')),
    content_enc      TEXT         NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clinical_notes_consultation_id ON clinical_notes(consultation_id);
CREATE INDEX idx_clinical_notes_patient_id      ON clinical_notes(patient_id);
CREATE INDEX idx_clinical_notes_author_id       ON clinical_notes(author_id);

CREATE RULE notes_no_update AS ON UPDATE TO clinical_notes DO INSTEAD NOTHING;
CREATE RULE notes_no_delete AS ON DELETE TO clinical_notes DO INSTEAD NOTHING;

-- ─── Prescriptions ────────────────────────────────────────────────────────────

CREATE TABLE prescriptions (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id  UUID         NOT NULL REFERENCES consultations(id),
    patient_id       UUID         NOT NULL,
    doctor_id        UUID         NOT NULL,
    pharmacy_id      UUID,
    medications_enc  TEXT         NOT NULL,
    notes_enc        TEXT,
    status           TEXT         NOT NULL DEFAULT 'pending'
                                  CHECK (status IN ('pending','sent','dispensed','cancelled')),
    dispensed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prescriptions_consultation_id ON prescriptions(consultation_id);
CREATE INDEX idx_prescriptions_patient_id      ON prescriptions(patient_id);
CREATE INDEX idx_prescriptions_status          ON prescriptions(status);

-- ─── Clinical Audit Log ───────────────────────────────────────────────────────

CREATE TABLE clinical_audit_log (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type  TEXT         NOT NULL,
    resource_id    UUID         NOT NULL,
    patient_id     UUID         NOT NULL,
    action         TEXT         NOT NULL CHECK (action IN ('create','read','update','delete')),
    accessor_id    UUID         NOT NULL,
    accessor_role  TEXT         NOT NULL,
    ip_address     TEXT         NOT NULL,
    request_id     TEXT         NOT NULL,
    changes        JSONB,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clinical_audit_resource_id ON clinical_audit_log(resource_id);
CREATE INDEX idx_clinical_audit_patient_id  ON clinical_audit_log(patient_id);
CREATE INDEX idx_clinical_audit_accessor_id ON clinical_audit_log(accessor_id);
CREATE INDEX idx_clinical_audit_created_at  ON clinical_audit_log(created_at);

CREATE RULE clinical_audit_no_update AS ON UPDATE TO clinical_audit_log DO INSTEAD NOTHING;
CREATE RULE clinical_audit_no_delete AS ON DELETE TO clinical_audit_log DO INSTEAD NOTHING;
