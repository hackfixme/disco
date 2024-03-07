CREATE TABLE _meta (
  -- This is a key-value metadata table, so limit it to only one row.
  id                INTEGER      PRIMARY KEY CHECK (id = 1),
  version           VARCHAR(32)  UNIQUE NOT NULL,
  public_key        VARCHAR(64)  UNIQUE NOT NULL,
  private_key_hash  VARCHAR(64)  UNIQUE NOT NULL
);
