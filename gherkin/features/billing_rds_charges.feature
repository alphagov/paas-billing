Feature: Test AWS RDS bills calculated correctly

  Background:
  Given a clean billing database
  # (This is run before each scenario, including those in scenario outline.)

  Scenario: Test initial RDS bill
  -------------------------------

  Given a tenant has a postgres small-10.5 between '2020-01-05 00:00:00' and '2020-06-05 11:00'
  When billing is run for Jun 2020
  Then the bill, including VAT, should be £6.44

####

  Scenario Outline: Test Postgres RDS bill
  ----------------------------------------

  Given a tenant has a <postgres database> between '<start date>' and '<end date>'
  When billing is run for <month and year>
  Then the bill, including VAT, should be £<bill>

  Examples:
    | postgres database | start date | end date | month and year | bill |
    | postgres small-11 | 2021-03-01 | 2021-04-01 | Mar 2021 | 82.24 |

# Calculation from https://calculator.aws/#/createCalculator/RDSPostgreSQL:
# 1 instance(s) x 0.078 USD hourly x 730 hours in a month = 56.9400 USD
# 100 GB per month x 0.253 USD x 1 instances = 25.30 USD (Storage Cost)
# These charges exclude VAT
# postgres small-11 has storage_in_mb = 100GB and is an AWS RDS db.t3.small (from service_plans table)
# Note that "postgres small-11" is listed because this is the entry that appears in the pricing_plans table. The entry that appears in service_plans is "small-11".
