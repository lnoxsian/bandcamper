#!/usr/bin/env bash
# ==============================================================================
# Bandcamper Installer
# Installs bandcamper to /usr/bin and configures shell completions
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
        error "Root privileges or sudo are required to install to /usr/bin."
        exit 1
    fi
fi

# Detect calling user and environment (handles sudo invocation correctly)
REAL_USER="${SUDO_USER:-$USER}"
REAL_HOME=$(getent passwd "${REAL_USER}" 2>/dev/null | cut -d: -f6 || echo "${HOME}")
USER_SHELL=$(getent passwd "${REAL_USER}" 2>/dev/null | cut -d: -f7 || echo "${SHELL:-/bin/bash}")
CURRENT_SHELL="$(basename "${USER_SHELL}")"

# Allow shell override via argument (e.g. ./install.sh zsh, ./install.sh fish, ./install.sh bash, ./install.sh all)
TARGET_SHELL="${1:-${CURRENT_SHELL}}"

# Locate script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Locate bandcamper binary
BIN_SRC=""
if [ -f "${SCRIPT_DIR}/bandcamper" ]; then
    BIN_SRC="${SCRIPT_DIR}/bandcamper"
elif [ -f "${SCRIPT_DIR}/../bin/bandcamper" ]; then
    BIN_SRC="${SCRIPT_DIR}/../bin/bandcamper"
elif [ -f "${SCRIPT_DIR}/../bandcamper" ]; then
    BIN_SRC="${SCRIPT_DIR}/../bandcamper"
elif command -v bandcamper >/dev/null 2>&1; then
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
elif [ -d "${SCRIPT_DIR}/../assets/completions" ]; then
    COMPLETION_DIR="${SCRIPT_DIR}/../assets/completions"
elif [ -d "${SCRIPT_DIR}/assets/completions" ]; then
    COMPLETION_DIR="${SCRIPT_DIR}/assets/completions"
fi

# 1. Install binary to /usr/bin
DEST_DIR="/usr/bin"
DEST_BIN="${DEST_DIR}/bandcamper"

info "Installing bandcamper to ${DEST_BIN}..."
$SUDO mkdir -p "${DEST_DIR}"
$SUDO cp -f "${BIN_SRC}" "${DEST_BIN}"
$SUDO chmod 755 "${DEST_BIN}"
success "Installed binary to ${DEST_BIN}"

# 2. Completion installation functions
install_bash_completion() {
    local src="${COMPLETION_DIR}/bandcamper.bash"
    if [ ! -f "${src}" ]; then
        warn "Bash completion script not found at '${src}', skipping."
        return
    fi

    local installed=false

    # Try system-wide bash-completion directory
    if [ -d "/usr/share/bash-completion/completions" ] || $SUDO mkdir -p "/usr/share/bash-completion/completions" 2>/dev/null; then
        $SUDO cp -f "${src}" "/usr/share/bash-completion/completions/bandcamper"
        $SUDO chmod 644 "/usr/share/bash-completion/completions/bandcamper"
        success "Installed Bash completion: /usr/share/bash-completion/completions/bandcamper"
        installed=true
    elif [ -d "/etc/bash_completion.d" ] || $SUDO mkdir -p "/etc/bash_completion.d" 2>/dev/null; then
        $SUDO cp -f "${src}" "/etc/bash_completion.d/bandcamper"
        $SUDO chmod 644 "/etc/bash_completion.d/bandcamper"
        success "Installed Bash completion: /etc/bash_completion.d/bandcamper"
        installed=true
    fi

    # User-level fallback / complement
    local user_bash_dir="${REAL_HOME}/.local/share/bash-completion/completions"
    mkdir -p "${user_bash_dir}" 2>/dev/null || true
    if [ -d "${user_bash_dir}" ]; then
        cp -f "${src}" "${user_bash_dir}/bandcamper" 2>/dev/null || true
        [ "$(id -u)" -eq 0 ] && [ -n "${SUDO_USER:-}" ] && chown -R "${REAL_USER}:" "${user_bash_dir}" 2>/dev/null || true
    fi

    if [ "${installed}" = true ]; then
        info "Bash completions are ready (restart shell or run: source /usr/share/bash-completion/completions/bandcamper)"
    fi
}

install_zsh_completion() {
    local src="${COMPLETION_DIR}/_bandcamper"
    if [ ! -f "${src}" ]; then
        warn "Zsh completion script not found at '${src}', skipping."
        return
    fi

    local installed=false

    # Try standard system site-functions
    for site_dir in "/usr/local/share/zsh/site-functions" "/usr/share/zsh/site-functions" "/usr/share/zsh/vendor-completions"; do
        if [ -d "${site_dir}" ] || $SUDO mkdir -p "${site_dir}" 2>/dev/null; then
            $SUDO cp -f "${src}" "${site_dir}/_bandcamper"
            $SUDO chmod 644 "${site_dir}/_bandcamper"
            success "Installed Zsh completion: ${site_dir}/_bandcamper"
            installed=true
            break
        fi
    done

    # User-level setup in ~/.zfunc
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

install_fish_completion() {
    local src="${COMPLETION_DIR}/bandcamper.fish"
    if [ ! -f "${src}" ]; then
        warn "Fish completion script not found at '${src}', skipping."
        return
    fi

    local installed=false

    # Try system vendor completions
    if [ -d "/usr/share/fish/vendor_completions.d" ] || $SUDO mkdir -p "/usr/share/fish/vendor_completions.d" 2>/dev/null; then
        $SUDO cp -f "${src}" "/usr/share/fish/vendor_completions.d/bandcamper.fish"
        $SUDO chmod 644 "/usr/share/fish/vendor_completions.d/bandcamper.fish"
        success "Installed Fish completion: /usr/share/fish/vendor_completions.d/bandcamper.fish"
        installed=true
    fi

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
        *bash*)
            install_bash_completion
            ;;
        *zsh*)
            install_zsh_completion
            ;;
        *fish*)
            install_fish_completion
            ;;
        all)
            info "Installing completions for all supported shells (bash, zsh, fish)..."
            install_bash_completion
            install_zsh_completion
            install_fish_completion
            ;;
        *)
            warn "Unrecognized shell '${TARGET_SHELL}'. Falling back to Bash and Zsh."
            install_bash_completion
            install_zsh_completion
            ;;
    esac
fi

echo ""
success "Installation complete! Run 'bandcamper --help' to get started."
