DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
TEST_DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
DATABASE_SCHEMA_DIR ?= $(PWD)/schema/sql

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

run-dev:
	## Runs the application with local credentials
	$(eval export CF_API_ADDRESS=https://api.${DEPLOY_ENV}.dev.cloudpipeline.digital)
	$(eval export CF_CLIENT_ID=paas-billing)
	$(eval export CF_CLIENT_SECRET=$(shell aws s3 cp s3://gds-paas-${DEPLOY_ENV}-state/cf-secrets.yml - | awk '/uaa_clients_paas_billing_secret/ { print $$2 }'))
	$(eval export CF_CLIENT_REDIRECT_URL=http://localhost:8881/oauth/callback)
	$(eval export COMPOSE_API_KEY=$(shell aws s3 cp s3://gds-paas-${DEPLOY_ENV}-state/compose-secrets.yml - | awk '/compose_api_key/ {print $$2}'))
	$(eval export CF_SKIP_SSL_VALIDATION=true)
	$(eval export DATABASE_URL=${DATABASE_URL})
	$(eval export DATABASE_SCHEMA_DIR=${DATABASE_SCHEMA_DIR})
	go run main.go

.PHONY: test
test: cloudfoundry/mocks/cloudfoundry.go cloudfoundry/mocks/io.go cloudfoundry/fakes/fake_usage_events_api.go collector/fakes/fake_event_fetcher.go compose/fakes/fake_client.go
	## Run all tests with race detector and coverage
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	$(eval export DATABASE_SCHEMA_DIR=${DATABASE_SCHEMA_DIR})
	./test.sh

.PHONY: quick-test
quick-test: cloudfoundry/fakes/mock_client.go cloudfoundry/fakes/mock_usage_events_api.go cloudfoundry/fakes/mock_io.go cloudfoundry/fakes/fake_usage_events_api.go collector/fakes/fake_event_fetcher.go compose/fakes/fake_client.go
	## Run all the tests in parallel with fail-fast enabled
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	ginkgo -succinct -nodes=4 -failFast ./...

cloudfoundry/fakes/mock_usage_events_api.go: cloudfoundry/usage_events_api.go
	mkdir -p cloudfoundry/fakes
	mockgen -package fakes -destination=$@ github.com/alphagov/paas-billing/cloudfoundry UsageEventsAPI
	sed -i.bak 's#github.com/alphagov/paas-billing/vendor/##g' $@
	rm -f $@.bak

cloudfoundry/fakes/mock_client.go: cloudfoundry/client.go
	mkdir -p cloudfoundry/fakes
	mockgen -package fakes -destination=$@ github.com/alphagov/paas-billing/cloudfoundry Client
	sed -i.bak 's#github.com/alphagov/paas-billing/vendor/##g' $@
	rm -f $@.bak

cloudfoundry/fakes/mock_io.go:
	mkdir -p cloudfoundry/fakes
	mockgen -package fakes -destination=$@ -package fakes io ReadCloser

cloudfoundry/fakes/fake_usage_events_api.go: cloudfoundry/usage_events_api.go
	go generate cloudfoundry/...

collector/fakes/fake_event_fetcher.go: collector/collector.go
	go generate collector/...

compose/fakes/fake_client.go: compose/compose.go
	go generate compose/...
