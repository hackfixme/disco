# Getting started

After you've [installed Disco](/README.md#installation), confirm the version you installed by running:

```sh
disco --version
```

It should output something like:
```
Disco v0.1.0 (commit/c2d4d8807b, go1.22.0, linux/amd64)
```

If you get an error like `command not found`, see the [Troubleshooting page](./troubleshooting.md).


## First-time initialization

Before running any other commands, the Disco data stores and local encryption keys need to be created. This is done once by running:

```sh
disco init
```

This will create the data stores in your system's default data store directory: `~/.local/share/disco` on Linux, `~/Library/Application Support/disco` on macOS, and `C:\Users\<Username>\AppData\Local\disco` on Windows. You can change this by setting the `--data-dir` option, or `DISCO_DATA_DIR` environment variable.

It should output something like:
```
New encryption key: 6uVB3iXnfdE3aR52vGQGtBHAfKTXx48PwhGwyQdRr4Ey

Make sure to store this key in a secure location, such as a password manager.

It will only be shown once, and you won't be able to access the data on this node without it!
```

You should follow that suggestion, and store the encryption key in a secure location.


## Setting the encryption key

Most Disco commands require the encryption key to read and write data. You can pass it via the `--encryption-key` option, or by setting the `DISCO_ENCRYPTION_KEY` environment variable. For example:

```sh
 export DISCO_ENCRYPTION_KEY=6uVB3iXnfdE3aR52vGQGtBHAfKTXx48PwhGwyQdRr4Ey
```

> [!WARNING]
> Be aware that on Linux the CLI arguments and process environment can be read by any
> other process run by the same user via the `/proc` filesystem, which means another
> process could read the Disco encryption key. If this is a concern for your use
> case, consider [running Disco inside a container instead](./container.md),
> or using another isolation mechanism (e.g. a virtual machine).
>
> Also, be careful with your shell history. Depending on your shell configuration,
> the Disco encryption key could be stored in your shell history. If using the
> `--encryption-key` option, prefer reading directly from your password manager.
> For example, using 1Password: `--encryption-key="$(op item get Disco)"`.
> If setting the `DISCO_ENCRYPTION_KEY` environment variable, ensure that your
> shell is configured to not store commands in history that begin with a space
> character. In Bash, you can set `HISTCONTROL=ignorespace` (or `ignoreboth`)
> in your `~/.bashrc`, and on Zsh add `setopt histignorespace` to your `~/.zshrc`.
> This way running ` export DISCO_ENCRYPTION_KEY=...` (note the leading space) won't
> be saved in your `~/.bash_history` or `~/.zsh_history` file.


## Setting and getting values

To store a value in the data store, use the `set` command. For example:

```sh
disco set myapp/mykey myvalue
```

Use a forward slash as key separator to group keys and create logical hierarchies.

Then to retrieve the value, use the `get` command:

```
$ disco get myapp/mykey
myvalue
```


## Listing keys

To list stored keys use the `ls` command:

```sh
$ disco ls
myapp/mykey
```


## Deleting keys

To delete keys use the `rm` command:

```sh
disco rm myapp/mykey
```


## Namespaces

Namespaces allow separating keys according to their purpose, or any other criteria. For example, it's common to separate keys that belong to different environments like development, staging and production. This way access to each environment can be controlled separately.

There are no naming or usage restrictions for namespaces, so feel free to use them however makes most sense for your use case.

Namespaces are created automatically when used, so it's not necessary to manage them manually.

They are supported by the `get`, `set` and `ls` commands with the `--namespace` option, or by setting the `DISCO_NAMESPACE` environment variable. Unless specified, the `default` namespace is used.

Here are some common operations involving namespaces:

- Setting a value in a specific namespace:
  ```sh
  disco set --namespace myns myapp/mynskey mynsvalue
  ```

- Getting a value from a specific namespace:
  ```sh
  $ disco get --namespace myns myapp/mynskey
  mynsvalue
  ```

- Listing keys in a specific namespace:
  ```sh
  $ disco ls --namespace myns
  myapp/mynskey
  ```

- Listing keys in all namespaces:
  ```sh
  $ disco ls --namespace='*'
  NAMESPACE   KEY
  default     myapp/mykey
  myns        myapp/mynskey
  ```

  Note that the asterisk needs to be quoted or escaped so that it's not interpreted by the shell.


## Roles

Roles allow controling access to Disco resources. They consist of a set of permissions that define the actions [users](#users) are allowed to perform on specific targets.

See more detailed information on the [Roles page](./roles.md).

- List all roles:
  ```sh
  $ disco role ls
  NAME    NAMESPACES   ACTIONS   TARGET
  admin   *            *         *
  node    *            read      store:*
  user    *            *         store:*
  ```

  These are the default roles created with `disco init`.

- Create a new role that can only read store keys and values in the `myapp` hierarchy in all namespaces:
  ```sh
  $ disco role add myrole 'r:*:store:myapp/*'
  ```

- Update the role so that it can read, write and delete all store keys and values in the `dev` and `staging` namespaces, and write store keys and values in the `myapp` hierarchy in the `prod` namespace:
  ```sh
  $ disco role update myrole 'rwd:dev,staging:store:*' 'w:prod:store:myapp/*'
  ```


## Users

Users can be added to allow remote access to Disco resources. They should have one or more associated roles that control which resources they can access.

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

- Update the roles of a user:
  ```sh
  $ disco user update myuser --roles=node,user
  ```

- Remove a user:
  ```sh
  $ disco user rm myuser
  ```


## Invites and remotes

Invites are created for users, and allow remote Disco nodes to connect to the node that created the invite.

- Create an invite token for a specific user:
  ```sh
  $ disco invite user myuser
  Token: 5RMduyPEncYL6EH9c3gzpzDbYq3vjE...
  Expires: 2024-04-18 23:54:10 (1h0m0s)
  ```

  Note that an invite token is sensitive information, and should be transmitted securely to the invited node.

- On the client node, add a `remote` object using the generated token, and specifying the [server](#server) address of the node that issued the invite:
  ```sh
  $ disco remote add myserver 10.0.0.10:2020 5RMduyPEncYL6EH9c3gzpzDbYq3vjE...
  ```

- Then the client node can use the `--remote` option on `get`, `set` and `ls` commands:
  ```sh
  $ disco get --remote myserver --namespace myns myapp/mykey
  myvalue
  ```


## Server

Before a client node can redeem their invite token, or access Disco resources remotely, the web server needs to be started:

```sh
$ disco serve --address 10.0.0.10:2020
```
