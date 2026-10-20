ALTER TABLE resume_analyses
    ADD COLUMN external_ai_consent BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN external_ai_consent_at TIMESTAMPTZ,
    ADD CONSTRAINT resume_analyses_consent_timestamp_check
        CHECK ((external_ai_consent = true AND external_ai_consent_at IS NOT NULL) OR
               (external_ai_consent = false AND external_ai_consent_at IS NULL));
