#!/usr/bin/env bash
# ==============================================================================
# package-rpm.sh - Build RPM package for bandcamper
# Usage: ./scripts/package-rpm.sh [ARCH] [VERSION] [OUTPUT_DIR]
# Example: ./scripts/package-rpm.sh x86_64 1.0.0 dist
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Check for rpmbuild
if ! command -v rpmbuild >/dev/null 2>&1; then
    echo "Error: 'rpmbuild' is required to build RPM packages." >&2
    echo "Please install it: 'sudo apt install rpm' (Debian/Ubuntu) or 'sudo dnf install rpm-build' (Fedora/RHEL)." >&2
    exit 1
fi

# Target architecture (default: host)
RAW_ARCH="${1:-}"
if [ -z "${RAW_ARCH}" ]; then
    HOST_M="$(uname -m)"
    case "${HOST_M}" in
        x86_64|amd64) RAW_ARCH="x86_64" ;;
        aarch64|arm64) RAW_ARCH="aarch64" ;;
        *) RAW_ARCH="x86_64" ;;
    esac
fi

# Normalize architecture to RPM format (x86_64, aarch64) and Go format (amd64, arm64)
case "${RAW_ARCH}" in
    x86_64|amd64)
        RPM_ARCH="x86_64"
        GO_ARCH="amd64"
        ;;
    aarch64|arm64)
        RPM_ARCH="aarch64"
        GO_ARCH="arm64"
        ;;
    all)
        echo "Building RPM packages for all architectures (x86_64, aarch64)..."
        "${BASH_SOURCE[0]}" x86_64 "${2:-}" "${3:-}"
        "${BASH_SOURCE[0]}" aarch64 "${2:-}" "${3:-}"
        exit 0
        ;;
    *)
        RPM_ARCH="${RAW_ARCH}"
        GO_ARCH="${RAW_ARCH}"
        ;;
esac

# Resolve version
RAW_VERSION="${2:-}"
if [ -z "${RAW_VERSION}" ]; then
    RAW_VERSION="$(cat "${REPO_DIR}/VERSION" 2>/dev/null || echo "0.1.0")"
fi
VERSION="${RAW_VERSION#v}"

# Output directory (resolve to absolute path)
RAW_OUTPUT_DIR="${3:-${REPO_DIR}/dist}"
mkdir -p "${RAW_OUTPUT_DIR}"
OUTPUT_DIR="$(cd "${RAW_OUTPUT_DIR}" && pwd)"

echo "==> Packaging bandcamper ${VERSION} for RPM (${RPM_ARCH})..."

# Locate binary or build if needed
BIN_SRC=""
CANDIDATE_PATHS=(
    "${OUTPUT_DIR}/bandcamper-linux-${GO_ARCH}"
    "${REPO_DIR}/bin/bandcamper-linux-${GO_ARCH}"
    "${REPO_DIR}/dist/bandcamper-linux-${GO_ARCH}"
)

for p in "${CANDIDATE_PATHS[@]}"; do
    if [ -f "${p}" ]; then
        BIN_SRC="${p}"
        break
    fi
done

if [ -z "${BIN_SRC}" ] && [ "${GO_ARCH}" = "amd64" ] && [ "$(uname -m)" = "x86_64" ] && [ -f "${REPO_DIR}/bin/bandcamper" ]; then
    BIN_SRC="${REPO_DIR}/bin/bandcamper"
fi

if [ -z "${BIN_SRC}" ]; then
    echo "    Binary not found, compiling bandcamper-linux-${GO_ARCH}..."
    BIN_SRC="${OUTPUT_DIR}/bandcamper-linux-${GO_ARCH}"
    GOOS=linux GOARCH="${GO_ARCH}" CGO_ENABLED=0 go build \
        -trimpath \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o "${BIN_SRC}" \
        "${REPO_DIR}/cmd/bandcamper"
fi

# Ensure BIN_SRC is an absolute path (rpmbuild runs %install in /tmp/.../BUILD)
BIN_SRC="$(cd "$(dirname "${BIN_SRC}")" && pwd)/$(basename "${BIN_SRC}")"

# Set up rpmbuild directory structure
RPMBUILD_DIR="$(mktemp -d)"
trap 'rm -rf "${RPMBUILD_DIR}"' EXIT

mkdir -p "${RPMBUILD_DIR}"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

SPEC_FILE="${RPMBUILD_DIR}/SPECS/bandcamper.spec"
COMPLETIONS_DIR="${REPO_DIR}/assets/completions"

cat > "${SPEC_FILE}" <<EOF
%define _build_id_links none
%define debug_package %{nil}

Name:           bandcamper
Version:        ${VERSION}
Release:        1%{?dist}
Summary:        Fast, robust Bandcamp downloader in Go
License:        MIT
URL:            https://github.com/lnoxsian/bandcamper
AutoReqProv:    no

%description
Bandcamper is a lightweight, cross-platform Bandcamp downloader written
in Go. It resolves artist, album, and track URLs, downloads audio streams
concurrently, embeds ID3v2 metadata, artwork, lyrics, and generates playlists.

%install
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/usr/share/bash-completion/completions
mkdir -p %{buildroot}/usr/share/zsh/site-functions
mkdir -p %{buildroot}/usr/share/fish/vendor_completions.d
mkdir -p %{buildroot}/usr/share/doc/bandcamper
mkdir -p %{buildroot}/usr/share/licenses/bandcamper

cp "${BIN_SRC}" %{buildroot}/usr/bin/bandcamper
chmod 755 %{buildroot}/usr/bin/bandcamper

if [ -d "${COMPLETIONS_DIR}" ]; then
    [ -f "${COMPLETIONS_DIR}/bandcamper.bash" ] && cp "${COMPLETIONS_DIR}/bandcamper.bash" %{buildroot}/usr/share/bash-completion/completions/bandcamper
    [ -f "${COMPLETIONS_DIR}/_bandcamper" ] && cp "${COMPLETIONS_DIR}/_bandcamper" %{buildroot}/usr/share/zsh/site-functions/_bandcamper
    [ -f "${COMPLETIONS_DIR}/bandcamper.fish" ] && cp "${COMPLETIONS_DIR}/bandcamper.fish" %{buildroot}/usr/share/fish/vendor_completions.d/bandcamper.fish
fi

[ -f "${REPO_DIR}/README.md" ] && cp "${REPO_DIR}/README.md" %{buildroot}/usr/share/doc/bandcamper/
[ -f "${REPO_DIR}/LICENSE" ] && cp "${REPO_DIR}/LICENSE" %{buildroot}/usr/share/licenses/bandcamper/

%files
/usr/bin/bandcamper
/usr/share/bash-completion/completions/bandcamper
/usr/share/zsh/site-functions/_bandcamper
/usr/share/fish/vendor_completions.d/bandcamper.fish
%doc /usr/share/doc/bandcamper/README.md
%license /usr/share/licenses/bandcamper/LICENSE

%changelog
* $(date "+%a %b %d %Y") lnoxsian <https://github.com/lnoxsian/bandcamper> - ${VERSION}-1
- Release ${VERSION}
EOF

# Build binary RPM
rpmbuild -bb \
    --define "_topdir ${RPMBUILD_DIR}" \
    --target "${RPM_ARCH}" \
    "${SPEC_FILE}"

# Copy built RPM(s) to output directory
find "${RPMBUILD_DIR}/RPMS" -type f -name "*.rpm" -exec cp -f {} "${OUTPUT_DIR}/" \;

echo "==> Successfully created RPM packages in ${OUTPUT_DIR}/"
