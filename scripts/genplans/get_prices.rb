#!/usr/bin/env ruby

require 'json'
require 'yaml'
require 'amazon-pricing'
require 'hashie'

def plans(manifest, service)
  manifest.extend Hashie::Extensions::DeepFind
  manifest
    .deep_select('plans')
    .flatten.select { |p| p['rds_properties']['engine'] == service }
end

def paas_db_to_aws_db
  {
    'postgres' => :postgresql,
    'mysql' => :mysql
  }
end

def storage_cost_in_gb_month
  {
    postgresql: {
      true => 0.253,
      false => 0.127
    },
    mysql: {
      true => 0.253,
      false => 0.127
    }
  }
end

def calculate_for_db(db_name, prices, plans, manifest)
  plans(manifest, db_name).map do |broker_plan|
    broker_plan_id = broker_plan['id']
    service_plan = plans.detect { |p| p['entity']['unique_id'] == broker_plan_id }
    raise "no service plan for id #{broker_plan_id}:\n#{plan.to_yaml}" if service_plan.nil?
    db_instance_class = broker_plan['rds_properties']['db_instance_class']
    rds_instance_type = prices.rds_instance_types
                              .detect { |t| t.api_name == db_instance_class }
    is_multi_az = broker_plan['rds_properties']['multi_az']
    price = rds_instance_type.price_per_hour(
      paas_db_to_aws_db[db_name],
      :ondemand,
      _term = nil,
      is_multi_az
    )
    storage_cost = storage_cost_in_gb_month[paas_db_to_aws_db[db_name]][is_multi_az]
    {
      name: "#{db_name} #{broker_plan['name']}",
      valid_from: '2017-01-01',
      plan_guid: service_plan['metadata']['guid'],
      storage_in_mb: broker_plan['rds_properties']['allocated_storage'] * 1024,
      memory_in_mb: 0,
      components: [{
        name: 'instance',
        formula: "ceil($time_in_seconds/3600) * #{price}",
        currency_code: 'USD',
        vat_code: 'Standard'
      }, {
        name: 'storage',
        formula: "($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * #{storage_cost}",
        currency_code: 'USD',
        vat_code: 'Standard'
      }]
    }
  end
end

aws = AwsPricing::RdsPriceList.new
rds_price_list = aws.get_region('eu-west-1')

rds_broker_plans = YAML.load_file('050-rds-broker.yml')
service_plans = JSON.parse(File.read('prod-plans.json'))

app_prices = JSON.parse(File.read('app_prices.json'))
postgres_prices = calculate_for_db('postgres', rds_price_list, service_plans, rds_broker_plans)
mysql_prices = calculate_for_db('mysql', rds_price_list, service_plans, rds_broker_plans)
elasticache_prices = JSON.parse(File.read('elasticache_prices.json'))
cdn_prices = JSON.parse(File.read('cdn_prices.json'))
compose_prices = JSON.parse(File.read('compose_prices.json'))

all_prices = app_prices + postgres_prices + mysql_prices + elasticache_prices + cdn_prices + compose_prices

puts JSON.pretty_generate(all_prices)
