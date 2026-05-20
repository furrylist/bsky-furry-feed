CREATE TABLE audit_attachments (
    id BIGSERIAL PRIMARY KEY,
    actor_did TEXT NOT NULL REFERENCES candidate_actors (did),
    mime_type TEXT NOT NULL,
    data BYTEA NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL
);
