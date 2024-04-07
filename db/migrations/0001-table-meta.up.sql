CREATE TABLE _meta (
  -- This is a key-value metadata table, so limit it to only one row.
  id                  INTEGER      PRIMARY KEY CHECK (id = 1),
  version             VARCHAR(32)  UNIQUE NOT NULL,
  server_tls_cert     VARCHAR      UNIQUE NOT NULL,
  server_tls_key_enc  BLOB         UNIQUE NOT NULL
);
