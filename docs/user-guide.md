# User Guide

## Installation

### Pre-compiled Release Packages

Download the release `.tar.gz` for your architecture from GitHub Releases, extract it, and execute the bundled installer:

```sh
tar -xzf bandcamper-linux-amd64.tar.gz
sudo ./install.sh
```

The script automatically:
1. Installs the `bandcamper` executable to `/usr/bin/bandcamper`.
2. Inspects your active shell (`bash`, `zsh`, or `fish`).
3. Installs and registers the appropriate tab-completion script to the system/user completions directory.

### Build from source

Requirements:

- Go 1.27.1 or a compatible newer Go toolchain.
- Network access to the Bandcamp pages and audio streams you intend to download.

From the repository root:

```sh
go build -o bandcamper ./cmd/bandcamper
```

The project also provides `make` and `just` recipes. See the [development guide](development.md).

### Install with `go install`

```sh
go install github.com/lnoxsian/bandcamper/cmd/bandcamper@latest
```

Make sure the Go install directory is on your `PATH`.

### Shell Completions

Pre-generated completion scripts for Bash, Zsh, and Fish are provided in [`assets/completions/`](../assets/completions):

- **Bash**:
  ```sh
  # Temporary / test
  source assets/completions/bandcamper.bash
  # Permanent
  sudo cp assets/completions/bandcamper.bash /etc/bash_completion.d/bandcamper
  ```
- **Zsh**:
  ```sh
  # Copy to a site-functions or fpath directory:
  cp assets/completions/_bandcamper ~/.zfunc/   # ensure ~/.zfunc is in fpath in .zshrc
  # Or source directly in ~/.zshrc:
  source /path/to/assets/completions/bandcamper.zsh
  ```
- **Fish**:
  ```sh
  cp assets/completions/bandcamper.fish ~/.config/fish/completions/
  ```

## Supported inputs

Bandcamper accepts one or more positional URLs:

```text
https://artist.bandcamp.com/
https://artist.bandcamp.com/album/album-name
https://artist.bandcamp.com/track/track-name
```

An artist URL expands to the releases found in the artist's public discography. An album URL downloads that release. A track URL downloads the release context containing that track.

Multiple URLs can be supplied in one invocation:

```sh
bandcamper \
  https://artist.bandcamp.com/album/first-release \
  https://artist.bandcamp.com/album/second-release
```

URLs can also be read from a file. Empty lines and lines beginning with `#` are ignored:

```text
# releases.txt
https://artist.bandcamp.com/album/first-release
https://artist.bandcamp.com/track/single
```

```sh
bandcamper --file releases.txt
```

## Common workflows

Download an album:

```sh
bandcamper https://artist.bandcamp.com/album/example-album
```

Download an artist discography:

```sh
bandcamper https://artist.bandcamp.com
```

Choose an output directory and worker count:

```sh
bandcamper --output ~/Music --jobs 4 URL
```

Generate a playlist alongside the downloaded tracks:

```sh
bandcamper --playlist m3u URL
```

Preview metadata and output filenames without creating downloads:

```sh
bandcamper --dry-run URL
```

Cancel active work with `Ctrl+C`. The process cancels its context and stops active downloads as they observe cancellation.

## Output behavior

By default, a release is written below `~/Music/{artist}/{album}`. Each track is written as an MP3 using `{tracknumber} - {title}.mp3`.

For each release, Bandcamper may also:

- Embed ID3v2 tags in each MP3.
- Embed cover artwork and lyrics when available.
- Save artwork as `cover.jpg` in the release directory.
- Generate an M3U, PLS, WPL, or ZPL playlist when requested.

Downloads use temporary `.part` files and are moved into place only after the operation completes. Existing files are skipped by default. Use `--overwrite` to replace them.

Filename and directory values are sanitized for common filesystem restrictions, including invalid characters and Windows reserved names.

## Logging and terminal output

Use `--verbose` for diagnostic output and `--quiet` to suppress normal progress output. Color is enabled by default and can be disabled with `--no-color` or the standard `NO_COLOR` environment variable.

Show the built-in help or version:

```sh
bandcamper --help
bandcamper --version
```

## Exit codes

| Code | Meaning |
| ---: | --- |
| 0 | Completed successfully |
| 1 | General failure, such as a configuration error |
| 2 | Invalid command-line arguments or missing input |
| 3 | Invalid Bandcamp URL |
| 4 | Page parsing failure |
| 5 | Download failure or one or more failed tracks |
| 6 | Reserved metadata failure code |

## Troubleshooting

### A URL is rejected

Check that the URL uses HTTPS and points to an artist, album, or track page on a `bandcamp.com` host. Remove surrounding punctuation or tracking text and try again.

### A download fails repeatedly

Transient network failures are retried with exponential backoff. Check connectivity, lower `--jobs`, and try again later. A release may also be unavailable as a public stream.

### Files are skipped

The default behavior is to skip an existing destination file. Use `--overwrite` when the existing file should be replaced. Avoid combining `--skip-existing` and `--overwrite`; if both are enabled, the downloader's existing-file policy should be treated as configuration-dependent and should be tested with a dry run first.

### Metadata or artwork is incomplete

Bandcamp page metadata is the source of truth. Some releases do not expose lyrics, artwork, genre, release dates, or playable streams. Disable individual features with `--no-tags`, `--no-artwork`, `--no-lyrics`, or `--no-save-artwork` when needed.
