help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

run-dev: ## Runs the application with local credentials
	$(eval export CF_API_ADDRESS=https://api.${DEPLOY_ENV}.dev.cloudpipeline.digital)
	$(eval export CF_USERNAME=admin)
	$(eval export CF_PASSWORD=$(shell aws s3 cp s3://gds-paas-${DEPLOY_ENV}-state/cf-secrets.yml - | awk '/uaa_admin_password/ { print $$2 }'))
	$(eval export CF_SKIP_SSL_VALIDATION=true)
	go run main.go

.PHONY: test
test: ## Runs all the tests
	go test $$(go list ./... | grep -v /vendor/)

.PHONY: generate-test-mocks
generate-test-mocks: ## Generates all test mocks
	mockgen -source=cloudfoundry/client.go -destination=cloudfoundry/mocks/client.go -package mocks
