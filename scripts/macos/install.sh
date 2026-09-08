#!/usr/bin/env bash
# ==============================================================================
# Bandcamper Installer for macOS
# Installs bandcamper to /usr/local/bin and configures shell completions
# ==============================================================================

set -euo pipefail

# ANSI color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

info() {
    echo -e "${BLUE}${BOLD}[INFO]${NC} $*"
}

success() {
    echo -e "${GREEN}${BOLD}[OK]${NC} $*"
}

warn() {
    echo -e "${YELLOW}${BOLD}[WARN]${NC} $*"
}

error() {
    echo -e "${RED}${BOLD}[ERROR]${NC} $*" >&2
}

# Determine sudo requirement
SUDO=""
if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        error "Root privileges or sudo are required to install to /usr/local/bin."
        exit 1
    fi
fi

# Detect calling user and environment (handles sudo invocation correctly on macOS)
REAL_USER="${SUDO_USER:-$USER}"

# Resolve user home and login shell (using dscl on macOS with safe fallbacks)
REAL_HOME=""
if command -v dscl >/dev/null 2>&1; then
    REAL_HOME="$(dscl . -read "/Users/${REAL_USER}" NFSHomeDirectory 2>/dev/null | awk '{print $2}')"
fi
if [ -z "${REAL_HOME}" ]; then
    REAL_HOME="$(eval echo "~${REAL_USER}")"
fi
[ -z "${REAL_HOME}" ] && REAL_HOME="${HOME}"

USER_SHELL=""
if command -v dscl >/dev/null 2>&1; then
    USER_SHELL="$(dscl . -read "/Users/${REAL_USER}" UserShell 2>/dev/null | awk '{print $2}')"
fi
[ -z "${USER_SHELL}" ] && USER_SHELL="${SHELL:-/bin/zsh}"
CURRENT_SHELL="$(basename "${USER_SHELL}")"

# Allow shell override via argument (e.g. ./install.sh zsh, ./install.sh bash, ./install.sh fish, ./install.sh all)
TARGET_SHELL="${1:-${CURRENT_SHELL}}"

# Locate script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Architecture detection on macOS
ARCH_RAW="$(uname -m)"
case "${ARCH_RAW}" in
    x86_64) GO_ARCH="amd64" ;;
    arm64|aarch64) GO_ARCH="arm64" ;;
    *) GO_ARCH="arm64" ;;
esac

# Locate bandcamper binary
BIN_SRC=""
CANDIDATE_BINS=(
    "${SCRIPT_DIR}/bandcamper"
    "${SCRIPT_DIR}/../../bin/bandcamper"
    "${SCRIPT_DIR}/../../dist/bandcamper-darwin-${GO_ARCH}"
    "${SCRIPT_DIR}/../../bin/bandcamper-darwin-${GO_ARCH}"
    "${SCRIPT_DIR}/../bin/bandcamper"
    "${SCRIPT_DIR}/../../bandcamper"
)

for b in "${CANDIDATE_BINS[@]}"; do
    if [ -f "${b}" ]; then
        BIN_SRC="${b}"
        break
    fi
done

if [ -z "${BIN_SRC}" ] && command -v bandcamper >/dev/null 2>&1; then
    BIN_SRC="$(command -v bandcamper)"
fi

if [ -z "${BIN_SRC}" ] || [ ! -f "${BIN_SRC}" ]; then
    error "Could not find 'bandcamper' binary in '${SCRIPT_DIR}'."
    echo "Please ensure the binary exists or compile it first (e.g. 'go build -o bandcamper ./cmd/bandcamper')." >&2
    exit 1
fi

# Locate completions directory
COMPLETION_DIR=""
if [ -d "${SCRIPT_DIR}/completions" ]; then
    COMPLETION_DIR="${SCRIPT_DIR}/completions"
elif [ -d "${SCRIPT_DIR}/../../assets/completions" ]; then
    COMPLETION_DIR="${SCRIPT_DIR}/../../assets/completions"
elif [ -d "${SCRIPT_DIR}/../assets/completions" ]; then
    COMPLETION_DIR="${SCRIPT_DIR}/../assets/completions"
elif [ -d "${SCRIPT_DIR}/assets/completions" ]; then
    COMPLETION_DIR="${SCRIPT_DIR}/assets/completions"
fi

# 1. Install binary to /usr/local/bin (macOS standard, avoids SIP restrictions on /usr/bin)
DEST_DIR="${INSTALL_DIR:-/usr/local/bin}"
DEST_BIN="${DEST_DIR}/bandcamper"

info "Installing bandcamper to ${DEST_BIN}..."
$SUDO mkdir -p "${DEST_DIR}"
$SUDO cp -f "${BIN_SRC}" "${DEST_BIN}"
$SUDO chmod 755 "${DEST_BIN}"

# Remove Gatekeeper quarantine flag if present on macOS
if command -v xattr >/dev/null 2>&1; then
    $SUDO xattr -d com.apple.quarantine "${DEST_BIN}" 2>/dev/null || true
fi

success "Installed binary to ${DEST_BIN}"

# Verify PATH
if [[ ":${PATH}:" != *":${DEST_DIR}:"* ]]; then
    warn "${DEST_DIR} is not currently in your PATH."
    echo "You may want to add it to your ~/.zprofile or ~/.zshrc:"
    echo "  export PATH=\"${DEST_DIR}:\$PATH\""
fi

