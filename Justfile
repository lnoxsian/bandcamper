# Bandcamper Justfile

set dotenv-load := false
export CGO_ENABLED := "0"

binary_name := "bandcamper"
cmd_dir := "./cmd/bandcamper"
bin_dir := "./bin"
dist_dir := "./dist"
version := `cat VERSION 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || echo "1.0.0"`
ldflags := "-s -w -X main.Version=" + version

# List available recipes
default:
    @just --list

# Build native binary for the current host to bin/bandcamper
build:
    @mkdir -p {{bin_dir}}
    go build -trimpath -ldflags="{{ldflags}}" -o {{bin_dir}}/{{binary_name}} {{cmd_dir}}
    @echo "Built {{bin_dir}}/{{binary_name}}"

# Build ultra-compact binary with UPX compression (~2.5 MB)
build-min: build
    @if command -v upx >/dev/null 2>&1; then \
        upx --best --lzma {{bin_dir}}/{{binary_name}}; \
        echo "Compressed {{bin_dir}}/{{binary_name}} with UPX"; \
    else \
        echo "Note: install 'upx' (e.g. sudo apt install upx) to shrink to ~2.5MB"; \
    fi

# Build for custom target OS and ARCH (e.g. just build-target linux arm64)
build-target os arch:
    @mkdir -p {{bin_dir}}
    GOOS={{os}} GOARCH={{arch}} go build -trimpath -ldflags="{{ldflags}}" -o {{bin_dir}}/{{binary_name}}-{{os}}-{{arch}}{{ if os == "windows" { ".exe" } else { "" } }} {{cmd_dir}}
    @echo "Built {{bin_dir}}/{{binary_name}}-{{os}}-{{arch}}{{ if os == "windows" { ".exe" } else { "" } }}"

# Build for Linux (default arch: amd64; e.g. just build-linux arm64)
build-linux arch="amd64":
    @just build-target linux {{arch}}

# Build Linux x86_64
build-linux-amd64:
    @just build-target linux amd64

# Build Linux ARM64
build-linux-arm64:
    @just build-target linux arm64

# Build for macOS (default arch: arm64; e.g. just build-darwin amd64)
build-darwin arch="arm64":
    @just build-target darwin {{arch}}

# Build macOS Intel x86_64
build-darwin-amd64:
    @just build-target darwin amd64

# Build macOS Apple Silicon ARM64
build-darwin-arm64:
    @just build-target darwin arm64

# Build for Windows (default arch: amd64; e.g. just build-windows arm64)
build-windows arch="amd64":
    @just build-target windows {{arch}}

# Build Windows x86_64 .exe
build-windows-amd64:
    @just build-target windows amd64

# Build Windows ARM64 .exe
build-windows-arm64:
    @just build-target windows arm64

# Build binaries for all supported platforms into bin/
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64
    @echo "All platform binaries compiled into {{bin_dir}}/"

# Install binary to $GOPATH/bin
install:
    go install -trimpath -ldflags="{{ldflags}}" {{cmd_dir}}
    @echo "Installed {{binary_name}}"

# Install shell completions (bash, zsh, fish) for user
install-completions:
    @mkdir -p ~/.local/share/bash-completion/completions ~/.config/fish/completions ~/.zfunc
    @cp assets/completions/bandcamper.bash ~/.local/share/bash-completion/completions/bandcamper
    @cp assets/completions/bandcamper.fish ~/.config/fish/completions/bandcamper.fish
    @cp assets/completions/_bandcamper ~/.zfunc/_bandcamper
    @echo "Installed shell completions into ~/.local/share/bash-completion/completions, ~/.config/fish/completions, and ~/.zfunc"

# Run the application with custom arguments (e.g. just run --help)
run *args:
    go run {{cmd_dir}} {{args}}

# Run tests with custom args (e.g. just test -run TestResolveURL or just test ./internal/bandcamp/...)
test *args:
    go test {{ if args == "" { "-v ./..." } else { args } }}

