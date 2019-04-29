package eventstore

import (
	"encoding/json"

	"github.com/alphagov/paas-billing/eventio"
)

var _ eventio.CurrencyRateReader = &EventStore{}

func (s *EventStore) GetOrgs(filter eventio.EventFilter) ([]eventio.CurrencyRate, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := queryJSON(tx, `
        with
        valid_orgs as (
            select
                *,
                tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
                    partition by guid order by valid_from rows between current row and 1 following
                )) as valid_for
            from
                orgs
        )
        select
            vo.guid,
            vo.name,
            vo.valid_from,
            vo.created_at,
            vo.updated_at,
            vo.quota_definition_guid
        from
            valid_orgs vo
        where
            vo.valid_for && tstzrange($1, $2)
        group by
            vo.guid,
            vo.name,
            vo.valid_from,
            vo.created_at,
            vo.updated_at,
            vo.quota_definition_guid
        order by
            valid_from
    `, filter.RangeStart, filter.RangeStop)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orgs := []eventio.Org{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var org eventio.Org
		if err := json.Unmarshal(b, &org); err != nil {
			return nil, err
		}
		orgs = append(orgs, org)

	}
	return orgs, nil
}
