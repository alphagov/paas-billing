# Gherkin tests

## Scope

These tests are used currently to test the calculation of billing figures.

Because this is an MVP, a hard-coded username/password is used when running the tests.

## How to set up

1. Install Gherkin

In this directory run:

```
go get github.com/cucumber/godog/cmd/godog@v0.11.0
```

2. Download and set up a local Postgres database on your laptop.

3. Create the `billinguser` user/role with superuser permissions on your *local* Postgres instance

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
