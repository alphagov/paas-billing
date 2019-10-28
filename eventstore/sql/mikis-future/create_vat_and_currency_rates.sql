CREATE TABLE IF NOT EXISTS currency_rates(
  code text NOT NULL,
  valid_from timestamptz NOT NULL,
  rate numeric NOT NULL,
  sequence SERIAL,

  PRIMARY KEY (code, valid_from),
  CONSTRAINT rate_must_be_greater_than_zero CHECK (rate > 0)
);

CREATE VIEW currency_rates_with_duration AS (
  SELECT
    *,
    TSTZRANGE(
      valid_from,
      LEAD(valid_from, 1, 'infinity') OVER (
        PARTITION BY
          code
        ORDER BY
          valid_from
        ROWS BETWEEN current row AND 1 following
      )
    ) AS duration
  FROM
    currency_rates
);

CREATE TABLE IF NOT EXISTS vat_rates (
  code text NOT NULL,
  valid_from timestamptz NOT NULL,
  rate numeric NOT NULL,
  sequence SERIAL,

  PRIMARY KEY (code, valid_from),
  CONSTRAINT rate_must_be_greater_than_zero CHECK (rate >= 0),
  CONSTRAINT valid_from_start_of_month CHECK (
    (extract (day from valid_from)) = 1 AND
    (extract (hour from valid_from)) = 0 AND
    (extract (minute from valid_from)) = 0 AND
    (extract (second from valid_from)) = 0
  )
);

CREATE VIEW vat_rates_with_duration AS (
  SELECT
    *,
    TSTZRANGE(
      valid_from,
      LEAD(valid_from, 1, 'infinity') OVER (
        PARTITION BY
          code
        ORDER BY
          valid_from
        ROWS BETWEEN current row AND 1 following
      )
    ) AS duration
  FROM
    vat_rates
);
