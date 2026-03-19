CREATE TYPE visibility AS ENUM ('private', 'public');

CREATE TABLE notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    media JSONB NOT NULL DEFAULT '[]',
    tags JSONB NOT NULL DEFAULT '[]',
    visibility visibility NOT NULL DEFAULT 'private',
    is_draft BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notes_user_id ON notes(user_id, created_at DESC);
CREATE INDEX idx_notes_visibility ON notes(visibility, created_at DESC) WHERE visibility = 'public';
CREATE INDEX idx_notes_tags ON notes USING GIN(tags);
