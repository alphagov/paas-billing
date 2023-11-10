#!/usr/bin/env ruby
# encoding: utf-8

# In August 2023, the resume marker for billing was incorrect due to a previous fix.
# When we came to fix billing the cf events list didn't hold all the records we needed.
# This uses the auditor db and re-creates all the billing service usage events since Aug 31st.
#
# To test this:
#   cf conduit auditor-db-copy
#   export A_DB_URI="postgres://user:pass@host:port/dbname" from the above
#   export B_DB_URI="postgres://localhost:5432/billing"

require 'securerandom'
require 'active_record'
require 'pg'
require 'ruby-progressbar'

unless ENV['B_DB_URI'].include?('5432')
  puts "SOURCE: #{ENV['A_DB_URI']}"
  puts "TARGET: #{ENV['B_DB_URI']}"
  puts
  puts "Do you want to continue? Please type Y for Yes or any other key for No."
  user_input = gets.chomp.upcase

  if user_input == 'Y'
    puts "Continuing the script..."
    # The rest of your code
  else
    puts "Exiting the script..."
  end
end

ActiveRecord::Base.logger = Logger.new(STDOUT)
ActiveRecord::Base.logger.level = ENV['DEBUG'] ? Logger::DEBUG : Logger::INFO

class AuditorBase < ActiveRecord::Base
  self.abstract_class = true
end

class BillingBase < ActiveRecord::Base
  self.abstract_class = true
end

class CfAuditEvent < AuditorBase
end

class ServiceUsageEvent < BillingBase
end

def event_type(event)
  case event.event_type
  when 'audit.service_instance.start_create'
    'CREATED'
  when 'audit.service_instance.create'
    'CREATED'
  when 'audit.user_provided_service_instance.create'
    'CREATED'
  when 'audit.service_instance.delete'
    'DELETED'
  when 'audit.user_provided_service_instance.delete'
    'DELETED'
  when 'audit.service_instance.start_update'
    'UPDATED'
  when 'audit.service_instance.update'
    'UPDATED'
  when 'audit.user_provided_service_instance.update'
    'UPDATED'
  when 'audit.service_instance.purge'
    'DELETED'
  else
    raise "Unknown event type: #{event.event_type}"
  end
end

def service_plans(guid=nil)
  @service_plans ||= cf("service_plans")
  return @service_plans unless guid

  service_plans.find { |p| p["guid"] == guid }
end

def service_instances(guid=nil)
  @service_instances ||= cf("service_instances")
  return @service_instances unless guid

  service_instances.find { |p| p["guid"] == guid }
end

def service_offerings(guid=nil)
  @service_offerings ||= cf("service_offerings")
  return @service_offerings unless guid

  service_offerings.find { |p| p["guid"] == guid }
end

def service_brokers(guid=nil)
  @service_brokers ||= cf("service_brokers")
  return @service_brokers unless guid

  service_brokers.find { |p| p["guid"] == guid }
end

def space_name(event)
  @spaces ||= cf("spaces")
  space = @spaces.find { |s| s["guid"] == event.space_guid }

  return space["name"] if space

  CfAuditEvent.where(
    actee: event.space_guid,
  ).first.actee_name
end

def service_plan_guid(event)
  guid = event.metadata.dig("request", "relationships", "service_plan", "data", "guid")
  guid ||= event.metadata.dig("request", "service_plan_guid")

  if guid.nil?
    events = CfAuditEvent.unscoped.where( actee: event.actee ).order(created_at: :desc).all
    events.each do |event|
      guid = event.metadata.dig("request", "relationships", "service_plan", "data", "guid")
      guid ||= event.metadata.dig("request", "service_plan_guid")

      return guid if guid
    end

    billing_event = ServiceUsageEvent.where(
      "raw_message->>'service_instance_guid' = ?", event.actee
    ).order(created_at: :desc)&.first
    return billing_event.raw_message["service_plan_guid"] if billing_event

    @faulty_records << event
  end

  guid
end

def service_plan_name(event, plan_guid)
  service_plans(plan_guid)["name"]
end

