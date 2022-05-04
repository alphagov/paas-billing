
What
===

This is a small script that checks whether the data in the cloud foundry database is similar to the data in the billing database about what services exist.

It does this by getting the past 2 days billing events and filtering the services from them. Then it gets current services that are running in cloud foundry that have been created at least 4 hours ago (to give them a chance to get into the billing database). It then compares the resource guids from the events to the the service resources. If there are any in the cloudfoundry set that have not been billed it will exit with an error. If there are greater than 5% in the billable events but not in the cloudfoundry set it will also error (there is a some need for this leeway as some services are transient, so no longer show up in the cloud foundry api output).

Why
===

The main failure mode it should find is if the billing collector stops running for some reason, this will pick it up and alert us to the problem.

However it should also be able to pick up if there are other errors. It does not look for an exact match as 1) things in the cloudfoundry db could be more up to date than the billing db as it take the billing db time top process everything that comes in and 2) things in the billing db are a historical record and if a service has shut down it would not show up in the cloud foundry db.

How
====

To run it against the a dev environment login to that environment and run:
```
BILLING_API_URL=https://billing.dev03.dev.cloudpipeline.digital CF_API_URL=https://api.dev03.dev.cloudpipeline.digital CF_BEARER_TOKEN=$(cf oauth-token | cut -d' ' -f 2) go run cf_billing_reconcilliation.go
```
