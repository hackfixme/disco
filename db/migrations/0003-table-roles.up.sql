CREATE TABLE roles (
  id         INTEGER      PRIMARY KEY,
  name       VARCHAR(32)  UNIQUE NOT NULL
);

CREATE TABLE role_permissions (
  role_id   INTEGER       NOT NULL,
  action    VARCHAR(32)   NOT NULL,
  target    VARCHAR(512)  NOT NULL,
  FOREIGN KEY(role_id) REFERENCES roles(id)
);

CREATE TABLE users_roles (
  user_id  INTEGER   NOT NULL,
  role_id  INTEGER   NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(role_id) REFERENCES roles(id)
);
