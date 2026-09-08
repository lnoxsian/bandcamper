# Bandcamper Makefile

BINARY_NAME := bandcamper
CMD_DIR := ./cmd/bandcamper
BIN_DIR := ./bin
DIST_DIR := ./dist
VERSION_FILE := VERSION
VERSION ?= $(shell cat $(VERSION_FILE) 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"
BUILD_FLAGS := -trimpath $(LDFLAGS)
CGO_ENABLED ?= 0
export CGO_ENABLED
GO ?= go

.PHONY: all build build-min build-target build-all \
        build-linux build-linux-amd64 build-linux-arm64 \
        build-darwin build-darwin-amd64 build-darwin-arm64 \
        build-windows build-windows-amd64 build-windows-arm64 \
        version set-version bump-patch bump-minor bump-major \
        install run test test-race test-cover vet fmt clean release help

all: build

## build: Compile native binary for the current platform to bin/bandcamper
build:
	@mkdir -p $(BIN_DIR)
	$(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Built $(BIN_DIR)/$(BINARY_NAME)"

## build-min: Build ultra-compact binary with UPX compression (~2.5 MB)
build-min: build
	@if command -v upx >/dev/null 2>&1; then \
		upx --best --lzma $(BIN_DIR)/$(BINARY_NAME); \
		echo "Compressed $(BIN_DIR)/$(BINARY_NAME) with UPX"; \
	else \
		echo "Note: install 'upx' (e.g. sudo apt install upx) to shrink to ~2.5MB"; \
	fi

## build-target: Build for custom OS and ARCH (e.g. make build-target OS=linux ARCH=arm64)
build-target:
	@if [ -z "$(OS)" ] || [ -z "$(ARCH)" ]; then \
		echo "Error: please specify OS and ARCH, e.g. make build-target OS=linux ARCH=arm64"; \
		exit 1; \
	fi
	@mkdir -p $(BIN_DIR)
	GOOS=$(OS) GOARCH=$(ARCH) $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-$(OS)-$(ARCH)$(if $(filter windows,$(OS)),.exe,) $(CMD_DIR)
	@echo "Built $(BIN_DIR)/$(BINARY_NAME)-$(OS)-$(ARCH)$(if $(filter windows,$(OS)),.exe,)"

## build-linux: Build Linux binary (default ARCH=amd64, or make build-linux ARCH=arm64)
build-linux:
	@$(MAKE) build-target OS=linux ARCH=$(or $(ARCH),amd64)

## build-linux-amd64: Build Linux x86_64 binary
build-linux-amd64:
	@$(MAKE) build-target OS=linux ARCH=amd64

## build-linux-arm64: Build Linux ARM64 binary
build-linux-arm64:
	@$(MAKE) build-target OS=linux ARCH=arm64

## build-darwin: Build macOS binary (default ARCH=arm64, or make build-darwin ARCH=amd64)
build-darwin:
	@$(MAKE) build-target OS=darwin ARCH=$(or $(ARCH),arm64)

## build-darwin-amd64: Build macOS Intel x86_64 binary
build-darwin-amd64:
	@$(MAKE) build-target OS=darwin ARCH=amd64

## build-darwin-arm64: Build macOS Apple Silicon ARM64 binary
build-darwin-arm64:
	@$(MAKE) build-target OS=darwin ARCH=arm64

## build-windows: Build Windows binary (default ARCH=amd64, or make build-windows ARCH=arm64)
build-windows:
	@$(MAKE) build-target OS=windows ARCH=$(or $(ARCH),amd64)

## build-windows-amd64: Build Windows x86_64 .exe binary
build-windows-amd64:
	@$(MAKE) build-target OS=windows ARCH=amd64

## build-windows-arm64: Build Windows ARM64 .exe binary
build-windows-arm64:
	@$(MAKE) build-target OS=windows ARCH=arm64

## build-all: Build binaries for all supported platforms into bin/
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64
	@echo "All platform binaries compiled into $(BIN_DIR)/"

## install: Install binary to $GOPATH/bin
install:
	$(GO) install $(LDFLAGS) $(CMD_DIR)
	@echo "Installed $(BINARY_NAME)"

## run: Run the application with custom arguments (e.g. make run ARGS="--help")
run:
	$(GO) run $(CMD_DIR) $(ARGS)

## test: Run unit and integration tests with optional ARGS (e.g. make test ARGS="-v ./internal/bandcamp/..." or make test ARGS="-run TestResolveURL ./...")
test:
	$(GO) test $(if $(ARGS),$(ARGS),-v ./...)

## test-race: Run tests with data race detector and optional ARGS
test-race:
	$(GO) test -race $(if $(ARGS),$(ARGS),-v ./...)

## test-cover: Run tests and generate coverage report
test-cover:
	@mkdir -p $(BIN_DIR)
	$(GO) test -coverprofile=$(BIN_DIR)/coverage.out $(if $(ARGS),$(ARGS),./...)
	$(GO) tool cover -func=$(BIN_DIR)/coverage.out

## vet: Run go vet static analysis
vet:
	$(GO) vet ./...

## fmt: Format all Go source files
fmt:
	$(GO) fmt ./...

## clean: Remove compiled binaries and test/build artifacts
clean:
	@rm -rf $(BIN_DIR) $(DIST_DIR) $(BINARY_NAME) $(BINARY_NAME).exe coverage.out
	@echo "Cleaned build artifacts."

## release: Build cross-platform release binaries into dist/
release: clean
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	GOOS=windows GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-arm64.exe $(CMD_DIR)
	@echo "Cross-platform release binaries compiled into $(DIST_DIR)/"

## version: Display the current version from the VERSION file
version:
	@cat $(VERSION_FILE)

## set-version: Update the VERSION file (e.g. make set-version V=1.0.1)
set-version:
	@NEW_VER="$(or $(V),$(VERSION))"; \
	if [ -z "$$NEW_VER" ] || [ "$$NEW_VER" = "$$(cat $(VERSION_FILE) 2>/dev/null)" ]; then \
		echo "Error: please specify a new version, e.g. make set-version V=1.0.1"; \
		exit 1; \
	fi; \
	echo "$$NEW_VER" > $(VERSION_FILE); \
	echo "Updated $(VERSION_FILE) to $$NEW_VER"

## bump-patch: Bump the patch version in VERSION file (e.g. 1.0.0 -> 1.0.1)
bump-patch:
	@awk -F. '{$$NF = $$NF + 1;} 1' OFS=. $(VERSION_FILE) > $(VERSION_FILE).tmp && mv $(VERSION_FILE).tmp $(VERSION_FILE)
	@echo "Bumped patch version to $$(cat $(VERSION_FILE))"

## bump-minor: Bump the minor version in VERSION file (e.g. 1.0.0 -> 1.1.0)
bump-minor:
	@awk -F. '{$$2 = $$2 + 1; $$3 = 0;} 1' OFS=. $(VERSION_FILE) > $(VERSION_FILE).tmp && mv $(VERSION_FILE).tmp $(VERSION_FILE)
	@echo "Bumped minor version to $$(cat $(VERSION_FILE))"

## bump-major: Bump the major version in VERSION file (e.g. 1.0.0 -> 2.0.0)
bump-major:
	@awk -F. '{$$1 = $$1 + 1; $$2 = 0; $$3 = 0;} 1' OFS=. $(VERSION_FILE) > $(VERSION_FILE).tmp && mv $(VERSION_FILE).tmp $(VERSION_FILE)
	@echo "Bumped major version to $$(cat $(VERSION_FILE))"

## help: Show this help message
help:
	@echo "Usage: make [target] [ARGS=\"...\"]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -e 's/^## //'
