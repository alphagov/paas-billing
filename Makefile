DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
TEST_DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

run-dev: db/bindata.go ## Runs the application with local credentials
	$(eval export CF_API_ADDRESS=https://api.${DEPLOY_ENV}.dev.cloudpipeline.digital)
	$(eval export CF_USERNAME=admin)
	$(eval export CF_PASSWORD=$(shell aws s3 cp s3://gds-paas-${DEPLOY_ENV}-state/cf-secrets.yml - | awk '/uaa_admin_password/ { print $$2 }'))
	$(eval export CF_SKIP_SSL_VALIDATION=true)
	$(eval export DATABASE_URL=${DATABASE_URL})
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
