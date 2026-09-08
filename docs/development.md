# Development Guide

## Repository layout

```text
cmd/bandcamper/       CLI entry point and process orchestration
internal/bandcamp/    URL resolution, HTTP access, and page parsing
internal/config/      TOML configuration and defaults
internal/downloader/  worker pool, retries, progress, and orchestration
internal/metadata/    ID3 tags, artwork, and lyrics
internal/playlist/    M3U, PLS, WPL, and ZPL writers
internal/storage/     output paths, templates, sanitization, and atomic files
tests/                integration tests and fixtures
docs/                 project documentation
```

The command package should remain thin. Domain logic belongs in the relevant `internal` package so it can be tested independently of CLI argument handling.

## Local commands

Run the complete test suite:

```sh
go test ./...
```

Run tests with the race detector:

```sh
go test -race ./...
```

Run static analysis:

```sh
go vet ./...
```

Format Go files:

```sh
go fmt ./...
```

Build the current platform:

```sh
go build -o bin/bandcamper ./cmd/bandcamper
```

Equivalent project recipes are available through both `make` and `just`:

```sh
make test
just test
make build
just build
```

The test recipes accept additional arguments. For example:

```sh
make test ARGS='-run TestResolveURL ./internal/bandcamp/...'
just test -run TestResolveURL ./internal/bandcamp/...
```

## Cross-compilation

Supported release targets are Linux, macOS, and Windows on `amd64` and `arm64`.

With Go directly:

```sh
GOOS=linux GOARCH=arm64 go build -o bin/bandcamper-linux-arm64 ./cmd/bandcamper
```

With `make`:

```sh
make build-target OS=linux ARCH=arm64
make build-all
make release
```

With `just`:

```sh
just build-target linux arm64
just build-all
just release
```

The recipes place development binaries in `bin/` and release binaries in `dist/`.

## Testing strategy

Tests are organized around package responsibilities:

- Bandcamp tests cover URL resolution, HTTP client behavior, and fixture parsing.
- Downloader tests cover concurrency, retries, cancellation, progress, and file outcomes.
- Metadata tests cover ID3 fields, artwork, and lyrics.
- Playlist tests cover format writers and relative track paths.
- Storage and configuration tests cover sanitization, templates, defaults, and TOML loading.
- Integration tests exercise the command flow using repository fixtures.

When changing parsing behavior, add or update a fixture-based test. When changing output naming, configuration, or concurrency, add a focused unit test before relying on a full integration test.

## Implementation flow

A download follows this path:

1. CLI arguments and the optional TOML file are merged into a configuration.
2. Each input is classified and normalized by the Bandcamp URL resolver.
3. Artist pages are expanded into release URLs; album and track pages are parsed into a normalized release model.
4. The downloader computes sanitized output paths and creates track jobs.
5. A worker pool downloads tracks with retry handling and context cancellation.
6. Successful tracks are finalized from temporary `.part` files.
7. Metadata and artwork are written according to configuration.
8. A playlist is generated from successful and skipped tracks when requested.

## Change guidelines

- Keep public behavior documented in `docs/` and the README consistent with the implementation.
- Preserve context cancellation through network and worker operations.
- Do not treat authentication, DRM, purchase restrictions, or access-control bypasses as supported behavior.
- Prefer standard-library APIs and the existing package boundaries before introducing new dependencies.
- Keep output writes atomic so interrupted downloads do not replace completed files.
- Run `go test ./...`, `go test -race ./...`, and `go vet ./...` before submitting changes that affect shared behavior.
