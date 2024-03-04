CREATE TABLE IF NOT EXISTS "yatai_third_party_auth" (
    id SERIAL PRIMARY KEY,
    uid VARCHAR(32) UNIQUE NOT NULL DEFAULT generate_object_id(),
    access_token VARCHAR(1000) NOT NULL,
    refresh_token VARCHAR(1000) NOT NULL,
    expires_in INTEGER NOT NULL,
    token_type VARCHAR(10),
    cluster_id INTEGER NOT NULL REFERENCES "cluster"("id") ON DELETE CASCADE,
    organization_id INTEGER NOT NULL REFERENCES "organization"("id") ON DELETE CASCADE,
    kube_namespace VARCHAR(128) NOT NULL,
    manifest JSONB,
    creator_id INTEGER NOT NULL REFERENCES "user"("id") ON DELETE CASCADE,
    latest_heartbeat_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    latest_installed_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE
);
