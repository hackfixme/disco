# Role-based Access Control

Disco uses a role-based access control (RBAC) system. It is a flexible way of controlling access to resources of Disco, from store namespaces to individual keys, API access, and management of users and roles themselves.


## Resources

Resources are objects that users can access. These include:
- `store`: the key-value store where values are encrypted. Keys are stored as plain text.
- `user`: manages user accounts.
- `role`: manages roles and permissions assigned to users.
- `invite`: manages invites of remote users.

## Permissions

Permissions are a set of rules that control how resources are accessed, and which specific objects are allowed to be accessed.

They consist of two concepts: actions and targets.

**Actions** dictate _how_ resources can be accessed. There are 3 possible actions:
- `read` (`r`): allows reading resource data.
- `write` (`w`): allows writing resource data, including creating new data.
- `delete` (`d**): allows deleting resource data.

**Targets** dictate _which_ objects that are part of the resource can be accessed. They can be granular and specific to a single key or object within the resource, or global for all objects, or even all resources. Targets are specified as `<resource>:<pattern>`.

For example, for the `store` resource, it's possible to allow access to all store data with `store:*`, a subset of keys with `store:myapp/*`, or a specific key with `store:myapp/mykey`.


## Roles

Roles are a collection of permissions, assigned to one or more users.


## Default roles

| Name  | Namespaces | Actions | Target  |
|-------|------------|---------|---------|
| admin | *          | *       | *       |
| node  | *          | read    | store:* |
| user  | *          | *       | store:* |

These roles are created by default when running `disco init`.
The `admin` role allows any action, on any target, in any namespace. This role is assigned to the local CLI user, but be careful with assigning it to any remote users, as it effectively gives unrestricted access to all data on the node.
The `node` role allows reading any store data in any namespace. This is a generic role that can be used for remote nodes that only need read permissions. It would be more secure to add more restrictive roles with a granular target for specific keys or key hierarchies instead.
The `user` role allows any action on any store data in any namespace. This is a generic role for users that can manage store data, but as with the `node` role, it would be more secure to create a more granular role.


## Custom roles

- ```sh
  disco role add myrole 'rwd:dev,prod:store:app1/*,app2/value'
  ```
  This adds a new `myrole` role, with read, write and delete permissions on the `store` resource, in `dev` and `prod` namespaces, for all keys under `app1/*`, and the `app2/value` key.
