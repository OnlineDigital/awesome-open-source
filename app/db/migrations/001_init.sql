CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE categories (
  id          bigserial PRIMARY KEY,
  slug        text NOT NULL UNIQUE,
  icon        text NOT NULL DEFAULT '',
  sort_order  int  NOT NULL DEFAULT 0,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE category_translations (
  category_id bigint NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  locale      text   NOT NULL,
  name        text   NOT NULL,
  description text   NOT NULL DEFAULT '',
  PRIMARY KEY (category_id, locale)
);

CREATE TABLE projects (
  id              bigserial PRIMARY KEY,
  slug            text NOT NULL UNIQUE,
  github_owner    text NOT NULL,
  github_repo     text NOT NULL,
  category_id     bigint REFERENCES categories(id) ON DELETE SET NULL,
  homepage        text NOT NULL DEFAULT '',
  license         text NOT NULL DEFAULT '',
  language        text NOT NULL DEFAULT '',
  topics          text[] NOT NULL DEFAULT '{}',
  avatar_url      text NOT NULL DEFAULT '',
  stars           int NOT NULL DEFAULT 0,
  forks           int NOT NULL DEFAULT 0,
  open_issues     int NOT NULL DEFAULT 0,
  archived        boolean NOT NULL DEFAULT false,
  default_branch  text NOT NULL DEFAULT '',
  gh_created_at   timestamptz,
  pushed_at       timestamptz,
  last_commit_at  timestamptz,
  last_release_at timestamptz,
  last_release_tag text NOT NULL DEFAULT '',
  activity_score  numeric(2,1),
  activity_tier   smallint,
  score_breakdown jsonb,
  signals         jsonb,
  youtube_id      text NOT NULL DEFAULT '',
  status          text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published')),
  featured        boolean NOT NULL DEFAULT false,
  embedding       vector,
  synced_at       timestamptz,
  sync_error      text NOT NULL DEFAULT '',
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  UNIQUE (github_owner, github_repo)
);
CREATE INDEX projects_category_idx ON projects (category_id) WHERE status = 'published';
CREATE INDEX projects_score_idx ON projects (activity_score DESC NULLS LAST);
CREATE INDEX projects_synced_idx ON projects (synced_at NULLS FIRST);

CREATE TABLE project_translations (
  project_id  bigint NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  locale      text   NOT NULL,
  tagline     text   NOT NULL DEFAULT '',
  description text   NOT NULL DEFAULT '',
  source      text   NOT NULL DEFAULT 'human' CHECK (source IN ('human','machine')),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, locale)
);

CREATE TABLE project_screenshots (
  id         bigserial PRIMARY KEY,
  project_id bigint NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  path       text NOT NULL,
  alt        text NOT NULL DEFAULT '',
  position   int  NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX project_screenshots_project_idx ON project_screenshots (project_id, position);

CREATE TABLE admin_users (
  id            bigserial PRIMARY KEY,
  email         text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now()
);
