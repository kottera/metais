#!/bin/sh
# Vets and tests weather-cli, and checks its formatting: what
# to run before every push, and what CI runs.
set -eu
cd "$(dirname "$0")/.."

go vet ./...
go test ./...

unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
	echo "gofmt would rewrite:" >&2
	echo "$unformatted" >&2
	exit 1
fi
