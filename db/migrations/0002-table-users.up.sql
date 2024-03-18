CREATE TABLE users (
  id                INTEGER      NOT NULL PRIMARY KEY,
  name              VARCHAR(32)  UNIQUE NOT NULL,
  type              INTEGER      CHECK( type IN (1, 2) ) NOT NULL, -- 1: local, 2: remote
  public_key        VARCHAR(64)  UNIQUE,
  private_key_hash  VARCHAR(64)  UNIQUE
);
