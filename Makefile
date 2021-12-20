DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
TEST_DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
CF_API_ADDRESS ?= $(shell cf target | awk '/API endpoint/ {print $$3}')
APP_ROOT ?= $(PWD)

bin/paas-billing: clean
	go build -o $@ .

run-dev: bin/paas-billing run-dev-exports
	{ ./bin/paas-billing collector & ./bin/paas-billing api2 ; } | ./scripts/colorize

run-dev-collector: bin/paas-billing run-dev-exports
	./bin/paas-billing collector | ./scripts/colorize

run-dev-api: bin/paas-billing run-dev-exports
	./bin/paas-billing api2 | ./scripts/colorize

run-dev-exports:
	## Runs the application with local credentials
	$(eval export CF_API_ADDRESS=${CF_API_ADDRESS})
	$(eval export CF_CLIENT_ID=paas-billing)
#	$(eval export CF_CLIENT_SECRET=$(shell aws s3 cp s3://gds-paas-${DEPLOY_ENV}-state/cf-vars-store.yml - | awk '/uaa_clients_paas_billing_secret/ { print $$2 }')) # TODO: this can't work anymore since 4f513c2c121a4b29daf1b64d67e12512ec55b7b8 in paas-cf
	$(eval export CF_CLIENT_REDIRECT_URL=http://localhost:8881/oauth/callback)
	$(eval export CF_SKIP_SSL_VALIDATION=true)
	$(eval export DATABASE_URL=${DATABASE_URL})
	$(eval export APP_ROOT=${APP_ROOT})
	@true

.PHONY: test
test: fakes/fake_usage_api_client.go fakes/fake_cf_client.go fakes/fake_event_fetcher.go fakes/fake_event_store.go fakes/fake_authorizer.go fakes/fake_authenticator.go fakes/fake_billable_event_rows.go fakes/fake_usage_event_rows.go fakes/fake_cf_data_client.go
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	$(eval export APP_ROOT=${APP_ROOT})
	ginkgo $(ACTION) -nodes=8 -r $(PACKAGE) -skipPackage acceptance_tests

gherkin_test: gherkin_test_lon gherkin_test_ie

gherkin_test_lon:
	rm config.json || continue
	rm gherkin/features/* || continue
	mkdir -p gherkin/features
	cp ../paas-cf/config/billing/tests/eu-west-1_billing_rds_charges.feature gherkin/features/
	cp ../paas-cf/config/billing/output/eu-west-1.json config.json
	cd gherkin && godog

gherkin_test_ie:
	rm config.json || continue
	rm gherkin/features/* || continue
	mkdir -p gherkin/features
	cp ../paas-cf/config/billing/tests/eu-west-2_billing_rds_charges.feature gherkin/features/
	cp ../paas-cf/config/billing/output/eu-west-2.json config.json
	cd gherkin && godog

.PHONY: smoke
smoke:
	## Runs the api acceptance tests against a dev environment as a smoke test to check
	$(eval export CF_BEARER_TOKEN=$(shell cf oauth-token | cut -d' ' -f2))
	$(eval export BILLING_API_URL ?= http://127.0.0.1:8881)
	echo "smoke test enabled against ${BILLING_API_ADDRESS}"
	ginkgo  -focus=".*from api" -r acceptance_tests

.PHONY: acceptance
acceptance:
	$(eval export BILLING_API_URL ?= http://127.0.0.1:8881)
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	$(eval export CF_BEARER_TOKEN=$(shell cf oauth-token | cut -d' ' -f2))
	ginkgo -r acceptance_tests

fakes/fake_usage_api_client.go: eventfetchers/cffetcher/cf_client.go
	counterfeiter -o $@ $< UsageEventsAPI

fakes/fake_cf_client.go: eventfetchers/cffetcher/cf_client.go
	counterfeiter -o $@ $< UsageEventsClient

fakes/fake_event_fetcher.go: eventio/event_fetcher.go
	counterfeiter -o $@ $< EventFetcher

fakes/fake_event_store.go: eventio/*.go
	counterfeiter -o $@ $< EventStore

fakes/fake_authorizer.go: apiserver/auth/authorizer.go
	counterfeiter -o $@ $< Authorizer

fakes/fake_authenticator.go: apiserver/auth/authenticator.go
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
	docker run -p 5432:5432 --name postgres -e POSTGRES_HOST_AUTH_METHOD=trust -d postgres:12.5

stop_postgres_docker:
	docker stop postgres
	docker rm postgres
