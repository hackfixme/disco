# Disco - Command-line interface

## Quick start

### Setting and getting values
```sh
# Use a forward slash as key separator to create hierarchies.
$ disco set myapp/mykey myvalue

$ disco get myapp/mykey
myvalue
```

List all keys:
```sh
$ disco ls
myapp/mykey
```


### Namespaces

Namespaces are created automatically when used. They are supported by the `get`,
`set` and `ls` commands.

Unless specified, the `default` namespace is used. To change this, set the value of the `DISCO_NAMESPACE` environment variable.

- Set a value in a specific namespace:
  ```sh
  $ disco set --namespace myns myapp/mykey myvalue
  ```

- List all keys in all namespaces:
  ```sh
  $ disco ls --namespace='*'
  default:myapp/mykey
  myns:myapp/mykey
  ```

  The namespace of the key is prefixed.


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

### `ls`

### `mount`

### `remote`

### `role`

### `serve`

### `user`
