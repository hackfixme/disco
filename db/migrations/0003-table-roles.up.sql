CREATE TABLE roles (
  id         INTEGER      PRIMARY KEY,
  name       VARCHAR(32)  UNIQUE NOT NULL
);

CREATE TABLE role_permissions (
  role_id     INTEGER       NOT NULL,
  namespaces  VARCHAR(128)  NOT NULL,
  actions     VARCHAR(16)   NOT NULL,
  target      VARCHAR(512)  NOT NULL,
  FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE
);

CREATE TABLE users_roles (
  user_id  INTEGER   NOT NULL,
  role_id  INTEGER   NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE
);
