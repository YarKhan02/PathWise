-- Corpus app schema: basic-gin.
--
-- org_id is the tenant discriminator. Relationships are declared as explicit
-- foreign keys, since PathWise will not invent a relationship where no
-- constraint exists.

CREATE TABLE organizations (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    org_id        BIGINT NOT NULL REFERENCES organizations (id),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'teacher'
);

CREATE TABLE batches (
    id     BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES organizations (id),
    name   TEXT NOT NULL
);

CREATE TABLE students (
    id          BIGSERIAL PRIMARY KEY,
    org_id      BIGINT NOT NULL REFERENCES organizations (id),
    batch_id    BIGINT NOT NULL REFERENCES batches (id),
    name        TEXT NOT NULL,
    email       TEXT NOT NULL,
    national_id TEXT
);

CREATE TABLE sessions (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (id),
    token      TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_students_org ON students (org_id);
CREATE INDEX idx_students_batch ON students (batch_id);
CREATE INDEX idx_batches_org ON batches (org_id);
