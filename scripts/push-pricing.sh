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

WARNING:
    - This script might break the collector, by creating a deadlock. See
      story https://www.pivotaltracker.com/story/show/154395415 for more
      info. Solution: Wait a while, restart collector.

Usage:
    ./push-pricing.sh <create|update> <princing_file.json>

To get the current pricing json:

    COLLECTOR_URL=https://paas-billing.cloudapps.digital
    curl ${COLLECTOR_URL}/pricing_plans -k \
        -H 'Accept: application/json' \
        -H "Authorization: $(cat ~/.cf/config.json  | jq .AccessToken -r)" \
        -o pricing.json
EOF
    exit 1
}

action=${1:-}
pricing_file=${2:-}
if [ -z "${action}" -o -z "${pricing_file}" ]; then
    usage
fi

COLLECTOR_URL=${COLLECTOR_URL:-https://paas-billing.cloudapps.digital}
pricing_plans_url="${COLLECTOR_URL}/pricing_plans"

for i in $(cat "${pricing_file}" | jq -r '.[].id' | sort -n); do
    data="$(cat "${pricing_file}" | jq ".[] | select(.id | contains($i))")"
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
