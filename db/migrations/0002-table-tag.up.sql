CREATE EXTENSION ltree;
CREATE TABLE tag (
  id          INTEGER PRIMARY KEY UNIQUE NOT NULL
, name        VARCHAR(64) UNIQUE NOT NULL
, path        LTREE UNIQUE NOT NULL
);
CREATE INDEX path_gist_idx ON tag USING gist (path); -- for <, <=, =, >=, >, @>, <@, @, ~, ?
CREATE INDEX path_btree_idx ON tag USING btree (path); -- for <, <=, =, >=, >

CREATE TABLE entity_tag (
  entity_id   INTEGER NOT NULL REFERENCES entity (id)
, tag_id      INTEGER NOT NULL REFERENCES tag (id)
-- NOTE: Order of columns matters in composite primary key indexes.
-- So consider adding another index on (tag_id, entity_id).
, CONSTRAINT entity_tag_pkey PRIMARY KEY (entity_id, tag_id)
);
