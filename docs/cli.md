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

Unless specified, the `default` namespace is used. To change this, set the value of the `DISCO_NAMESPACE` environment variable, or set the `--namespace` option.

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

  The namespace of the key is shown as a prefix.


### Roles
- List all roles:
  ```sh
  $ disco role ls
  NAME    NAMESPACES   ACTIONS   TARGET
  admin   *            *         *
  node    *            read      store:*
  user    *            *         store:*
  ```

- Create a new role that can only read store keys and values in the `myapp` hierarchy in all namespaces:
  ```sh
  $ disco role add myrole 'r:*:store:myapp/*'
  ```

- Update the role so that it can read, write and delete all store keys and values in the `dev` and `staging` namespaces, and write store keys and values in the `myapp` hierarchy in the `prod` namespace:
  ```sh
  $ disco role update myrole 'rwd:dev,staging:store:*' 'w:prod:store:myapp/*'
  ```


### Users

- Add a new user with the `myrole` role:
  ```sh
  $ disco user add myuser --roles myrole
  ```

- List all users:
  ```sh
  $ disco user ls
  NAME     ROLES
  myuser   myrole
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

### `remote`

### `role`

### `serve`

### `user`
