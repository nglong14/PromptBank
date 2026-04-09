-- Initialize the database schema

-- Enable the uuid-ossp extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create the users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create the prompts table
CREATE TABLE IF NOT EXISTS prompts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    category TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create the prompt_versions table
CREATE TABLE IF NOT EXISTS prompt_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    prompt_id UUID NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    assets JSONB NOT NULL DEFAULT '{}',
    framework_id TEXT NOT NULL DEFAULT '',
    technique_ids TEXT[] NOT NULL DEFAULT '{}',
    composed_output TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(prompt_id, version_number)
);

-- Create the prompt_lineage table
CREATE TABLE IF NOT EXISTS prompt_lineage (
    id BIGSERIAL PRIMARY KEY,
    prompt_id UUID NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
    derived_from_prompt_id UUID NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
    derived_from_version_id UUID REFERENCES prompt_versions(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(prompt_id)
);

-- Create the set_updated_at function
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create the prompts_set_updated_at trigger
DROP TRIGGER IF EXISTS prompts_set_updated_at ON prompts;
CREATE TRIGGER prompts_set_updated_at
BEFORE UPDATE ON prompts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
