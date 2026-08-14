#!/usr/bin/env sh
# Automated assessment of a specific authorized device by IP.
# Uses -y (non-interactive authorization). Requires ADB over TCP enabled
# on the device (adb tcpip 5555) and explicit authorization to assess it.
set -eu

ADDR="${1:?usage: assess_ip.sh <ip-address>}"
PROFILE="${PROFILE:-deep}"

mkdir -p reports
./bin/jabari assess ip "${ADDR}" -y --profile "${PROFILE}" -o json \
    > "reports/assessment-${ADDR}.json"