# 2. Completion installation functions
install_zsh_completion() {
    local src="${COMPLETION_DIR}/_bandcamper"
    if [ ! -f "${src}" ]; then
        warn "Zsh completion script not found at '${src}', skipping."
        return
    fi

    local installed=false

    # macOS Homebrew and system site-functions candidates
    local zsh_dirs=(
        "/opt/homebrew/share/zsh/site-functions"
        "/usr/local/share/zsh/site-functions"
        "/usr/share/zsh/site-functions"
    )

    for site_dir in "${zsh_dirs[@]}"; do
        if [ -d "${site_dir}" ] || $SUDO mkdir -p "${site_dir}" 2>/dev/null; then
            if $SUDO cp -f "${src}" "${site_dir}/_bandcamper" 2>/dev/null; then
                $SUDO chmod 644 "${site_dir}/_bandcamper"
                success "Installed Zsh completion: ${site_dir}/_bandcamper"
                installed=true
                break
            fi
        fi
    done

    # User-level fallback into ~/.zfunc
    local user_zfunc="${REAL_HOME}/.zfunc"
    mkdir -p "${user_zfunc}" 2>/dev/null || true
    if [ -d "${user_zfunc}" ]; then
        cp -f "${src}" "${user_zfunc}/_bandcamper" 2>/dev/null || true
        [ "$(id -u)" -eq 0 ] && [ -n "${SUDO_USER:-}" ] && chown -R "${REAL_USER}:" "${user_zfunc}" 2>/dev/null || true
    fi

    if [ "${installed}" = true ]; then
        info "Zsh completions are ready (restart shell or run: autoload -Uz compinit && compinit)"
    else
        warn "Placed Zsh completion in ~/.zfunc/_bandcamper. Ensure ~/.zfunc is in your fpath:"
        echo "  fpath=(~/.zfunc \$fpath)"
        echo "  autoload -Uz compinit && compinit"
    fi
}

install_bash_completion() {
    local src="${COMPLETION_DIR}/bandcamper.bash"
    if [ ! -f "${src}" ]; then
        warn "Bash completion script not found at '${src}', skipping."
        return
    fi

    local installed=false

    # macOS Homebrew bash completion directories
    local bash_dirs=(
        "/opt/homebrew/etc/bash_completion.d"
        "/usr/local/etc/bash_completion.d"
        "/usr/local/share/bash-completion/completions"
    )

    for bash_dir in "${bash_dirs[@]}"; do
        if [ -d "${bash_dir}" ] || $SUDO mkdir -p "${bash_dir}" 2>/dev/null; then
            if $SUDO cp -f "${src}" "${bash_dir}/bandcamper" 2>/dev/null; then
                $SUDO chmod 644 "${bash_dir}/bandcamper"
                success "Installed Bash completion: ${bash_dir}/bandcamper"
                installed=true
                break
            fi
        fi
    done

    # User-level fallback
    local user_bash_dir="${REAL_HOME}/.local/share/bash-completion/completions"
    mkdir -p "${user_bash_dir}" 2>/dev/null || true
    if [ -d "${user_bash_dir}" ]; then
        cp -f "${src}" "${user_bash_dir}/bandcamper" 2>/dev/null || true
        [ "$(id -u)" -eq 0 ] && [ -n "${SUDO_USER:-}" ] && chown -R "${REAL_USER}:" "${user_bash_dir}" 2>/dev/null || true
    fi

    if [ "${installed}" = true ]; then
        info "Bash completions are ready (restart shell or source the completion file)"
    fi
}

install_fish_completion() {
    local src="${COMPLETION_DIR}/bandcamper.fish"
    if [ ! -f "${src}" ]; then
        warn "Fish completion script not found at '${src}', skipping."
        return
    fi

    local installed=false

    # Homebrew vendor completions
    local fish_dirs=(
        "/opt/homebrew/share/fish/vendor_completions.d"
        "/usr/local/share/fish/vendor_completions.d"
    )

    for fish_dir in "${fish_dirs[@]}"; do
        if [ -d "${fish_dir}" ] || $SUDO mkdir -p "${fish_dir}" 2>/dev/null; then
            if $SUDO cp -f "${src}" "${fish_dir}/bandcamper.fish" 2>/dev/null; then
                $SUDO chmod 644 "${fish_dir}/bandcamper.fish"
                success "Installed Fish completion: ${fish_dir}/bandcamper.fish"
                installed=true
                break
            fi
        fi
    done

    # User-level fish completions
    local user_fish_dir="${REAL_HOME}/.config/fish/completions"
    mkdir -p "${user_fish_dir}" 2>/dev/null || true
    if [ -d "${user_fish_dir}" ]; then
        cp -f "${src}" "${user_fish_dir}/bandcamper.fish" 2>/dev/null || true
        [ "$(id -u)" -eq 0 ] && [ -n "${SUDO_USER:-}" ] && chown -R "${REAL_USER}:" "${user_fish_dir}" 2>/dev/null || true
        success "Installed Fish completion: ${user_fish_dir}/bandcamper.fish"
        installed=true
    fi

    if [ "${installed}" = true ]; then
        info "Fish completions will be available automatically in new fish sessions."
    fi
}

# 3. Apply completions based on shell
if [ -z "${COMPLETION_DIR}" ] || [ ! -d "${COMPLETION_DIR}" ]; then
    warn "Completions directory not found. Skipping completion installation."
else
    info "Detected user shell: ${CURRENT_SHELL} (target: ${TARGET_SHELL})"
    case "${TARGET_SHELL}" in
        *zsh*)
            install_zsh_completion
            ;;
        *bash*)
            install_bash_completion
            ;;
        *fish*)
            install_fish_completion
            ;;
        all)
            info "Installing completions for all supported shells (zsh, bash, fish)..."
            install_zsh_completion
            install_bash_completion
            install_fish_completion
            ;;
        *)
            warn "Unrecognized shell '${TARGET_SHELL}'. Falling back to Zsh (default on macOS)."
            install_zsh_completion
            ;;
    esac
fi

echo ""
success "Installation complete! Run 'bandcamper --help' to get started."
