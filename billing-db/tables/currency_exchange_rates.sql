CREATE TABLE IF NOT EXISTS currency_exchange_rates
(
    from_ccy CHAR(3) NOT NULL, -- ISO currency code we're converting from
    to_ccy CHAR(3) NOT NULL, -- ISO currency code we're converting to
    valid_from timestamptz NOT NULL,
    valid_to timestamptz NOT NULL,
    rate NUMERIC NOT NULL,

    PRIMARY KEY (from_ccy, to_ccy, valid_to),
    CONSTRAINT rate_must_be_greater_than_zero CHECK (rate > 0)
);