# Run tests with data race detector and custom args
test-race *args:
    go test -race {{ if args == "" { "-v ./..." } else { args } }}

# Run tests and generate coverage report
test-cover *args:
    @mkdir -p {{bin_dir}}
    go test -coverprofile={{bin_dir}}/coverage.out {{ if args == "" { "./..." } else { args } }}
    go tool cover -func={{bin_dir}}/coverage.out

# Run go vet static analysis
vet:
    go vet ./...

# Format all Go source files
fmt:
    go fmt ./...

# Clean compiled binaries and test/build artifacts
clean:
    rm -rf {{bin_dir}} {{dist_dir}} {{binary_name}} {{binary_name}}.exe coverage.out
    @echo "Cleaned build artifacts."

# Build cross-platform release binaries into dist/
release: clean
    @mkdir -p {{dist_dir}}
    GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="{{ldflags}}" -o {{dist_dir}}/{{binary_name}}-linux-amd64 {{cmd_dir}}
    GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="{{ldflags}}" -o {{dist_dir}}/{{binary_name}}-linux-arm64 {{cmd_dir}}
    GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="{{ldflags}}" -o {{dist_dir}}/{{binary_name}}-darwin-amd64 {{cmd_dir}}
    GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="{{ldflags}}" -o {{dist_dir}}/{{binary_name}}-darwin-arm64 {{cmd_dir}}
    GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="{{ldflags}}" -o {{dist_dir}}/{{binary_name}}-windows-amd64.exe {{cmd_dir}}
    GOOS=windows GOARCH=arm64 go build -trimpath -ldflags="{{ldflags}}" -o {{dist_dir}}/{{binary_name}}-windows-arm64.exe {{cmd_dir}}
    @echo "Cross-platform release binaries compiled into {{dist_dir}}/"

# Build Debian (.deb) package (e.g. just package-deb amd64)
package-deb arch="amd64":
    @./scripts/linux/package-deb.sh {{arch}} {{version}} {{dist_dir}}

# Build Debian (.deb) packages for all architectures
package-deb-all:
    @./scripts/linux/package-deb.sh amd64 {{version}} {{dist_dir}}
    @./scripts/linux/package-deb.sh arm64 {{version}} {{dist_dir}}

# Build RPM package (e.g. just package-rpm x86_64)
package-rpm arch="x86_64":
    @./scripts/linux/package-rpm.sh {{arch}} {{version}} {{dist_dir}}

# Build RPM packages for all architectures
package-rpm-all:
    @./scripts/linux/package-rpm.sh x86_64 {{version}} {{dist_dir}}
    @./scripts/linux/package-rpm.sh aarch64 {{version}} {{dist_dir}}

# Build all Linux packages (.deb and .rpm) into dist/
packages: package-deb-all package-rpm-all

# Display the current version from the VERSION file
version:
    @cat VERSION

# Update the version in the VERSION file (e.g. just set-version 1.0.1)
set-version new_version:
    @echo "{{new_version}}" > VERSION
    @echo "Updated VERSION to {{new_version}}"

# Bump the patch version in the VERSION file (e.g. 1.0.0 -> 1.0.1)
bump-patch:
    @awk -F. '{$$NF = $$NF + 1;} 1' OFS=. VERSION > VERSION.tmp && mv VERSION.tmp VERSION
    @echo "Bumped patch version to `cat VERSION`"

# Bump the minor version in the VERSION file (e.g. 1.0.0 -> 1.1.0)
bump-minor:
    @awk -F. '{$$2 = $$2 + 1; $$3 = 0;} 1' OFS=. VERSION > VERSION.tmp && mv VERSION.tmp VERSION
    @echo "Bumped minor version to `cat VERSION`"

# Bump the major version in the VERSION file (e.g. 1.0.0 -> 2.0.0)
bump-major:
    @awk -F. '{$$1 = $$1 + 1; $$2 = 0; $$3 = 0;} 1' OFS=. VERSION > VERSION.tmp && mv VERSION.tmp VERSION
    @echo "Bumped major version to `cat VERSION`"
