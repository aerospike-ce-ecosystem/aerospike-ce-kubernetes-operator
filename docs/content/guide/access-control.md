---
sidebar_position: 4
title: Access Control (ACL)
---

# Access Control (ACL)

This guide covers configuring authentication and authorization for Aerospike CE clusters using the operator.

## Overview

Aerospike CE supports access control (ACL) to restrict who can connect to the cluster and what operations they can perform. When ACL is enabled:

- All client connections must authenticate with a username and password
- Each user is assigned one or more **roles** that define their permissions
- The operator manages role and user creation via the Aerospike admin API

:::warning
Aerospike CE 8.x does not support the `security` stanza in `aerospike.conf`. ACL is managed entirely through the operator's `aerospikeAccessControl` spec, which uses the Aerospike admin API to configure users and roles at runtime.
:::

## Prerequisites

### Create Kubernetes Secrets

Each user's password must be stored in a Kubernetes Secret. The Secret must contain a `password` key:

```bash
# Create a secret for the admin user
kubectl -n aerospike create secret generic admin-secret \
  --from-literal=password='admin-password-here'

# Create a secret for an application user
kubectl -n aerospike create secret generic app-secret \
  --from-literal=password='app-password-here'
```

## Basic ACL Configuration

The minimal ACL configuration requires at least one user with both `sys-admin` and `user-admin` roles:

```yaml
apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: aerospike-acl
  namespace: aerospike
spec:
  size: 3
  image: aerospike:ce-8.1.1.1
  aerospikeAccessControl:
    users:
      - name: admin
        secretName: admin-secret
        roles:
          - sys-admin
          - user-admin
      - name: appuser
        secretName: app-secret
        roles:
          - read-write
  aerospikeConfig:
    service:
      cluster-name: aerospike-acl
    namespaces:
      - name: test
        replication-factor: 2
        storage-engine:
          type: memory
```

## Built-in Roles

Aerospike CE provides the following predefined roles. These can be assigned directly to users without defining them in the `roles` list:

| Role | Description |
|------|-------------|
| `user-admin` | Create/drop users, grant/revoke roles |
| `sys-admin` | Cluster administration (truncate, config, info commands) |
| `data-admin` | Index management (create/drop secondary indexes, UDFs) |
| `read` | Read records |
| `write` | Write (insert/update/delete) records |
| `read-write` | Read and write records |
| `read-write-udf` | Read, write, and execute UDFs |
| `truncate` | Truncate namespaces/sets |

:::info
The `superuser` role exists only in Aerospike Enterprise Edition. CE clusters must use the built-in roles listed above or define custom roles.
:::

## Custom Roles

Define custom roles with fine-grained privileges:

```yaml
spec:
  aerospikeAccessControl:
    roles:
      - name: inventory-reader
        privileges:
          - read.inventory           # Read on 'inventory' namespace
      - name: orders-writer
        privileges:
          - read-write.orders        # Read-write on 'orders' namespace
          - read-write.orders.items  # Read-write on 'orders.items' set
    users:
      - name: admin
        secretName: admin-secret
        roles:
          - sys-admin
          - user-admin
      - name: inventory-svc
        secretName: inventory-secret
        roles:
          - inventory-reader
      - name: orders-svc
        secretName: orders-secret
        roles:
          - orders-writer
```

### Privilege Format

Privileges follow the format: `<code>[.<namespace>[.<set>]]`

| Code | Description |
|------|-------------|
| `read` | Read records |
| `write` | Write records |
| `read-write` | Read and write records |
| `read-write-udf` | Read, write, and execute UDFs |
| `sys-admin` | System administration |
| `user-admin` | User administration |
| `data-admin` | Data administration |
| `truncate` | Truncate data |

**Examples:**
- `read` — global read on all namespaces
- `read-write.myns` — read-write on the `myns` namespace
- `write.myns.myset` — write on the `myset` set within `myns`

## Webhook Validation Rules

The operator's admission webhook enforces the following ACL rules:

1. **Admin user required**: At least one user must have both `sys-admin` and `user-admin` roles. Without this, the operator cannot manage ACL after initial setup.

2. **Role cross-validation**: Every role referenced by a user must be either a built-in role or explicitly defined in the `roles` list. Referencing an undefined role is rejected.

3. **Privilege code validation**: Each privilege in a custom role must use a valid privilege code. Invalid codes are rejected with a descriptive error.

4. **Secret requirement**: Every user must have a `secretName` pointing to a Kubernetes Secret containing the password.

## Troubleshooting

### Common Validation Errors

**"must have at least one user with both 'sys-admin' and 'user-admin' roles"**

Ensure at least one user has both roles:

```yaml
users:
  - name: admin
    secretName: admin-secret
    roles:
      - sys-admin
      - user-admin    # Both roles required on the same user
```

**"user X references undefined role Y"**

The role must be a built-in role or defined in the `roles` list:

```yaml
roles:
  - name: custom-role       # Define the custom role
    privileges:
      - read.myns
users:
  - name: myuser
    secretName: myuser-secret
    roles:
      - custom-role          # Now this reference is valid
```

**"role X has invalid privilege code Y"**

Use only valid privilege codes (`read`, `write`, `read-write`, `read-write-udf`, `sys-admin`, `user-admin`, `data-admin`, `truncate`):

```yaml
roles:
  - name: my-role
    privileges:
      - read-write.myns     # Valid
      # - admin.myns        # Invalid — use 'sys-admin' or 'data-admin'
```

### Checking ACL Status

After deployment, verify ACL is working:

```bash
# Check operator logs for ACL sync
kubectl -n aerospike-operator logs -l control-plane=controller-manager | grep -i acl

# Connect to the cluster with credentials
kubectl -n aerospike exec -it aerospike-acl-0-0 -- asadm -Uadmin -Padmin-password-here

# List users (inside asadm)
manage acl show users
```
