# Billing API

## Overview

The "Billing API" allows us to query resource usage and calculate costs for
that usage.

* A materialised view called `billing` is created in the database that gets updated at a set interval (currently 1hr) which normalizes the raw event data into rows of `resource_guid`, `time period`, `pricing_plan_id`. The view caches and indexes data in the following form so that we can perform queries against it efficiently.

| ... | duration | guid | org | space | plan | memory_in_mb | ... |
| --- | --- | --- | --- | --- | --- | --- | --- |
| ... | `2017-11-01 08:14` to `2017-11-19 08:14` | {thing1} | {org1} | {space1} | {planA} | x | ... |
| ... | `2017-11-01 09:10` to `2017-11-19 10:10` | {thing2} | {org2} | {space1} | {planB} | y | ... |

* The `pricing_plans` and `pricing_plan_components` tables are added to the database which contains the formulas required to calculate costs for each resource.

For example a pricing_plan row might look like:

| id | name | valid_from | plan_guid |
| --- | --- | --- | --- | --- |
| 1 | compute instance | 2017-11-01 | {plan_guid} |
| 2 | tiny postgres | 2017-11-01 | {plan_guid} |

And a pricing_plan_components row might look like:

| id | pricing_plan_id | name | formula |
| --- | --- | --- | --- |
| 1 | 1 | Memory usage | `($memory_in_mb / 1000) * 0.33` |
| 2 | 1 | Runtime usage | `$time_in_seconds * 0.15` |

In this example the final formula for the `compute instance` plan would be:

```
($memory_in_mb / 1000) * 0.33 + $time_in_seconds * 0.15
```

* The API would join the rows from that view with data from the `pricing_plans` and `pricing_plan_components` tables which contains the information required to calculate the prices.

* Pricing plans can change over time so they have a valid_from field. The monetized calculation handles splitting usage over the valid ranges.

* In the pricing plans you can use the following functions:
    - **ceil**: converts to the nearest integer greater than or equal to argument. It can be used to calculate billable hours, e.g. `$time_in_seconds / 3600 * 1.5` will bill the tenants for 1.5 for every started hour.

* A REST/JSON API exposes aggregated data at several levels. Only guid details are returned in the data at the moment. If you want names you would need to call out the cf:
    - `/organisations` list totals for all orgs
    - `/organisations/:org_guid` list total for a single org
    - `/organisations/:org_guid/spaces` list totals for each space in an org
    - `/organisations/:org_guid/resources` list totals for each resource in org (a resource is a single thing - like an "app" or a "service")
    - `/spaces` list totals for all spaces
    - `/spaces/:space_guid` list total for a single space
    - `/spaces/:space_guid/resources` list totals for each resource in space
    - `/resources` list totals for all resources
    - `/resources/:resource_guid` list total for a single resource
    - `/events` list all events ("events" are all the start/stop points with calculated billing, unlike "resources" which are aggregate totals for each item over a range). Events would allow you to see _when_ something happens
    - `/resources/:resource_guid/events` as above but for a single resource
    - `/pricing_plans` fetch the pricing plans
    - `/report/:org_guid` generate a report for the given `:org_guid`
    - `/forecast/report` generate a forecast report for a given set of events

* A (throwaway) example HTML rendering of an aggregated report can be found at `/` you will be prompted to login via UAA. This is meant purely as an illustration of what is possible for now.

### Authorization

All API request require a valid token set in the `Authorization` header.

To get a valid token you must send a request to `/oauth/callback?code=[AUTH_CODE]&state=[AUTH_STATE]`

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
