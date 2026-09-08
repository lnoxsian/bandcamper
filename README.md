# Bandcamper

A fast, lightweight, cross-platform Bandcamp downloader written in Go, inspired by the feature set of [Otiel/BandcampDownloader](https://github.com/Otiel/BandcampDownloader).

Bandcamper is a single-binary CLI capable of resolving Bandcamp artist, album, and track URLs, downloading audio streams concurrently, embedding complete ID3v2 metadata (including cover artwork and lyrics), and generating playlists.

---

## Features

- **URL Resolution**: Automatic classification and normalization for:
  - Bandcamp artist pages (`https://artist.bandcamp.com` or `https://artist.bandcamp.com/music`)
  - Bandcamp album pages (`https://artist.bandcamp.com/album/example-album`)
  - Bandcamp track pages (`https://artist.bandcamp.com/track/example-track`)
- **Resilient Metadata Extraction**: Decodes embedded JSON (`TralbumData`), JSON-LD (`application/ld+json`), and OpenGraph metadata.
- **Concurrent Downloads**: Configurable worker pool for downloading multiple tracks simultaneously.
- **Robust Network Handling**: Exponential backoff retry for transient network errors and rate limits (408, 429, 500, 502, 503, 504).
- **Graceful Cancellation**: Cancel active downloads cleanly using `Ctrl+C`.
- **Safe Storage**: Atomic downloads via temporary `.part` files to prevent partial or corrupted files.
- **Duplicate Protection**: Skip existing files with `--skip-existing` or overwrite with `--overwrite`.
- **ID3v2 Metadata & Lyrics**: Embeds Title, Artist, Album, Album Artist, Track Number/Total, Year, Genre, Comment, and synchronized/unsynchronized lyrics.
- **Embedded & Saved Artwork**: High-resolution cover artwork is embedded into every MP3 and saved alongside the album as `cover.jpg`.
- **Custom Templates**: Fully customizable output directory and filename patterns (e.g. `{artist}/{album}/{tracknumber} - {title}.mp3`).
- **Playlists**: Generates M3U, PLS, WPL, and ZPL playlists automatically.
- **Dry-Run Mode**: Inspect metadata and output paths without downloading anything.
- **Zero Heavy Runtimes**: Compiled directly to a single native binary (no Python, Node.js, or .NET required).

---

## Installation

### From Source

```bash
git clone https://github.com/lnoxsian/bandcamper.git
cd bandcamper
go build -o bandcamper ./cmd/bandcamper
```

### Or using `go install`

```bash
go install github.com/lnoxsian/bandcamper/cmd/bandcamper@latest
```

---

## Usage

### Basic Commands

Download an entire album:
```bash
bandcamper https://artist.bandcamp.com/album/example-album
```

Download a single track:
```bash
bandcamper https://artist.bandcamp.com/track/example-track
```

Download all releases by an artist (entire discography):
```bash
bandcamper https://artist.bandcamp.com
```

Download multiple URLs:
```bash
bandcamper https://artist.bandcamp.com/album/one https://artist.bandcamp.com/album/two
```

Download URLs from a text file:
```bash
bandcamper --file urls.txt
```

---

## CLI Options

| Flag | Shorthand | Default | Description |
| :--- | :--- | :--- | :--- |
| `--output <dir>` | `-o` | `~/Music` | Output root directory |
| `--jobs <n>` | `-j` | `4` | Number of concurrent download workers |
| `--retry <n>` | | `3` | Maximum retry attempts for transient failures |
| `--filename <tmpl>` | | `{tracknumber} - {title}.mp3` | Filename template |
| `--directory <tmpl>` | | `{artist}/{album}` | Directory template |
| `--playlist <fmt>` | | `""` | Generate playlist (`m3u`, `pls`, `wpl`, `zpl`) |
| `--skip-existing` | | `true` | Skip downloading already existing files |
| `--overwrite` | | `false` | Overwrite existing files |
| `--dry-run` | | `false` | Preview files without downloading |
| `--no-tags` | | `false` | Disable ID3 metadata embedding |
| `--no-artwork` | | `false` | Disable embedding artwork into audio |
| `--no-lyrics` | | `false` | Disable embedding lyrics into audio |
| `--no-save-artwork` | | `false` | Do not save `cover.jpg` file |
| `--config <file>` | | *(OS default)* | Path to custom TOML config file |
| `-v, --verbose` | | `false` | Enable verbose debug output |
| `-q, --quiet` | | `false` | Suppress output except errors |
| `--version` | | | Display version information |
| `--help` | `-h` | | Show help message |

---

## Templates

Supported template placeholders for `--filename` and `--directory`:

- `{artist}`: Track or release artist
- `{album}`: Album or release title
- `{albumartist}`: Album artist
- `{title}`: Track title
- `{tracknumber}`: Track number (zero-padded, e.g. `01`, `02`)
- `{tracktotal}`: Total number of tracks in release
- `{year}`: Release year (e.g. `2024`)
- `{genre}`: Genre tag

### Examples

```bash
# Save into Year - Album folders
bandcamper --directory "{artist}/{year} - {album}" https://artist.bandcamp.com/album/example

# Filename with artist and track number
bandcamper --filename "{artist} - {tracknumber} - {title}.mp3" https://artist.bandcamp.com/album/example
```

---

## Configuration File

Bandcamper can be configured using a `config.toml` file.

Default paths:
- **Linux/BSD**: `~/.config/bandcamper/config.toml`
- **macOS**: `~/Library/Application Support/bandcamper/config.toml`
- **Windows**: `%APPDATA%\bandcamper\config.toml`

### Example `config.toml`

```toml
output = "~/Music"
workers = 4
retry_count = 3
skip_existing = true
overwrite = false

directory = "{artist}/{album}"
filename = "{tracknumber} - {title}.mp3"

embed_artwork = true
save_artwork = true
embed_lyrics = true
embed_tags = true

playlist = "m3u"
```

---

## Exit Codes

Bandcamper returns meaningful exit codes suitable for automation and scripting:

- `0`: Success
- `1`: General failure
- `2`: Invalid command line arguments
- `3`: Invalid Bandcamp URL
- `4`: Parsing failure
- `5`: Download failure
- `6`: Metadata failure

---

## Testing

Run unit and integration test suites:

```bash
go test ./...
go test -race ./...
go vet ./...
```

---

## License

MIT License. See [LICENSE](LICENSE) for details.
