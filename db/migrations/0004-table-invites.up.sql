CREATE TABLE invites (
  id           INTEGER       PRIMARY KEY,
  uuid         VARCHAR(32)   UNIQUE NOT NULL,
  created_at   TIMESTAMP     NOT NULL,
  expires      TIMESTAMP     NOT NULL,
  user_id      INTEGER       NOT NULL,
  token        VARCHAR(128)  NOT NULL,
  privkey_enc  BLOB          NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
