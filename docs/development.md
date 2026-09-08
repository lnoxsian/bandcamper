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

### Linux package creation (.deb and .rpm)

Debian (`.deb`) and Red Hat / Fedora (`.rpm`) packages can be created locally via `scripts/linux/package-deb.sh` and `scripts/linux/package-rpm.sh`, or using `make` / `just`:

```sh
# Build Debian packages
make package-deb ARCH=amd64    # or: just package-deb amd64
make package-deb-all           # builds amd64 and arm64

# Build RPM packages
make package-rpm ARCH=x86_64   # or: just package-rpm x86_64
make package-rpm-all           # builds x86_64 and aarch64

# Build all packages
make packages                  # or: just packages
```

### GitHub Actions release pipeline

Cross-platform builds and GitHub Releases are automated via `.github/workflows/release.yml`:

- **Triggers**: Manual execution via **Actions → Release → Run workflow** (`workflow_dispatch`), where you can optionally specify a tag name (defaults to `VERSION`), draft status, or pre-release status.
- **Verification**: Executes `go vet` and `go test -race ./...` before building.
- **Matrix**: Builds 6 cross-platform targets (`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`).
- **Packaging**:
  - Compressed `.tar.gz` (Linux/macOS) and `.zip` (Windows) archives bundling docs, shell completions, and `install.sh`.
  - Debian packages: `bandcamper_<version>_amd64.deb` and `bandcamper_<version>_arm64.deb`.
  - RPM packages: `bandcamper-<version>-1.x86_64.rpm` and `bandcamper-<version>-1.aarch64.rpm`.
  - Standalone binaries and a SHA-256 `checksums.txt` manifest.
- **Publication**: Automatically attaches all assets and auto-generated release notes to the GitHub Release.

A companion `.github/workflows/ci.yml` runs test and lint validation on all pull requests and main-branch pushes.

## Version management

The project version is tracked in the root [`VERSION`](../VERSION) file and injected during compilation via `-ldflags`.

View the current version:

```sh
make version
just version
```

Set a specific version:

```sh
make set-version V=1.0.1
just set-version 1.0.1
```

Bump semver components automatically:

```sh
make bump-patch     # or: just bump-patch (e.g. 1.0.0 -> 1.0.1)
make bump-minor     # or: just bump-minor (e.g. 1.0.0 -> 1.1.0)
make bump-major     # or: just bump-major (e.g. 1.0.0 -> 2.0.0)
```

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
