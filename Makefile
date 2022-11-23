DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
TEST_DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
CF_API_ADDRESS ?= $(shell cf target | awk '/api endpoint/ {print $$3}')
APP_ROOT ?= $(PWD)

bin/paas-billing: clean
	go build -o $@ .

run-dev: bin/paas-billing run-dev-exports
	{ ./bin/paas-billing collector & ./bin/paas-billing api ; } | ./scripts/colorize

run-dev-collector: bin/paas-billing run-dev-exports
	./bin/paas-billing collector | ./scripts/colorize

run-dev-api: bin/paas-billing run-dev-exports
	./bin/paas-billing api | ./scripts/colorize

run-dev-exports:
	## Runs the application with local credentials
	$(eval export CF_API_ADDRESS=${CF_API_ADDRESS})
	$(eval export CF_CLIENT_ID=paas-billing)
	$(eval export CF_CLIENT_SECRET=$(shell aws s3 cp s3://gds-paas-${DEPLOY_ENV}-state/cf-vars-store.yml - | awk '/uaa_clients_paas_billing_secret/ { print $$2 }'))
	$(eval export CF_CLIENT_REDIRECT_URL=http://localhost:8881/oauth/callback)
	$(eval export CF_SKIP_SSL_VALIDATION=true)
	$(eval export DATABASE_URL=${DATABASE_URL})
	$(eval export APP_ROOT=${APP_ROOT})
	@true

.PHONY: test
test: fakes/fake_usage_api_client.go fakes/fake_cf_client.go fakes/fake_event_fetcher.go fakes/fake_event_store.go fakes/fake_authorizer.go fakes/fake_authenticator.go fakes/fake_billable_event_rows.go fakes/fake_usage_event_rows.go fakes/fake_cf_data_client.go
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	$(eval export APP_ROOT=${APP_ROOT})
	go run github.com/onsi/ginkgo/v2/ginkgo $(ACTION) -nodes=8 -r $(PACKAGE) -skip-package=acceptance_tests

# .PHONY: gherkin_test
gherkin_test: gherkin_test_lon gherkin_test_ie

# .PHONY: gherkin_test_lon
gherkin_test_lon:
	mkdir -p gherkin/features
	cp ../paas-cf/config/billing/tests/eu-west-1_billing_rds_charges.feature gherkin/features/
	cp ../paas-cf/config/billing/output/eu-west-1.json config.json
	cd gherkin && go run github.com/cucumber/godog/cmd/godog run
	rm config.json
	rm gherkin/features/*

gherkin_test_ie:
	mkdir -p gherkin/features
	cp ../paas-cf/config/billing/tests/eu-west-2_billing_rds_charges.feature gherkin/features/
	cp ../paas-cf/config/billing/output/eu-west-2.json config.json
	cd gherkin && go run github.com/cucumber/godog/cmd/godog run
	rm config.json
	rm gherkin/features/*



.PHONY: smoke
smoke:
	## Runs the api acceptance tests against a dev environment as a smoke test to check
	$(eval export CF_BEARER_TOKEN=$(shell cf oauth-token | cut -d' ' -f2))
	$(eval export BILLING_API_URL ?= http://127.0.0.1:8881)
	echo "smoke test enabled against ${BILLING_API_ADDRESS}"
	go run github.com/onsi/ginkgo/v2/ginkgo  -focus=".*from api" -r acceptance_tests

.PHONY: integration
integration:
	#cd gherkin && go run github.com/cucumber/godog/cmd/godog run
	echo skip

.PHONY: acceptance
acceptance:
	$(eval export BILLING_API_URL ?= http://127.0.0.1:8881)
	$(eval export CF_BEARER_TOKEN=$(shell cf oauth-token | cut -d' ' -f2))
	go run github.com/onsi/ginkgo/v2/ginkgo -r acceptance_tests

fakes/fake_usage_api_client.go: eventfetchers/cffetcher/cf_client.go
	go run github.com/maxbrunsfeld/counterfeiter/v6 -o $@ $< UsageEventsAPI

fakes/fake_cf_client.go: eventfetchers/cffetcher/cf_client.go
	go run github.com/maxbrunsfeld/counterfeiter/v6 -o $@ $< UsageEventsClient

fakes/fake_event_fetcher.go: eventio/event_fetcher.go
	go run github.com/maxbrunsfeld/counterfeiter/v6 -o $@ $< EventFetcher

fakes/fake_event_store.go: eventio/*.go
	go run github.com/maxbrunsfeld/counterfeiter/v6 -o $@ $< EventStore

fakes/fake_authorizer.go: apiserver/auth/authorizer.go
	go run github.com/maxbrunsfeld/counterfeiter/v6 -o $@ $< Authorizer

fakes/fake_authenticator.go: apiserver/auth/authenticator.go
	go run github.com/maxbrunsfeld/counterfeiter/v6 -o $@ $< Authenticator

fakes/fake_billable_event_rows.go: eventio/event_billable.go
	go run github.com/maxbrunsfeld/counterfeiter/v6 -o $@ $< BillableEventRows

fakes/fake_usage_event_rows.go: eventio/event_usage.go
	go run github.com/maxbrunsfeld/counterfeiter/v6 -o $@ $< UsageEventRows

fakes/fake_cf_data_client.go: cfstore/cfstore_client.go
	go run github.com/maxbrunsfeld/counterfeiter/v6 -o $@ $< CFDataClient

clean:
	rm -f bin/paas-billing

start_postgres_docker:
	docker run -p 5432:5432 --name postgres -e POSTGRES_HOST_AUTH_METHOD=trust -d postgres:12.5

stop_postgres_docker:
	docker stop postgres
	docker rm postgres
