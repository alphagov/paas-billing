DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
TEST_DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
CF_API_ADDRESS ?= https://api.${DEPLOY_ENV}.dev.cloudpipeline.digital
APP_ROOT ?= $(PWD)

bin/paas-billing: clean
	go build -o $@ .

run-dev: bin/paas-billing
	## Runs the application with local credentials
	$(eval export CF_API_ADDRESS=${CF_API_ADDRESS})
	$(eval export CF_CLIENT_ID=paas-billing)
	$(eval export CF_CLIENT_SECRET=$(shell aws s3 cp s3://gds-paas-${DEPLOY_ENV}-state/cf-secrets.yml - | awk '/uaa_clients_paas_billing_secret/ { print $$2 }'))
	$(eval export CF_CLIENT_REDIRECT_URL=http://localhost:8881/oauth/callback)
	$(eval export COMPOSE_API_KEY=$(shell aws s3 cp s3://gds-paas-${DEPLOY_ENV}-state/compose-secrets.yml - | awk '/compose_api_key/ {print $$2}'))
	$(eval export CF_SKIP_SSL_VALIDATION=true)
	$(eval export DATABASE_URL=${DATABASE_URL})
	$(eval export APP_ROOT=${APP_ROOT})
	./bin/paas-billing | ./scripts/colorize

.PHONY: test
test: fakes/fake_usage_api_client.go fakes/fake_cf_client.go fakes/fake_event_fetcher.go fakes/fake_compose_client.go fakes/fake_event_store.go fakes/fake_authorizer.go fakes/fake_authenticator.go fakes/fake_billable_event_rows.go fakes/fake_usage_event_rows.go fakes/fake_cf_data_client.go
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	$(eval export APP_ROOT=${APP_ROOT})
	ginkgo -nodes=8 -r

.PHONY: smoke
smoke:
	## Runs the app/blackbox tests against a dev environment as a smoke test to check
	$(eval export CF_API_ADDRESS=${CF_API_ADDRESS})
	$(eval export CF_CLIENT_ID=paas-billing)
	$(eval export CF_CLIENT_SECRET=$(shell aws s3 cp s3://gds-paas-${DEPLOY_ENV}-state/cf-secrets.yml - | awk '/uaa_clients_paas_billing_secret/ { print $$2 }'))
	$(eval export CF_CLIENT_REDIRECT_URL=http://localhost:8881/oauth/callback)
	$(eval export COMPOSE_API_KEY=$(shell aws s3 cp s3://gds-paas-${DEPLOY_ENV}-state/compose-secrets.yml - | awk '/compose_api_key/ {print $$2}'))
	$(eval export CF_SKIP_SSL_VALIDATION=true)
	$(eval export APP_ROOT=${APP_ROOT})
	$(eval export ENABLE_SMOKE_TESTS=true)
	$(eval export TEST_AUTH_TOKEN=$(shell cf oauth-token))
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	ginkgo -nodes=2 .
	

fakes/fake_usage_api_client.go: eventfetchers/cffetcher/cf_client.go
	counterfeiter -o $@ $< UsageEventsAPI

fakes/fake_cf_client.go: eventfetchers/cffetcher/cf_client.go
	counterfeiter -o $@ $< UsageEventsClient

fakes/fake_compose_client.go: eventfetchers/composefetcher/compose_client.go
	counterfeiter -o $@ $< ComposeClient

fakes/fake_event_fetcher.go: eventio/event_fetcher.go
	counterfeiter -o $@ $< EventFetcher

fakes/fake_event_store.go: eventio/event_store.go
	counterfeiter -o $@ $< EventStore

fakes/fake_authorizer.go: eventserver/auth/authorizer.go
	counterfeiter -o $@ $< Authorizer

fakes/fake_authenticator.go: eventserver/auth/authenticator.go
	counterfeiter -o $@ $< Authenticator

fakes/fake_billable_event_rows.go: eventio/event_billable.go
	counterfeiter -o $@ $< BillableEventRows

fakes/fake_usage_event_rows.go: eventio/event_usage.go
	counterfeiter -o $@ $< UsageEventRows

fakes/fake_cf_data_client.go: cfstore/cfstore_client.go
	counterfeiter -o $@ $< CFDataClient

clean:
	rm -f bin/paas-billing

start_postgres_docker:
	docker run --rm -p 5432:5432 --name postgres -e POSTGRES_PASSWORD= -d postgres:9.5

stop_postgres_docker:
	docker stop postgres
	docker rm postgres