def service_offering_guid(event, plan_guid)
  plan = service_plans(plan_guid)

  plan["relationships"]["service_offering"]["data"]["guid"]
rescue TypeError => e
  pp event
  puts e.message
  puts e.backtrace
  # exit
end

def service_offering_name(event, plan_guid)
  guid = service_offering_guid(event, plan_guid)
  service_offerings(guid)["name"]
rescue TypeError => e
  pp event
  puts e.message
  puts e.backtrace
  # exit
end

def service_broker_guid(event, plan_guid)
  offering_guid = service_offering_guid(event, plan_guid)
  offering = service_offerings(offering_guid)
  offering["relationships"]["service_broker"]["data"]["guid"]
rescue TypeError => e
  pp event
  puts e.message
  puts e.backtrace
  # exit
end

def service_broker_name(event, plan_guid)
  guid = service_broker_guid(event, plan_guid)
  service_brokers(guid)["name"]
end

def instance_type(event)
  raise unless %w[service_instance user_provided_service_instance].include?(event.actee_type)

  event.actee_type == 'service_instance' ? 'managed_service_instance' : 'user_provided_service_instance'
end

def cf(uri)
  json = JSON.parse(`cf curl '/v3/#{uri}?per_page=5000'`)["resources"]
end

AuditorBase.establish_connection(ENV['A_DB_URI'])
BillingBase.establish_connection(ENV['B_DB_URI'])

# If we are requiring this file we stop here
if __FILE__ == $0
  @faulty_records = []

  ServiceUsageEvent.transaction do
    events = CfAuditEvent
      .where("created_at > ?", '2023-08-31 13:45:57+00')
      .where("created_at <= ?", '2023-10-03 18:04:40+00')
      .where(
      event_type: [
        'audit.service_instance.start_create',
        'audit.service_instance.delete',
        'audit.service_instance.update',
        'audit.service_instance.purge',
        'audit.user_provided_service_instance.create',
        'audit.user_provided_service_instance.delete',
        'audit.user_provided_service_instance.update'
      ]
    ).where.not(
      "actor_name LIKE 'BACC%'"
    ).order(created_at: :desc)

    puts "Total count: #{events.count}"
    progress_bar = ProgressBar.create(total: events.length, format: '%a %bᗧ%i %p%% %t')

    i = 0
    events.each do |event|
      progress_bar.increment
      guid = SecureRandom.uuid

      sname = space_name(event)
      begin
        progress_bar.title = "Record #{i += 1} of #{events.length}"

        service_guid = nil
        service_plan_guid = nil
        service_label = nil
        service_plan_name = nil

        if !event.actee_type.include? "user_provided_service"
          service_plan_guid = service_plan_guid(event)
          service_guid      = service_offering_guid(event, service_plan_guid)
          service_label     = service_offering_name(event, service_plan_guid)
          service_plan_name = service_plan_name(event, service_plan_guid)
        end

        s = ServiceUsageEvent.new(
          guid: guid,
          created_at: event.created_at,
          raw_message: {
            "state": event_type(event),
            "org_guid": event.organization_guid,
            "space_guid": event.space_guid,
            "space_name": sname,
            "service_guid": service_guid,
            "service_label": service_label,
            "service_plan_guid": service_plan_guid,
            "service_plan_name": service_plan_name,
            "service_instance_guid": event.actee,
            "service_instance_name": event.actee_name,
            "service_instance_type": instance_type(event)
          }
        )

        if !event.actee_type.include? "user_provided_service"
          s.raw_message["service_broker_guid"] = service_broker_guid(event, service_plan_guid)
          s.raw_message["service_broker_name"] = service_broker_name(event, service_plan_guid)
        end

        s.save!
      rescue NoMethodError => e
        unless sname.match(/^(BACC|SMOKE|CATS|PERF)-/)
          puts
          pp event
          puts "NoMethodError: #{e.message}"
          puts e.backtrace
          puts
        end
      end
    end
  end

  @faulty_records.uniq!
  puts
  puts
  puts
  puts "Completed with #{@faulty_records.count} faulty records"
  puts
  pp @faulty_records
end
