-- Add "profiles" table (1:1 with users)
CREATE TABLE IF NOT EXISTS "profiles" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "username" character varying(32) NOT NULL,
  "display_name" character varying(100) NULL,
  "bio" text NULL,
  "avatar_url" text NULL,
  "default_language" character varying(32) NOT NULL DEFAULT 'python',
  "editor_theme" character varying(32) NOT NULL DEFAULT 'dark',
  "company" character varying(100) NULL,
  "github_url" character varying(255) NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_users_profile" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS "idx_profiles_user_id" ON "profiles" ("user_id");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_profiles_username" ON "profiles" ("username");

-- Add "sessions" table (single active session per user + metadata)
CREATE TABLE IF NOT EXISTS "sessions" (
  "id" character varying(64) NOT NULL,
  "user_id" uuid NOT NULL,
  "provider" character varying(20) NOT NULL DEFAULT 'password',
  "ip_address" character varying(45) NULL,
  "user_agent" text NULL,
  "device_name" character varying(120) NULL,
  "expires_at" timestamptz NOT NULL,
  "last_rotated_at" timestamptz NOT NULL,
  "last_active_at" timestamptz NOT NULL,
  "created_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_users_sessions" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS "idx_sessions_expires_at" ON "sessions" ("expires_at");
CREATE INDEX IF NOT EXISTS "idx_sessions_user_id" ON "sessions" ("user_id");
