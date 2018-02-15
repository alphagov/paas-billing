# paas-billing

A Golang application for polling the CloudFoundry API for app and service usage events, storing them in Postgres, and querying them.

Cloud Foundry docs on usage events: https://docs.cloudfoundry.org/running/managing-cf/usage-events.html

The application periodically fetches the latest app and service usage events from the following endpoints:
 * /v2/app_usage_events - [API docs](http://apidocs.cloudfoundry.org/272/app_usage_events/list_all_app_usage_events.html)
 * /v2/service_usage_events - [API docs](https://apidocs.cloudfoundry.org/272/service_usage_events/list_service_usage_events.html)

The API endpoints are paginated and we are using the *after_guid* parameter to get the next batch of events. We are not processing the latest events (COLLECTOR_RECORD_MIN_AGE) as these can be incomplete because events are only present if the related transactions are finished. (The events are ordered as they are started.)

Note: The API lists the events in order but the timestamps on events should not be used to sequence events (local clock issue in a multi-server environment).

## Configuration

The application handles the following environment variables:

 * **DATABASE_URL**: Postgres connection string (postgres://user:pass@host:port)
 * **CF_API_ADDRESS**: Cloud Foundry API endpoint
 * **CF_CLIENT_ID**: Cloud Foundry client id
 * **CF_CLIENT_SECRET**: Cloud Foundry client secret
 * **CF_CLIENT_REDIRECT_URL**: OAuth authorization callback url (must match what is specified in uaa client configuration)
 * CF_USERNAME: Cloud Foundry username
 * CF_PASSWORD: Cloud Foundry password
 * CF_SKIP_SSL_VALIDATION: whether the API client should skip the SSL certificate validation (use only for development!)
 * CF_TOKEN: Cloud Foundry OAuth token
 * CF_USER_AGENT: user agent when connecting to Cloud Foundry
 * COLLECTOR_DEFAULT_SCHEDULE (def: 1m): how often to fetch new data from the API
 * COLLECTOR_MIN_WAIT_TIME (def: 3s): if we are able to fetch the maximum number of items we only wait this much before the next fetch (this allows us to speed up the the processing if necessary)
 * COLLECTOR_FETCH_LIMIT (def: 50): how many items to fetch from the API in one request, must be a positive integer. Max: 100.
 * COLLECTOR_RECORD_MIN_AGE (def: 5m): stop processing records from the API if a record is found with less than a minimum age. This guarantees that we don't miss events from ongoing transactions.

**Variables in bold are required.**

**Note**: in development you can use CF_USERNAME/CF_PASSWORD instead of CF_CLIENT_ID/CF_CLIENT_SECRET for authentication.

## Dependency handling

We use [dep](https://github.com/golang/dep) for dependecy management. If you start to use a new 3rd party library please update your dependencies with ```dep ensure```.

## Database table structure

App and service events are stored respectively in app_usage_events and service_usage_events tables.

The tables have the following fields:
 * **id**: sequence to sort events ,
 * **guid**: unique char(36) event identifier
 * **created_at**: creation time
 * **raw_message**: a jsonb field to store the value of the *entity* field from the API response

## Testing

Use the provided **test** make target.

```
make test
```

### Regenerating the mocks

The tests are using generated mocks created with [gomock](https://github.com/golang/mock). If you change any of the interfaces you have to regenerate the mocks.

Due to [golang/mock#138](https://github.com/golang/mock/issues/138) and [golang/mock#95](https://github.com/golang/mock/issues/95) you will need to install the same version of `gomock` as vendored in this project:

    GOMOCK_VERSION=$(dep status | awk '/github.com\/golang\/mock/ {print $2}')
    go get -d github.com/golang/mock && \
    git -C ${GOPATH}/src/github.com/golang/mock checkout "${GOMOCK_VERSION}" && \
    go install github.com/golang/mock/mockgen

Then regenerate the mocks with the **generate-test-mocks** make target:

    make generate-test-mocks

## Run application locally

### Create a temporary Postgres server

Locally you can use a container for Postgres with [Docker for Mac](https://docs.docker.com/docker-for-mac/) or [Docker for Linux](https://docs.docker.com/engine/installation/linux/ubuntu/):

```
docker run -p 5432:5432 --name postgres -e POSTGRES_PASSWORD= -d postgres:9.5

# Clean up after
docker rm -f postgres
```

If you want to use a different database you should set the **DATABASE_URL** environment variable to your connection string.

### Run the application

You can use the provided **run-dev** make target:

```
make run-dev
```

The task will extract the necessary secrets for Cloud Foundry and set up the important environment variables.
