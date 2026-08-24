# nadi-server

Central server for Nadi: receives agent heartbeats, stores metrics, serves the dashboard.

## Development

Prerequisites: Go 1.22+ (see `go.mod`), GNU make.

```sh
make build        # builds bin/nadi-server for the current OS/arch
make build-linux  # cross-compiles a Linux amd64 binary (CGO disabled)
make test         # run tests
make lint         # gofmt check + go vet
make run          # run from source
```

## Project layout

```
cmd/nadi-server/  entrypoint
```

See `PLAN.md` (parent repo) for the full roadmap.
