## API usage examples

### Authorization

All API request require a valid token set in the `Authorization` header.

To get a valid token you must sent a requst to `/oauth/callback?code=[AUTH_CODE]&state=[AUTH_STATE]`

To get `AUTH_CODE` and `AUTH_STATE` values you must authorize the app with UAA by visiting `/oauth/authorize` which will redirect you to login via UAA, and eventully give you a redirect to the callback url.

### Setup pricing plans

```
curl \
    -X POST \
    -H 'Accept: application/json'  \
    -H 'Authorization: Bearer ACCESS_TOKEN'  \
    --data '{"valid_from": "2010-01-01", "plan_guid": "MY_SERVICE_PLAN_GUID", "formula": "$time_in_seconds * 2", "name":"planA"}' \
    -H 'Content-Type: application/json' \
        'http://localhost:8881/pricing_plans'
```

Create a pricing plan for the "compute" service (used for calculating per instance costs). The GUID for this plan is hardcoded as `f4d4b95a-f55e-4593-8d54-3364c25798c4`.

```
curl -H 'Accept: application/json' -X POST --data '{"valid_from": "2010-01-01", "plan_guid": "f4d4b95a-f55e-4593-8d54-3364c25798c4", "formula": "$memory_in_mb * $time_in_seconds * 2", "name":"planA"}' -H 'Content-Type: application/json' 'http://localhost:8881/pricing_plans'
```

Query an endpoint with a range (times must be in ISO8601 format as below):

```
curl -vv -H 'Accept: application/json' 'http://localhost:8881/organisations?from=2010-01-01T00:00:00Z&to=2017-12-01T00:00:00Z' | jq .
```
