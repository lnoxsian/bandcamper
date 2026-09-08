# bash completion for bandcamper                         -*- shell-script -*-

_bandcamper() {
    local cur prev words cword
    if type _init_completion >/dev/null 2>&1; then
        _init_completion -n : || return
    else
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
    fi

    local opts=(
        --config
        -f --file
        -o --output
        -j --jobs
        --retry
        --filename
        --directory
        --playlist
        --skip-existing
        --overwrite
        --no-tags
        --no-artwork
        --no-lyrics
        --no-save-artwork
        --dry-run
        -v --verbose
        -q --quiet
        --color
        --no-color
        --version
        -h --help
    )

    case "${prev}" in
        --config)
            local configs=( $(compgen -f -X '!*.toml' -- "${cur}") )
            if [ ${#configs[@]} -gt 0 ]; then
                COMPREPLY=( "${configs[@]}" )
            else
                COMPREPLY=( $(compgen -f -- "${cur}") )
            fi
            return 0
            ;;
        -f|--file)
            COMPREPLY=( $(compgen -f -- "${cur}") )
            return 0
            ;;
        -o|--output)
            COMPREPLY=( $(compgen -d -- "${cur}") )
            return 0
            ;;
        --playlist)
            COMPREPLY=( $(compgen -W "m3u pls wpl zpl" -- "${cur}") )
            return 0
            ;;
        -j|--jobs)
            COMPREPLY=( $(compgen -W "1 2 4 8 16" -- "${cur}") )
            return 0
            ;;
        --retry)
            COMPREPLY=( $(compgen -W "0 1 2 3 5" -- "${cur}") )
            return 0
            ;;
        --filename)
            COMPREPLY=( $(compgen -W '"{tracknumber} - {title}.mp3" "{artist} - {title}.mp3"' -- "${cur}") )
            return 0
            ;;
        --directory)
            COMPREPLY=( $(compgen -W '"{artist}/{album}" "{artist}" "{album}"' -- "${cur}") )
            return 0
            ;;
    esac

    if [[ "${cur}" == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts[*]}" -- "${cur}") )
        return 0
    fi
}

complete -F _bandcamper bandcamper
