#!/bin/bash
set -eu -o pipefail

COLLECTOR_URL=${COLLECTOR_URL:-https://paas-billing.cloudapps.digital}
report_url="${COLLECTOR_URL}/report"

usage() {
    cat <<"EOF"
Small hacky script generate the bills for a given month
for all the orgs that are NOT using the "default" quota.

It downloads the .json, .html and generates .pdf using wkhtmltopdf.
The managers of the org are stored in 'managers.txt'

NOTES:
    - First login into the target CF to get the token.
    - It expects that the pricing formulas ALREADY exist in the collector app
    - Override $COLLECTOR_URL to point to a non prod collector
    - Assumes you have docker installed to run wkhtmltopdf

Usage:
    ./bulk-pdf-bill-generation.sh YYYY-MM
    ./bulk-pdf-bill-generation.sh 2018-01

EOF
    exit 1
}


month=${1:-}
if [ -z "${month}" ] || ! echo ${month} | grep -qe '2[0-9][0-9][0-9]-[01][0-9]'; then
    usage
fi

date_from=$(date -d "${month}-01" "+%Y-%m-%dT00%%3A00%%3A00Z")
date_to=$(date -d "${month}-01 +1 month" "+%Y-%m-%dT00%%3A00%%3A00Z")

cf_token="$(cat ~/.cf/config.json  | jq .AccessToken -r)"

default_quota_guid="$(cf curl '/v2/quota_definitions?q=name%3Adefault' | jq .resources[].metadata.guid -r)"

for org_guid in $(cf curl  /v2/organizations?order-by=name  | \
    jq ".resources[] | select(.entity.quota_definition_guid | contains(\"$default_quota_guid\" ) | not) | .metadata.guid" -r); do

    org_name=$(cf curl /v2/organizations/${org_guid} | jq .entity.name -r)
    org_managers=$(cf curl /v2/organizations/${org_guid}/managers | jq -r  .resources[].entity.username)

    echo "Retrieving report for org: ${org_name} org_guid: ${org_guid} managers: $(echo ${org_managers} | xargs)"

    target_dir="reports/${org_name}"
    mkdir -p "${target_dir}"
    echo ${org_managers} > "${target_dir}/managers.txt"

    curl -qs "${report_url}/${org_guid}?from=${date_from}&to=${date_to}" -k \
        -H 'Accept: application/json' \
        -H "Authorization: ${cf_token}" \
        -o "${target_dir}/report_${month}.json"

    curl -qs "${report_url}/${org_guid}?from=${date_from}&to=${date_to}" -k \
        -H 'Accept: text/html' \
        -H "Authorization: ${cf_token}" \
        -o "${target_dir}/report_${month}.html"

    sed -i '/<form/,/<\/form/d' "${target_dir}/report_${month}.html"

    echo "Converting to PDF"
    docker run \
        -ti -v $(pwd):/workdir \
        openlabs/docker-wkhtmltopdf \
        --encoding utf-8 \
	--print-media-type \
        "/workdir/${target_dir}/report_${month}.html" \
        "/workdir/${target_dir}/report_${month}.pdf"
done


