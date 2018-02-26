#!/bin/bash

set -eu -o pipefail

usage() {
    cat <<"EOF"
Small hacky script to push in batch all the pricing policies
from a pricing.json file.

NOTES:
    - First login into the target CF to get the token.
    - It expects that the pricing formulas ALREADY exist in the collector app
    - Override $COLLECTOR_URL to point to a non prod collector

Usage:
    ./push-pricing.sh <get|create|update> <pricing_plans.json> <pricing_plan_components.json>

To get the current pricing data run the command with the "get" subcommand first. E.g.

$ ./push-pricing.sh get pricing_plans.json pricing_plan_components.json

EOF
    exit 1
}

action=${1:-}
pricing_plans_file=${2:-}
pricing_plan_components_file=${3:-}
if [ -z "${action}" -o -z "${pricing_plans_file}" -o -z "${pricing_plan_components_file}" ]; then
    usage
fi

COLLECTOR_URL=${COLLECTOR_URL:-https://paas-billing.cloudapps.digital}
pricing_plans_url="${COLLECTOR_URL}/pricing_plans"
pricing_plan_components_url="${COLLECTOR_URL}/pricing_plan_components"

if [ "$action" = "get" ]; then
    curl --fail "${pricing_plans_url}" \
        -k -s \
        -H "Authorization: $(cat ~/.cf/config.json  | jq .AccessToken -r)" \
        -H 'Content-Type: application/json' -H 'Accept: application/json' \
        | jq -r "." > "${pricing_plans_file}"
    echo "Current plans saved to ${pricing_plans_file}"

    curl --fail "${pricing_plan_components_url}" \
        -k -s \
        -H "Authorization: $(cat ~/.cf/config.json  | jq .AccessToken -r)" \
        -H 'Content-Type: application/json' -H 'Accept: application/json' \
        | jq -r "." > "${pricing_plan_components_file}"
    echo "Current plan components saved to ${pricing_plan_components_file}"

    exit 0
fi

for i in $(cat "${pricing_plans_file}" | jq -r '.[].id' | sort -n); do
    data="$(cat "${pricing_plans_file}" | jq ".[] | select(.id | contains($i))")"
    case "${action}" in
        create)
            curl --fail "${pricing_plans_url}" \
                -k \
                -X POST \
                -H "Authorization: $(cat ~/.cf/config.json  | jq .AccessToken -r)" \
                -H 'Content-Type: application/json' -H 'Accept: application/json' \
                --data "$data"
            echo
            ;;
        update)
            curl --fail "${pricing_plans_url}/$i" \
                -k \
                -X PUT \
                -H "Authorization: $(cat ~/.cf/config.json  | jq .AccessToken -r)" \
                -H 'Content-Type: application/json' -H 'Accept: application/json' \
                --data "$data"
            echo
            ;;
        *)
            echo "Unknown action: $action"
            usage
            ;;
    esac
done

for i in $(cat "${pricing_plan_components_file}" | jq -r '.[].id' | sort -n); do
    data="$(cat "${pricing_plan_components_file}" | jq ".[] | select(.id | contains($i))")"
    case "${action}" in
        create)
            curl --fail "${pricing_plan_components_url}" \
                -k \
                -X POST \
                -H "Authorization: $(cat ~/.cf/config.json  | jq .AccessToken -r)" \
                -H 'Content-Type: application/json' -H 'Accept: application/json' \
                --data "$data"
            echo
            ;;
        update)
            curl --fail "${pricing_plan_components_url}/$i" \
                -k \
                -X PUT \
                -H "Authorization: $(cat ~/.cf/config.json  | jq .AccessToken -r)" \
                -H 'Content-Type: application/json' -H 'Accept: application/json' \
                --data "$data"
            echo
            ;;
        *)
            echo "Unknown action: $action"
            usage
            ;;
    esac
done
