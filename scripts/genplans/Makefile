prices.json: 050-rds-broker.yml prod-plans.json app_prices.json elasticache_prices.json cdn_prices.json
	bundle exec ./get_prices.rb > $@

clean:
	rm -f prices.json
