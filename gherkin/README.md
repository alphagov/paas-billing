# Gherkin tests

## Scope

These tests are used currently to test the calculation of billing figures.

Because this is an MVP, a hard-coded username/password is used when running the tests.

## How to set up

1. Install Gherkin

In this directory run:

```
go install github.com/cucumber/godog/cmd/godog@v0.11.0
```

2. Download and set up a local Postgres database on your laptop.

3. Create the `billinguser` user/role with superuser permissions on your *local* Postgres instance

Mac only:

On a mac you may need to run first

```
/usr/local/opt/postgres/bin/createuser -s postgres
```

Linux and Mac:

Log into Postgres as the `postgres` shell login (e.g. `sudo -u postgres psql`), then:

```
CREATE DATABASE billing;
CREATE USER billinguser WITH PASSWORD 'billinguser' SUPERUSER;
```


## How to run

Run the tests using Gherkin

```
make gherkin_test
```

This will copy the config and tests from `../paas-cf/billing/config/` for each region we deploy in and run the tests in turn. This is done each region has different costs and we need to test locally for the region we are in.


