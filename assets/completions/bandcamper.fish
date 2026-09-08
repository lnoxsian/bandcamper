# fish completion for bandcamper

# Disable file completion for arguments unless explicitly handled
complete -c bandcamper -f

# Help & Version
complete -c bandcamper -s h -l help -d "Show help message and exit"
complete -c bandcamper -l version -d "Show version information and exit"

# Options taking files/directories
complete -c bandcamper -l config -r -F -d "Path to configuration file"
complete -c bandcamper -s f -l file -r -F -d "File containing Bandcamp URLs"
complete -c bandcamper -s o -l output -r -a "(__fish_complete_directories)" -d "Output directory for downloads"

# Options taking specific values or numbers
complete -c bandcamper -s j -l jobs -r -a "1 2 4 8 16" -d "Number of concurrent download workers"
complete -c bandcamper -l retry -r -a "0 1 2 3 5" -d "Number of retries for failed downloads"
complete -c bandcamper -l playlist -r -a "m3u pls wpl zpl" -d "Playlist format to generate"
complete -c bandcamper -l filename -r -d "Filename format template"
complete -c bandcamper -l directory -r -d "Directory format template"

# Boolean switches
complete -c bandcamper -l skip-existing -d "Skip downloading existing files"
complete -c bandcamper -l overwrite -d "Overwrite existing files"
complete -c bandcamper -l no-tags -d "Disable ID3 metadata embedding"
complete -c bandcamper -l no-artwork -d "Disable embedding artwork into audio files"
complete -c bandcamper -l no-lyrics -d "Disable embedding lyrics into audio files"
complete -c bandcamper -l no-save-artwork -d "Do not save cover artwork file alongside music"
complete -c bandcamper -l dry-run -d "Simulate operations without downloading files"
complete -c bandcamper -s v -l verbose -d "Enable verbose debug output"
complete -c bandcamper -s q -l quiet -d "Suppress normal output except errors"
complete -c bandcamper -l color -d "Enable ANSI colored terminal output"
complete -c bandcamper -l no-color -d "Disable ANSI colored terminal output"
