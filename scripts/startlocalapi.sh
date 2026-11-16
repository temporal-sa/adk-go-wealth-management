#!/bin/bash
# Start the API for the web front end, at http://127.0.0.1:8000.
cd "$(dirname "$0")" || exit 1
[ -f ../setclaimcheck.sh ] && source ../setclaimcheck.sh
exec go run ../cmd/api
