DROP TABLE IF EXISTS currency_exchange_rates;

CREATE TABLE currency_exchange_rates
(
    from_ccy CHAR(3) NOT NULL, -- ISO currency code we're converting from
    to_ccy CHAR(3) NOT NULL, -- ISO currency code we're converting to
    valid_from timestamptz NOT NULL,
    valid_to timestamptz NOT NULL,
    rate NUMERIC NOT NULL,

    PRIMARY KEY (from_ccy, to_ccy, valid_to),
    CONSTRAINT rate_must_be_greater_than_zero CHECK (rate > 0),
    CONSTRAINT rate_one_when_same_currency CHECK ((from_ccy != to_ccy) OR (from_ccy = to_ccy AND rate = 1))
);
