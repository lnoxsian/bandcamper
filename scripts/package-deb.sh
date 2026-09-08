#!/usr/bin/env bash
# ==============================================================================
# package-deb.sh - Build Debian (.deb) package for bandcamper
# Usage: ./scripts/package-deb.sh [ARCH] [VERSION] [OUTPUT_DIR]
# Example: ./scripts/package-deb.sh amd64 1.0.0 dist
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Target architecture (default: host or amd64)
RAW_ARCH="${1:-}"
if [ -z "${RAW_ARCH}" ]; then
    HOST_M="$(uname -m)"
    case "${HOST_M}" in
        x86_64|amd64) RAW_ARCH="amd64" ;;
        aarch64|arm64) RAW_ARCH="arm64" ;;
        *) RAW_ARCH="amd64" ;;
    esac
fi

# Normalize architecture to Debian format (amd64, arm64)
case "${RAW_ARCH}" in
    x86_64|amd64) DEB_ARCH="amd64" ;;
    aarch64|arm64) DEB_ARCH="arm64" ;;
    all)
        echo "Building Debian packages for all architectures (amd64, arm64)..."
        "${BASH_SOURCE[0]}" amd64 "${2:-}" "${3:-}"
        "${BASH_SOURCE[0]}" arm64 "${2:-}" "${3:-}"
        exit 0
        ;;
    *) DEB_ARCH="${RAW_ARCH}" ;;
esac

# Resolve version
RAW_VERSION="${2:-}"
if [ -z "${RAW_VERSION}" ]; then
    RAW_VERSION="$(cat "${REPO_DIR}/VERSION" 2>/dev/null || echo "0.1.0")"
fi
VERSION="${RAW_VERSION#v}"

# Output directory
OUTPUT_DIR="${3:-${REPO_DIR}/dist}"
mkdir -p "${OUTPUT_DIR}"

echo "==> Packaging bandcamper ${VERSION} for Debian (${DEB_ARCH})..."

# Locate binary or build if needed
BIN_SRC=""
CANDIDATE_PATHS=(
    "${OUTPUT_DIR}/bandcamper-linux-${DEB_ARCH}"
    "${REPO_DIR}/bin/bandcamper-linux-${DEB_ARCH}"
    "${REPO_DIR}/dist/bandcamper-linux-${DEB_ARCH}"
)

for p in "${CANDIDATE_PATHS[@]}"; do
    if [ -f "${p}" ]; then
        BIN_SRC="${p}"
        break
    fi
done

# If host matches and bin/bandcamper exists, use it
if [ -z "${BIN_SRC}" ] && [ "${DEB_ARCH}" = "amd64" ] && [ "$(uname -m)" = "x86_64" ] && [ -f "${REPO_DIR}/bin/bandcamper" ]; then
    BIN_SRC="${REPO_DIR}/bin/bandcamper"
fi

# If binary still not found, compile it now
if [ -z "${BIN_SRC}" ]; then
    echo "    Binary not found, compiling bandcamper-linux-${DEB_ARCH}..."
    BIN_SRC="${OUTPUT_DIR}/bandcamper-linux-${DEB_ARCH}"
    GOOS=linux GOARCH="${DEB_ARCH}" CGO_ENABLED=0 go build \
        -trimpath \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o "${BIN_SRC}" \
        "${REPO_DIR}/cmd/bandcamper"
fi

# Create temporary staging directory
STAGE_DIR="$(mktemp -d)"
trap 'rm -rf "${STAGE_DIR}"' EXIT

DEBIAN_DIR="${STAGE_DIR}/DEBIAN"
BIN_DIR="${STAGE_DIR}/usr/bin"
BASH_COMP_DIR="${STAGE_DIR}/usr/share/bash-completion/completions"
ZSH_COMP_DIR="${STAGE_DIR}/usr/share/zsh/vendor-completions"
FISH_COMP_DIR="${STAGE_DIR}/usr/share/fish/vendor_completions.d"
DOC_DIR="${STAGE_DIR}/usr/share/doc/bandcamper"

