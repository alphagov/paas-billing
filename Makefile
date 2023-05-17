DOCKER_POSTGRES_PORT ?= 15432
DATABASE_URL ?= postgres://postgres:@localhost:$(DOCKER_POSTGRES_PORT)/?sslmode=disable
TEST_DATABASE_URL ?= postgres://postgres:@localhost:$(DOCKER_POSTGRES_PORT)/?sslmode=disable

CFCLI := cf
CF_API_ADDRESS ?= $(shell $(CFCLI) target | awk '/API endpoint/ {print $$3}')
APP_ROOT ?= $(PWD)

LISTEN_HOST ?= 127.0.0.1
COLLECTOR_LISTEN_PORT ?= 8880
COLLECTOR_LISTENER := $(LISTEN_HOST):$(COLLECTOR_LISTEN_PORT)
API_LISTEN_PORT ?= 8881
API_LISTENER := $(LISTEN_HOST):$(API_LISTEN_PORT)


LOG_LEVEL ?= info


.PHONY: .cf-is-authenticated
.cf-is-authenticated:
	$(if $(shell $(CFCLI) orgs >/dev/null 2>&1 && echo "OK"),\
		@true,\
		$(error cf cli is not logged in. Use cf login --sso to log in)\
	)

.PHONY: .cf-is-dev
.cf-is-dev: .cf-is-authenticated
	$(if $(patsubst %dev.cloudpipeline.digital,,$(CF_API_ADDRESS)),\
	  	$(error Not logged into a dev env. Current target: $(CF_API_ADDRESS)),\
		@true)

.PHONY: .cf-target-admin-billing
.cf-target-admin-billing:
	@$(CFCLI) target -o admin -s billing > /dev/null

bin/paas-billing: clean
	go build -o $@ .

config.json: ci/example_config.json
	@cp $< $@

.PHONY: dev-prerequisites
dev-prerequisites: run-dev-exports config.json bin/paas-billing

.PHONY: run-dev
run-dev:
	$(info Running collector on $(COLLECTOR_LISTENER), api on $(API_LISTENER))
	$(MAKE) -j2 run-dev-collector run-dev-api

.PHONY: run-dev-collector
run-dev-collector: dev-prerequisites
	$(eval export PORT=$(COLLECTOR_LISTEN_PORT))
	./bin/paas-billing collector | ./scripts/colorize

.PHONY: run-dev-api
run-dev-api: dev-prerequisites
	$(eval export PORT=$(API_LISTEN_PORT))
	./bin/paas-billing api | ./scripts/colorize

.PHONY: run-dev-exports
run-dev-exports: .cf-is-dev .cf-target-admin-billing
	## Runs the application with local credentials
	$(eval export CF_API_ADDRESS=${CF_API_ADDRESS})
	$(eval export CF_CLIENT_ID=paas-billing)
	$(eval export CF_CLIENT_SECRET=$(shell cf env paas-billing-collector | awk '/CF_CLIENT_SECRET:/ { print $$2 }'))
	$(eval export CF_CLIENT_REDIRECT_URL=http://localhost:8881/oauth/callback)
	$(eval export CF_SKIP_SSL_VALIDATION=true)
	$(eval export DATABASE_URL=${DATABASE_URL})
	$(eval export APP_ROOT=${APP_ROOT})
	$(eval export LISTEN_HOST=${LISTEN_HOST})
	$(eval export LOG_LEVEL=${LOG_LEVEL})
	@true

.PHONY: test
test: fakes
	$(eval export TEST_DATABASE_URL=${TEST_DATABASE_URL})
	$(eval export APP_ROOT=${APP_ROOT})
	go run github.com/onsi/ginkgo/v2/ginkgo $(ACTION) -nodes=8 -r $(PACKAGE) -skip-package=acceptance_tests


.PHONY: gherkin_test
gherkin_test: gherkin_test_lon gherkin_test_ie

.PHONY: gherkin_test_lon
gherkin_test_lon:
	mkdir -p gherkin/features
	cp ../paas-cf/config/billing/tests/eu-west-1_billing_rds_charges.feature gherkin/features/
	cp ../paas-cf/config/billing/output/eu-west-1.json config.json
	cd gherkin && go run github.com/cucumber/godog/cmd/godog run
	rm config.json
	rm gherkin/features/*

.PHONY: gherkin_test_ie
gherkin_test_ie:
	mkdir -p gherkin/features
	cp ../paas-cf/config/billing/tests/eu-west-2_billing_rds_charges.feature gherkin/features/
	cp ../paas-cf/config/billing/output/eu-west-2.json config.json
	cd gherkin && go run github.com/cucumber/godog/cmd/godog run
	rm config.json
	rm gherkin/features/*

.PHONY: integration
integration:
	cd gherkin && go run github.com/cucumber/godog/cmd/godog run

.PHONY: smoke
smoke:
	## Runs the api acceptance tests against a dev environment as a smoke test to check
	$(eval export CF_BEARER_TOKEN=$(shell cf oauth-token | cut -d' ' -f2))
	$(eval export BILLING_API_URL ?= http://$(API_LISTENER))
	echo "smoke test enabled against ${BILLING_API_URL}"
	go run github.com/onsi/ginkgo/v2/ginkgo --label-filter="smoke" -r acceptance_tests

.PHONY: acceptance
acceptance:
	$(eval export CF_BEARER_TOKEN=$(shell cf oauth-token | cut -d' ' -f2))
	$(eval export BILLING_API_URL ?= http://$(API_LISTENER))
	$(eval export METRICSPROXY_API_URL ?= http://$(PROXYMETRICS_LISTENER))
	go run github.com/onsi/ginkgo/v2/ginkgo -r acceptance_tests


.PHONY: fakes
fakes:
	go generate ./...

.PHONY: clean
clean:
	rm -f bin/paas-billing

.PHONY: start_postgres_docker
start_postgres_docker:
	docker run --rm -p $(DOCKER_POSTGRES_PORT):5432 --name paas-billing-postgres -e POSTGRES_HOST_AUTH_METHOD=trust -d postgres:12.5 -N 500

.PHONY: stop_postgres_docker
stop_postgres_docker:
	docker stop paas-billing-postgres
