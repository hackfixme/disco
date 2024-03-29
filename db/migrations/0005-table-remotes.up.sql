CREATE TABLE remotes (
  id               INTEGER       PRIMARY KEY,
  created_at       TIMESTAMP     NOT NULL,
  name             VARCHAR(32)   UNIQUE NOT NULL,
  address          VARCHAR(128)  UNIQUE NOT NULL,
  client_cert_enc  BLOB          NOT NULL
);
