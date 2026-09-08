# Configuration Reference

Bandcamper loads defaults, then an optional TOML configuration file, then applies command-line overrides. A file passed with `--config` is required to exist and parse successfully. When no explicit file is supplied, a missing default file is allowed and built-in defaults are used.

## Configuration file locations

The default path is platform-specific:

| Platform | Path |
| --- | --- |
| Linux, BSD, and other POSIX systems | `$XDG_CONFIG_HOME/bandcamper/config.toml`, or `~/.config/bandcamper/config.toml` when `XDG_CONFIG_HOME` is unset |
| macOS | `~/Library/Application Support/bandcamper/config.toml` |
| Windows | `%APPDATA%\\bandcamper\\config.toml` |

A custom path can be supplied with `--config`. A leading `~/` is expanded.

## TOML example

```toml
output = "~/Music"
workers = 4
retry_count = 3
retry_delay = "1s"
skip_existing = true
overwrite = false

filename = "{tracknumber} - {title}.mp3"
directory = "{artist}/{album}"

embed_artwork = true
save_artwork = true
embed_lyrics = true
embed_tags = true

playlist = "m3u"
dry_run = false
verbose = false
quiet = false
color = true

user_agent = "Mozilla/5.0"
timeout = "30s"
```

`retry_delay` and `timeout` use Go duration syntax, such as `500ms`, `1s`, or `30s`.

## Built-in defaults

| Setting | Default |
| --- | --- |
| `output` | `~/Music` |
| `workers` | `4` |
| `retry_count` | `3` |
| `retry_delay` | `1s` |
| `skip_existing` | `true` |
| `overwrite` | `false` |
| `filename` | `{tracknumber} - {title}.mp3` |
| `directory` | `{artist}/{album}` |
| `embed_artwork` | `true` |
| `save_artwork` | `true` |
| `embed_lyrics` | `true` |
| `embed_tags` | `true` |
| `playlist` | empty, so no playlist is generated |
| `dry_run` | `false` |
| `verbose` | `false` |
| `quiet` | `false` |
| `color` | `true` |
| `timeout` | `30s` |

## Command-line options

| Option | Meaning |
| --- | --- |
| `--output`, `-o` | Output root directory. |
| `--file`, `-f` | Read URLs from a text file. Blank lines and `#` comments are ignored. |
| `--jobs`, `-j` | Number of concurrent download workers. |
| `--retry` | Number of retries for failed downloads. |
| `--filename` | Track filename template. |
| `--directory` | Release directory template. |
| `--playlist` | Playlist format: `m3u`, `pls`, `wpl`, or `zpl`. |
| `--skip-existing` | Enable skipping of existing files. |
| `--overwrite` | Allow existing files to be overwritten. |
| `--no-tags` | Do not write ID3 metadata. |
| `--no-artwork` | Do not embed artwork in audio files. |
| `--no-lyrics` | Do not embed lyrics in audio files. |
| `--no-save-artwork` | Do not save the separate `cover.jpg`. |
| `--dry-run` | Resolve metadata and print planned output without downloading. |
| `--config` | Use a specific TOML configuration file. |
| `--verbose`, `-v` | Enable verbose output. |
| `--quiet`, `-q` | Suppress normal output except errors. |
| `--color` | Enable colored output. |
| `--no-color` | Disable colored output. |
| `--version` | Print the version and exit. |
| `--help`, `-h` | Print command usage and exit. |

Boolean command-line flags are additive for most settings: for example, `--no-tags` turns tagging off, while `--overwrite` turns overwriting on. String and numeric flags override their TOML values when provided.

## Templates

The following tokens are available in `filename` and `directory` templates:

| Token | Value |
| --- | --- |
| `{artist}` | Track or release artist |
| `{album}` | Release title |
| `{albumartist}` | Album artist, falling back to artist |
| `{title}` | Track title; primarily useful in filenames |
| `{tracknumber}` | Zero-padded track number, normally `01`, `02`, and so on |
| `{tracktotal}` | Total track count; primarily useful in filenames |
| `{year}` | Four-digit release year, when available |
| `{genre}` | Release genre, when available |

Directory separators can be written with `/` or `\\`. Each path component is sanitized independently. Filenames always use the `.mp3` extension, even when the filename template omits it.

Examples:

```sh
bandcamper \
  --directory "{artist}/{year} - {album}" \
  --filename "{tracknumber} - {title}.mp3" \
  URL
```

```toml
directory = "{genre}/{artist}/{album}"
filename = "{tracknumber} - {artist} - {title}.mp3"
```

## Precedence and environment

The effective configuration is resolved in this order:

1. Built-in defaults.
2. The default TOML file, if present, or the explicit file passed with `--config`.
3. Command-line overrides.
4. `NO_COLOR`, `--no-color`, or `--color=false` behavior for terminal color.

The application does not expose TOML settings for arbitrary HTTP headers. `user_agent` and `timeout` are the supported network-related settings.
