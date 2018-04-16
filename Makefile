DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
TEST_DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
APP_ROOT ?= $(PWD)

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
	$(eval export APP_ROOT=${APP_ROOT})
	go run main.go

.PHONY: test
test:
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	$(eval export APP_ROOT=${APP_ROOT})
	ginkgo -nodes=6 -r

.PHONY: test
generate-mocks: store/fakes/fake_event_storer.go cloudfoundry/fakes/mock_client.go cloudfoundry/fakes/mock_usage_events_api.go cloudfoundry/fakes/mock_io.go cloudfoundry/fakes/fake_usage_events_api.go collector/fakes/fake_event_fetcher.go compose/fakes/fake_client.go
	echo "regenerating mocks"

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
	go generate collector/collector.go

compose/fakes/fake_client.go: compose/compose.go
	go generate compose/...

store/fakes/fake_event_storer.go: store/store.go
	go generate store/store.go
