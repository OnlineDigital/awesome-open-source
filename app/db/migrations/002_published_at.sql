ALTER TABLE projects ADD COLUMN published_at timestamptz;
UPDATE projects SET published_at = created_at WHERE status = 'published';
CREATE INDEX projects_published_order_idx ON projects (published_at, id) WHERE status = 'published';
