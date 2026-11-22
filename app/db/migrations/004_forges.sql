-- Driver-based code hosting: projects can live on GitHub, GitLab or future forges.
ALTER TABLE projects RENAME COLUMN github_owner TO owner;
ALTER TABLE projects RENAME COLUMN github_repo TO repo;
ALTER TABLE projects RENAME COLUMN gh_created_at TO repo_created_at;
ALTER TABLE projects ADD COLUMN forge text NOT NULL DEFAULT 'github';
ALTER TABLE projects ADD COLUMN repo_url text NOT NULL DEFAULT '';
UPDATE projects SET repo_url = 'https://github.com/' || owner || '/' || repo;
ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_github_owner_github_repo_key;
-- Repository paths are case-insensitive on every forge; refuse to guess which case-variant duplicate to keep.
DO $$
DECLARE dup text;
BEGIN
  SELECT string_agg(owner || '/' || repo || ' (#' || id || ')', ', ') INTO dup FROM projects p
  WHERE EXISTS (SELECT 1 FROM projects q WHERE q.id <> p.id AND q.forge = p.forge AND lower(q.owner) = lower(p.owner) AND lower(q.repo) = lower(p.repo));
  IF dup IS NOT NULL THEN
    RAISE EXCEPTION 'projects differing only by letter case must be merged or deleted before migration 004: %', dup;
  END IF;
END $$;
CREATE UNIQUE INDEX projects_forge_repo_key ON projects (forge, lower(owner), lower(repo));
