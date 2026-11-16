#!/bin/bash
# Start the API for the web front end against Temporal Cloud (settings in setcloudenv.sh),
# at http://127.0.0.1:8000.
cd "$(dirname "$0")" || exit 1
[ -f ../setclaimcheck.sh ] && source ../setclaimcheck.sh
if [ ! -f ../setcloudenv.sh ]; then
  echo "Copy setcloudenv.example to setcloudenv.sh and fill it in first."
  exit 1
fi
source ../setcloudenv.sh
exec go run ../cmd/api
