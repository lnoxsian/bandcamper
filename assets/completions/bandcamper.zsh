#compdef bandcamper

# Sourceable zsh completion wrapper for bandcamper
# Allows sourcing directly in .zshrc: source path/to/bandcamper.zsh

0="${${0:#$ZSH_ARGZERO}:-${(%):-%N}}"
completion_dir="${0:A:h}"

if [[ -f "${completion_dir}/_bandcamper" ]]; then
    source "${completion_dir}/_bandcamper"
    compdef _bandcamper bandcamper 2>/dev/null || true
fi
