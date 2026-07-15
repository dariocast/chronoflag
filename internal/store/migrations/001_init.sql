CREATE TABLE IF NOT EXISTS instances (
  id text PRIMARY KEY,
  snapshot jsonb NOT NULL,
  tier text NOT NULL DEFAULT 'free',
  lifecycle text NOT NULL DEFAULT 'active',
  last_control_at timestamptz NOT NULL,
  archived_at timestamptz,
  purge_at timestamptz
);
CREATE INDEX IF NOT EXISTS instances_retention_idx ON instances (lifecycle, last_control_at, purge_at);
CREATE TABLE IF NOT EXISTS capabilities (
  token_hash bytea PRIMARY KEY,
  instance_id text NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
  scope text NOT NULL CHECK (scope IN ('control','view'))
);
CREATE TABLE IF NOT EXISTS clock_events (
  instance_id text NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
  sequence bigint NOT NULL,
  clock_id text NOT NULL,
  command_id text NOT NULL,
  event jsonb NOT NULL,
  PRIMARY KEY (instance_id, sequence),
  UNIQUE (instance_id, command_id)
);
CREATE INDEX IF NOT EXISTS clock_events_clock_idx ON clock_events (instance_id, clock_id, sequence DESC);
