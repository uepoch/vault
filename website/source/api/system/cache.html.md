---
layout: "api"
page_title: "/sys/cache - HTTP API"
sidebar_current: "docs-http-system-cache"
description: |-
  The `/sys/cache` endpoint is used to access the cache layer of the physical backend.
---

# `/sys/cache`

The `/sys/cache` endpoint is used to access the cache layer of the physical backend.

This endpoint throws an error in case of disabled cache layer.

## Read Cache entry

This endpoint reads the encrypted value of the key stored in the cache at the given path. This is the raw path
in the storage backend and not the logical path that is exposed via the mount
system.

**This endpoint requires 'sudo' capability.**

| Method   | Path                         | Produces               |
| :------- | :--------------------------- | :--------------------- |
| `GET`    | `/sys/cache/:path`             | `200 application/json` |

### Parameters

- `path` `(string: <required>)` – Specifies the raw path in the storage backend.
  This is specified as part of the URL.

### Sample Request

```
$ curl \
    ---header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/sys/cache/secret/foo
```

### Sample Response

```json
{
  "value": "{'foo':'bar'}"
}
```
## List Raw

This endpoint returns a list keys for a given path prefix. Prefix can be empty

**This endpoint requires 'sudo' capability.**

| Method   | Path                         | Produces               |
| :------- | :--------------------------- | :--------------------- |
| `LIST`   | `/sys/raw/:prefix` | `200 application/json` |
| `GET`   | `/sys/raw/:prefix?list=true` | `200 application/json` |


### Sample Request

```
$ curl \
    --header "X-Vault-Token: ..." \
    --request LIST \
    http://127.0.0.1:8200/v1/sys/cache/core
```

### Sample Response

```json
{
  "data":{
    "keys":[
      "core/audit",
      "core/auth",
      "core/local-auth",
      "..."
    ]
  }
}
```

## Purge cache

This endpoint purge the whole cache layer.
This can have performance impact on Vault's response time while it's rebuilding the cache.

| Method   | Path                         | Produces               |
| :------- | :--------------------------- | :--------------------- |
| `DELETE` | `/sys/cache/`             | `204 (empty body)`     |

### Sample Request

```
$ curl \
    --header "X-Vault-Token: ..." \
    --request DELETE \
    http://127.0.0.1:8200/v1/sys/cache
```
