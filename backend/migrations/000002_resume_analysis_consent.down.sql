ALTER TABLE resume_analyses
    DROP CONSTRAINT IF EXISTS resume_analyses_consent_timestamp_check,
    DROP COLUMN IF EXISTS external_ai_consent_at,
    DROP COLUMN IF EXISTS external_ai_consent;
