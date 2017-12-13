# db_repair

## Overview


Events that occurred before usage event collection began will be missing and
it's possible that no events have been seen due to very long running
services/apps. `db_repair` is a tool for synthesising STARTED/CREATED events to
ensure we have a consistent view of the resource usage.

The following cases will cause a synthetic STARTED/CREATED event to be inserted into the database:

* No recorded event for an app we know is running in cf
* No recorded event for a service we know is running in cf
* First recorded event for an app is STOPPED
* First recorded event for a service is DELETED

The following cases are intentionally ignored:

* If we saw a service UPDATED event but without a preceding CREATED: this is
ignored as we do not know which plan the service would have been on.
* If we saw an app update event (STARTED with previous state STARTED):
this is ignored as we have prioritised cases with greater financial impact.

All synthetic events are created with an `id` of `0`. This means it's possible
to reverse the process safely by deleting all `app_usage_events` and
`service_usage_events` with `id = 0`.

## Build

```
go build -o db_repair
```

## Configuration

Configuation is via environment variables.

| Name | required | Description |
|---|---|---|
| `DATABASE_URL` | yes | Postgres connection string (postgres://user:pass@host:port) |
| `CF_API_ADDRESS` | yes | Cloud Foundry API endpoint |
| `CF_TOKEN` | yes | Cloud Foundry OAuth token with `cloud_controller.admin` or `cloud_controller.admin_read_only` scope |
| `CF_SKIP_SSL_VALIDATION` | - | Allow insecure connections |

## Usage

```
$ db_repair -h

Usage of db_repair:
  -dry-run
    	Do not commit database transaction
  -purge-fake-events
    	Delete all previously created fake events

```

## Example

```
DATABASE_URL='postgres://postgres:@localhost/postgres?sslmode=disable' \
CF_TOKEN=$(cf oauth-token | awk '{ print $2 }') \
CF_API_ADDRESS=https://api.${DEPLOY_ENV}.dev.cloudpipeline.digital \
CF_SKIP_SSL_VALIDATION=true \
./db_repair --dry-run
```

