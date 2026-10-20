# weather-cli

Prints the weather for a place.

## Building

`weather-cli` needs Go 1.25 or later.

```sh
go build ./cmd/weather-cli
./weather-cli
```

The repository holds these commands too, each built the same way:

- `weather-import`: Loads observations into the weather store.

## Checking

`scripts/check.sh` vets and tests the module and checks its formatting.
Run it before every push.
The CI workflow runs it on every push to `main` and every pull request.

## Container

```sh
docker build -t weather-cli .
docker run --rm weather-cli
```

The image holds the `weather-cli` binary and nothing else.

## Licence

Copyright 2026 Kottera. Released under the `Apache-2.0` licence.
