CREATE TABLE remotes (
  id                   INTEGER       PRIMARY KEY,
  created_at           TIMESTAMP     NOT NULL,
  name                 VARCHAR(32)   UNIQUE NOT NULL,
  address              VARCHAR(128)  NOT NULL,
  tls_ca_cert          VARCHAR(512)  NOT NULL,
  tls_server_san       VARCHAR(32)   NOT NULL,
  tls_client_cert_enc  BLOB          NOT NULL,
  tls_client_key_enc   BLOB          NOT NULL
);
