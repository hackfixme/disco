# Disco - Command-line interface

## Quick start

### Setting and getting values
```sh
# Use a forward slash as key separator to create hierarchies.
$ disco set myapp/mykey myvalue

$ disco get myapp/mykey
myvalue
```


### Namespaces

- List all namespaces:
  ```sh
  $ disco ns ls
  development
  staging
  production
  ```

- Create a new namespace:
  ```sh
  $ disco ns add myns
  ```

- Set a value in the new namespace:
  ```sh
  $ disco --namespace myns set myapp/mykey myvalue
  ```

  Unless specified, the development namespace is used by default.
  To change this, run:
  ```sh
  $ disco ns default myns
  ```


### Roles
- List all roles:
  ```sh
  $ disco role ls
  NAME     PERMISSIONS
  admin    *:*:rw
  user     keys:*:ro,values:*:ro
  ```

- Create a new role that can only read keys and values in the myapp hierarchy:
  ```sh
  $ disco role new myrole 'keys:myapp/*:ro,values:myapp/*:ro'
  ```


### Users

- Add a new user with the `myrole` role, in the development and staging namespaces:
  ```sh
  $ disco user add myuser --namespaces development,myns --roles myrole
  ```

- List all users:
  ```sh
  $ disco user ls
  NAME     NAMESPACES          ROLES
  admin    *                   admin
  user     *                   user
  myuser   development,myns    myrole
  ```

- Create an invite key for a specific user:
  ```sh
  $ disco user invite myuser
  Invite key: KxI7jh7rkHw7MMNqLF7cJmzjLS9o0EI/MAYF...
  Expires: 2024-03-01 16:32:20 UTC
  ```


### Server

- Start the HTTP server so that the client can connect:
  ```sh
  $ disco serve --address 10.0.0.10:8080
  ```


### Remotes

- On the client machine, add a new remote node using the invite key:
  ```sh
  $ disco remote add myserver 10.0.0.10:8080 KxI7jh7rkHw7MMNqLF7cJmzjLS9o0EI/MAYF...
  ```

- Then use the `--remote` option on `get` or `set` commands:
  ```sh
  $ disco get --remote myserver --namespace myns myapp/mykey
  myvalue
  ```


## Commands

### `get`

### `set`

### `mount`

### `namespace`/`ns`

### `remote`

### `role`

### `serve`

### `user`