mkdir -p "${DEBIAN_DIR}" "${BIN_DIR}" "${BASH_COMP_DIR}" "${ZSH_COMP_DIR}" "${FISH_COMP_DIR}" "${DOC_DIR}"

# Copy binary
cp -f "${BIN_SRC}" "${BIN_DIR}/bandcamper"
chmod 755 "${BIN_DIR}/bandcamper"

# Copy completions if available
COMPLETIONS_DIR="${REPO_DIR}/assets/completions"
if [ -d "${COMPLETIONS_DIR}" ]; then
    [ -f "${COMPLETIONS_DIR}/bandcamper.bash" ] && cp -f "${COMPLETIONS_DIR}/bandcamper.bash" "${BASH_COMP_DIR}/bandcamper" && chmod 644 "${BASH_COMP_DIR}/bandcamper"
    [ -f "${COMPLETIONS_DIR}/_bandcamper" ] && cp -f "${COMPLETIONS_DIR}/_bandcamper" "${ZSH_COMP_DIR}/_bandcamper" && chmod 644 "${ZSH_COMP_DIR}/_bandcamper"
    [ -f "${COMPLETIONS_DIR}/bandcamper.fish" ] && cp -f "${COMPLETIONS_DIR}/bandcamper.fish" "${FISH_COMP_DIR}/bandcamper.fish" && chmod 644 "${FISH_COMP_DIR}/bandcamper.fish"
fi

# Copy documentation
[ -f "${REPO_DIR}/README.md" ] && cp -f "${REPO_DIR}/README.md" "${DOC_DIR}/README.md" && chmod 644 "${DOC_DIR}/README.md"
[ -f "${REPO_DIR}/LICENSE" ] && cp -f "${REPO_DIR}/LICENSE" "${DOC_DIR}/copyright" && chmod 644 "${DOC_DIR}/copyright"

# Calculate installed size in KB
INSTALLED_SIZE="$(du -sk "${STAGE_DIR}" | cut -f1)"

# Generate DEBIAN/control file
cat > "${DEBIAN_DIR}/control" <<EOF
Package: bandcamper
Version: ${VERSION}
Section: sound
Priority: optional
Architecture: ${DEB_ARCH}
Installed-Size: ${INSTALLED_SIZE}
Maintainer: lnoxsian <https://github.com/lnoxsian/bandcamper>
Homepage: https://github.com/lnoxsian/bandcamper
Description: Fast, robust Bandcamp downloader in Go
 Bandcamper is a lightweight, cross-platform Bandcamp downloader written
 in Go. It resolves artist, album, and track URLs, downloads audio streams
 concurrently, embeds ID3v2 metadata, artwork, lyrics, and generates playlists.
EOF
chmod 644 "${DEBIAN_DIR}/control"

# Target .deb filename
OUTPUT_FILE="${OUTPUT_DIR}/bandcamper_${VERSION}_${DEB_ARCH}.deb"

# Build package using dpkg-deb or fallback to ar/tar
if command -v dpkg-deb >/dev/null 2>&1; then
    dpkg-deb --build --root-owner-group "${STAGE_DIR}" "${OUTPUT_FILE}" 2>/dev/null || \
    dpkg-deb --build "${STAGE_DIR}" "${OUTPUT_FILE}"
else
    echo "    dpkg-deb not found, assembling .deb package using ar and tar..."
    WORK_DIR="$(mktemp -d)"
    echo "2.0" > "${WORK_DIR}/debian-binary"
    (cd "${DEBIAN_DIR}" && tar --numeric-owner --owner=0 --group=0 -czf "${WORK_DIR}/control.tar.gz" .)
    (cd "${STAGE_DIR}" && tar --numeric-owner --owner=0 --group=0 --exclude="./DEBIAN" -czf "${WORK_DIR}/data.tar.gz" .)
    ar -rcs "${OUTPUT_FILE}" "${WORK_DIR}/debian-binary" "${WORK_DIR}/control.tar.gz" "${WORK_DIR}/data.tar.gz"
    rm -rf "${WORK_DIR}"
fi

echo "==> Successfully created ${OUTPUT_FILE}"
