CREATE TYPE entity_type AS ENUM ('person', 'organization');
CREATE TABLE entity (
  id          integer PRIMARY KEY GENERATED ALWAYS AS IDENTITY
, uuid        varchar(32) UNIQUE NOT NULL
, created_at  timestamp NOT NULL
, updated_at  timestamp
, type        entity_type NOT NULL
, name        varchar(256) UNIQUE NOT NULL
, description varchar(2048)
, location    varchar(512)
);
