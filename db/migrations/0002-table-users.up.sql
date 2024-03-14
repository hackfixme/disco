CREATE TABLE users (
  id                INTEGER      NOT NULL PRIMARY KEY,
  name              VARCHAR(32)  UNIQUE NOT NULL,
  public_key        VARCHAR(64)  UNIQUE,
  private_key_hash  VARCHAR(64)  UNIQUE
);
