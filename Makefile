DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
TEST_DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
DATABASE_SCHEMA_DIR ?= $(PWD)/db/sql

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

run-dev: ## Runs the application with local credentials
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
test: ## Run all tests with race detector and coverage
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	$(eval export DATABASE_SCHEMA_DIR=${DATABASE_SCHEMA_DIR})
	./test.sh

.PHONY: quick-test
quick-test: ## Run all the tests in parallel with fail-fast enabled
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	ginkgo -succinct -nodes=4 -failFast ./...

.PHONY: generate-test-mocks
generate-test-mocks: ## Generates all test mocks
	mockgen -package mocks -destination mocks/cloudfoundry.go github.com/alphagov/paas-billing/cloudfoundry Client,UsageEventsAPI
	mockgen -package mocks -destination mocks/db.go github.com/alphagov/paas-billing/db SQLClient
	mockgen -destination=mocks/io.go -package mocks io ReadCloser
	go generate ./...
