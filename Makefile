DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
TEST_DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
BILLING_API_PORT ?= 8881

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

run-dev: db/bindata.go ## Runs the application with local credentials
	$(eval export CF_API_ADDRESS=https://api.${DEPLOY_ENV}.dev.cloudpipeline.digital)
	$(eval export CF_CLIENT_ID=paas-usage-events-collector)
	$(eval export CF_CLIENT_SECRET=$(shell aws s3 cp s3://gds-paas-${DEPLOY_ENV}-state/cf-secrets.yml - | awk '/uaa_clients_paas_usage_events_collector_secret/ { print $$2 }'))
	$(eval export CF_CLIENT_REDIRECT_URL=http://localhost:8881/oauth/callback)
	$(eval export CF_SKIP_SSL_VALIDATION=true)
	$(eval export DATABASE_URL=${DATABASE_URL})
	$(eval export BILLING_API_PORT=${BILLING_API_PORT})
	go run main.go

.PHONY: test
test: ## Run all tests with race detector and coverage
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	./test.sh

.PHONY: quick-test
quick-test: db/bindata.go ## Run all the tests in parallel with fail-fast enabled
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	ginkgo -succinct -nodes=4 -failFast ./...

db/bindata.go: db/sql/*.sql db/migrate.go
	go generate ./db

.PHONY: generate-test-mocks
generate-test-mocks: ## Generates all test mocks
	mockgen -package mocks -destination mocks/cloudfoundry.go github.com/alphagov/paas-usage-events-collector/cloudfoundry Client,UsageEventsAPI
	mockgen -package mocks -destination mocks/db.go github.com/alphagov/paas-usage-events-collector/db SQLClient
	mockgen -destination=mocks/io.go -package mocks io ReadCloser
