BEGIN;
CREATE TABLE IF NOT EXISTS users (
    user_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email text NOT NULL UNIQUE,
    password_hash text NOT NULL
);
CREATE TABLE IF NOT EXISTS analyses (
    analysis_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(user_id),
    resume_text text NOT NULL CHECK (length(btrim(resume_text)) > 0),
    vacancy_text text NOT NULL CHECK (length(btrim(vacancy_text)) BETWEEN 1 AND 20000),
    result jsonb NOT NULL,
    score_source text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (jsonb_typeof(result) = 'object' AND result ? 'score'),
    CHECK (jsonb_typeof(result->'score') = 'number'),
    CHECK ((result->>'score')::numeric BETWEEN 0 AND 100),
    CHECK ((result->>'score')::numeric = trunc((result->>'score')::numeric))
);
CREATE INDEX IF NOT EXISTS analyses_owner_order
ON analyses(user_id, created_at DESC, analysis_id DESC);
COMMIT;
