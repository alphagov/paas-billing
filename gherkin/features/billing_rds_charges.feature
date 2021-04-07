# Please specify the expected charge in pounds and pence only (e.g. £5.67), not as fractions of a penny (e.g. £5.671).
# Please also note the month needs to be specified in a 3 character format ('Jun 2020' not 'June 2020').

Feature: Test AWS RDS bills calculated correctly

  # Run before each scenario, including those in scenario outline.
  Background:
  Given a clean billing database

  Scenario: Test initial RDS charge
  Given a tenant has a postgres small-10.5 between '2020-01-05 00:00:00' and '2020-06-05 11:00'
  When billing is run for Jun 2020
  Then the charge, including VAT, should be £6.44

  Scenario Outline: Test Postgres RDS charge
  Given a tenant has a <postgres database> between '<start date>' and '<end date>'
  When billing is run for <month and year>
  Then the charge, including VAT, should be £<charge>

  Examples:
    | postgres database | start date | end date | month and year | charge |
    | postgres small-10.5 | 2020-01-05 | 2020-06-05 | Jun 2020 | 6.03 |
