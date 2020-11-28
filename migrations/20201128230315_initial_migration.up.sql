CREATE TABLE blobs (
    id uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid (),
    file_name text NOT NULL,
    mime_type text NOT NULL,
    file_size_bytes bigint NOT NULL,
    extension TEXT NOT NULL,
    data bytea NOT NULL,
    views integer NOT NULL DEFAULT 0,

    deleted_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE TABLE gcodes (
    id uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid (),
    name TEXT NOT NULL,
    blob_id UUID NOT NULL REFERENCES blobs(id),
    deleted_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    created_at timestamptz NOT NULL DEFAULT NOW()

);
