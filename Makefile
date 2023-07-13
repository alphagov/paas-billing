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
PROXYMETRICS_LISTEN_PORT ?= 8882
PROXYMETRICS_LISTENER := $(LISTEN_HOST):$(PROXYMETRICS_LISTEN_PORT)


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
	$(info Running collector on $(COLLECTOR_LISTENER), api on $(API_LISTENER), metrics proxy on $(PROXYMETRICS_LISTENER))
	$(MAKE) -j3 run-dev-collector run-dev-api run-dev-proxy-metrics

.PHONY: run-dev-collector
run-dev-collector: dev-prerequisites
	$(eval export VCAP_APPLICATION={"application_id":"1285957d-3111-48c4-bc08-0eb8695b8cac","application_name":"paas-billing-collector","application_version":"6f5abc5c-1e2b-4f59-b64c-9bcbfb0c401a","instance_id":"4f628ba0-f37c-44d6-8c54-9f3b013afa11","instance_index":0,"organization_id":"a0caff22-cc01-4370-bc36-592a4aba2dfd","organization_name":"admin","process_id":"16a7ea94-a296-4025-b96c-90ebd1433495","process_type":"web","space_id":"efdd5187-536e-49d8-8b8c-55436515ada6","space_name":"billing"})
	$(eval export PORT=$(COLLECTOR_LISTEN_PORT))
	./bin/paas-billing collector | ./scripts/colorize

.PHONY: run-dev-api
run-dev-api: dev-prerequisites
	$(eval export VCAP_APPLICATION={"application_id":"17f5830c-01df-4201-bc66-d33f4b1b2244","application_name":"paas-billing-api","application_version":"461feda8-03ba-4ede-a5b5-04524147c621","instance_id":"0c59e331-e446-4ec2-aced-432b44397231","instance_index":0,"organization_id":"a0caff22-cc01-4370-bc36-592a4aba2dfd","organization_name":"admin","process_id":"22e75a85-cb20-4a51-806a-6e0aa5901364","process_type":"web","space_id":"efdd5187-536e-49d8-8b8c-55436515ada6","space_name":"billing"})
	$(eval export PORT=$(API_LISTEN_PORT))
	./bin/paas-billing api | ./scripts/colorize

.PHONY: run-proxy-metrics
run-dev-proxy-metrics: dev-prerequisites
	$(eval export VCAP_APPLICATION=$(shell cf curl "/v3/apps/$$(cf app paas-billing-metrics-proxy --guid)/env" | jq -c '.application_env_json.VCAP_APPLICATION'))
	$(eval export APP_NAMES=paas-billing-api,paas-billing-collector)
	$(eval export PORT=$(PROXYMETRICS_LISTEN_PORT))

	./bin/paas-billing proxymetrics | ./scripts/colorize

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

.PHONY: gherkin_test_ie
gherkin_test_ie:
	mkdir -p gherkin/features
	cp ../paas-cf/config/billing/tests/eu-west-1_billing_rds_charges.feature gherkin/features/
	cp ../paas-cf/config/billing/output/eu-west-1.json config.json
	cd gherkin && go run github.com/cucumber/godog/cmd/godog run
	rm config.json
	rm gherkin/features/*

.PHONY: gherkin_test_lon
gherkin_test_lon:
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
	$(eval export CF_ADMIN_BEARER_TOKEN ?= $(shell cf oauth-token | cut -d' ' -f2))
	$(eval export BILLING_API_URL ?= http://$(API_LISTENER))
	echo "smoke test enabled against ${BILLING_API_URL}"
	go run github.com/onsi/ginkgo/v2/ginkgo --label-filter="smoke" -r acceptance_tests

.PHONY: acceptance
acceptance:
	$(eval export CF_ADMIN_BEARER_TOKEN ?= $(shell cf oauth-token | cut -d' ' -f2))
	$(eval export BILLING_API_URL ?= http://$(API_LISTENER))
	$(eval export METRICSPROXY_API_URL ?= http://$(PROXYMETRICS_LISTENER))
	go run github.com/onsi/ginkgo/v2/ginkgo --focus "Returns events for multi-org requests" -v -r acceptance_tests


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
