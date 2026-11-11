-- Curated topic pages with feature comparison tables, extra (resync-proof) project topics, and MCP access tokens.
ALTER TABLE projects ADD COLUMN extra_topics text[] NOT NULL DEFAULT '{}';
CREATE INDEX projects_all_topics_idx ON projects USING gin ((topics || extra_topics));

CREATE TABLE topics (
  id          bigserial PRIMARY KEY,
  slug        text NOT NULL UNIQUE,
  sort_order  int  NOT NULL DEFAULT 0,
  published   boolean NOT NULL DEFAULT false,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE topic_translations (
  topic_id bigint NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  locale   text   NOT NULL,
  name     text   NOT NULL DEFAULT '',
  summary  text   NOT NULL DEFAULT '',
  body     text   NOT NULL DEFAULT '',
  PRIMARY KEY (topic_id, locale)
);

CREATE TABLE topic_features (
  id       bigserial PRIMARY KEY,
  topic_id bigint NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  position int NOT NULL DEFAULT 0
);
CREATE INDEX topic_features_topic_idx ON topic_features (topic_id, position);

CREATE TABLE topic_feature_translations (
  feature_id  bigint NOT NULL REFERENCES topic_features(id) ON DELETE CASCADE,
  locale      text   NOT NULL,
  name        text   NOT NULL DEFAULT '',
  description text   NOT NULL DEFAULT '',
  PRIMARY KEY (feature_id, locale)
);

-- state: yes | no | partial | text (only the note is shown). notes: {"en": "...", "ro": "..."}.
CREATE TABLE project_feature_values (
  project_id bigint NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  feature_id bigint NOT NULL REFERENCES topic_features(id) ON DELETE CASCADE,
  state      text   NOT NULL CHECK (state IN ('yes','no','partial','text')),
  notes      jsonb  NOT NULL DEFAULT '{}',
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, feature_id)
);

CREATE TABLE mcp_tokens (
  id           bigserial PRIMARY KEY,
  name         text NOT NULL,
  token        text NOT NULL UNIQUE,
  created_at   timestamptz NOT NULL DEFAULT now(),
  rotated_at   timestamptz,
  last_used_at timestamptz,
  revoked_at   timestamptz
);
