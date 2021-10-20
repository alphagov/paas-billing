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

####

  Scenario: Test bill during the month RDS is upgraded
  ----------------------------------------------------

  Given a tenant has a postgres xlarge-ha-9.5 between '2018-11-01 00:00:00' and '2020-06-07 00:59'
  And the tenant has a postgres xlarge-ha-10.5 between '2020-06-07 00:59' and '2020-06-07 01:59'
  And the tenant has a postgres xlarge-ha-11 high-iops between '2020-06-07 01:59' and '2020-12-01'

  When billing is run for Jun 2020

  Then the bill, including VAT, should be £4380.43

# Calculation for the above:
# Note that $3.224 is the price per hour but bills are calculated to the second
# Calculated cost in USD between '2018-11-01 00:00:00' and '2020-06-07 00:59' (instance followed by storage) is ((521940)*(3.224/3600)) + ((521940/2678400)*((0.253*(2097152/1024)))) = 568.39702509
# Calculated cost in USD between '2020-06-07 00:59' and '2020-06-07 01:59' (instance followed by storage) is ((3600)*(3.224/3600)) + ((3600/2678400)*((0.253*(2097152/1024)))) = 3.920430108
# Check the storage cost formula (https://aws.amazon.com/rds/postgresql/pricing/) - need multi-AZ charge ($0.253) because HA
# Calculated cost in USD between '2020-06-07 01:59' and '2020-07-01 00:00' (instance followed by storage) is ((2066460)*(3.152/3600)) + ((2066460/2678400)*((0.253*(10485760/1024)))) = 3808.112977778
# Total cost in GBP, excluding VAT = 0.8*(568.39702509 + 3.920430108 + 3808.112977778) = £3504.344346381
# Total cost in GBP, including VAT = (0.8*(568.39702509 + 3.920430108 + 3808.112977778))*1.25 = £4380.430432976
# Note: the AWS charges exclude VAT
# References for calculation: see comparison chart above. Also https://aws.amazon.com/rds/previous-generation/ and https://aws.amazon.com/rds/postgresql/pricing/
# You can get the number of seconds between two dates easily using `select extract(epoch from ('2020-06-07 00:59:00'::timestamp - '2020-01-01 00:00:00'::timestamp));`
