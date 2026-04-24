# pd-cli

Terminal UI (TUI) client for the personal-dashboard. Connects to
`dashboard-api` and renders weather/pollen data, refreshing on an interval.

This module is a standalone Go module with its own `go.mod` and a nested
`go.work` that keeps it out of the repo-root Go workspace.

> **Renaming the binary:** the name `pd-cli` appears in three places.
> To rename:
>
> 1. Change `BINARY := pd-cli` at the top of `Makefile`.
> 2. Change `appName = "pd-cli"` in `cmd/main.go`.
> 3. Find-and-replace `pd-cli` in this README.

## Prerequisites

- Go 1.25 or newer (`go version`)
- `make` (standard on macOS and Linux)
- For Pi builds: nothing extra — Go cross-compiles natively

## Run locally

```
make dev
```

Defaults to `http://localhost:8080` with a 5-minute refresh. Override with env
vars:

```
DASHBOARD_API_URL=http://api-staging.ianbeefang.com REFRESH_INTERVAL=30s make dev
```

Or with flags via `go run`:

```
go run ./cmd -url=http://api-staging.ianbeefang.com -refresh=30s
```

Enable debug logging with `DEBUG=true`. Quit the TUI with `q` or `Ctrl+C`.

## Build

All build artifacts land in `dist/` (gitignored).

Native binary for your host:

```
make build            # produces ./dist/pd-cli
```

Cross-compile for Raspberry Pi Zero W:

```
make build-pi-zero    # produces ./dist/pd-cli-pi-zero
```

Clean artifacts:

```
make clean            # removes ./dist/
```

## Building for a Raspberry Pi Zero W

The Pi Zero W has an ARMv6 CPU. If you build with `GOARM=7` or omit
`GOARM`, the binary contains instructions the CPU cannot execute, and
running it prints `Exec format error`.

Verify the Pi's architecture before building:

```
ssh pi@<pi-address> 'uname -m'    # expect: armv6l
```

Build and deploy:

```
make build-pi-zero
scp dist/pd-cli-pi-zero pi@<pi-address>:~/pd-cli
ssh pi@<pi-address> 'chmod +x ~/pd-cli && ~/pd-cli'
```

Sanity-check the artifact before copying:

```
file dist/pd-cli-pi-zero
# expect: ELF 32-bit LSB executable, ARM, EABI5 version 1 (SYSV), ...
```

## Other targets (not scripted)

Only the targets actively deployed to are wired up in the Makefile. For
one-offs (prefix with `mkdir -p dist &&` if `dist/` doesn't exist yet):

| Hardware                       | `uname -m` | Build command                                                                |
|--------------------------------|-----------|------------------------------------------------------------------------------|
| Pi Zero / Zero W / 1 / A / B   | `armv6l`  | `make build-pi-zero`                                                         |
| Pi 2 / 3 (32-bit OS)           | `armv7l`  | `GOOS=linux GOARCH=arm GOARM=7 go build -o dist/pd-cli-armv7 ./cmd`          |
| Pi 4 / 5 (64-bit OS)           | `aarch64` | `GOOS=linux GOARCH=arm64 go build -o dist/pd-cli-arm64 ./cmd`                |
| Generic Linux (Intel/AMD)      | `x86_64`  | `GOOS=linux GOARCH=amd64 go build -o dist/pd-cli-amd64 ./cmd`                |

If a target becomes a regular build, add it to the Makefile.

## Test

```
make test
```

## Notes on the nested `go.work`

The `go.work` in this directory isolates the CLI from the repo-root Go
workspace. Go resolves workspaces by walking up from the current directory
and using the nearest `go.work` it finds — this one shadows the root one,
so the CLI builds purely against its own `go.mod`. This is intentional:
the CLI has no dependency on `services/shared` (see commit `4eeedab`), and
keeping it out of the root workspace avoids coupling that doesn't exist.
