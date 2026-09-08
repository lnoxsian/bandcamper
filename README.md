
<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logo/logo_light.png">
    <source media="(prefers-color-scheme: light)" srcset="./assets/logo/logo_dark.png">
    <img alt="Bandcamper logo" src="./assets/logo/logo_dark.png" width="50%">
  </picture>
</div>

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

### Pre-compiled Packages & Binaries

Download from [GitHub Releases](https://github.com/lnoxsian/bandcamper/releases):

**Debian / Ubuntu (`.deb`)**:
```bash
sudo dpkg -i bandcamper_*.deb
```

**Fedora / RHEL / CentOS (`.rpm`)**:
```bash
sudo rpm -i bandcamper-*.rpm
```

**Standalone Tarball (`.tar.gz`)**:
```bash
tar -xzf bandcamper-linux-amd64.tar.gz
sudo ./install.sh
```

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
# Bandcamper

Bandcamper is a lightweight, cross-platform Bandcamp downloader written in Go. It resolves public artist, album, and track pages, downloads available MP3 streams, writes metadata and artwork, and can generate playlists.

## Quick start

Build from source:

```sh
go build -o bandcamper ./cmd/bandcamper
```

Download an album:

```sh
./bandcamper https://artist.bandcamp.com/album/example-album
```

Preview the resolved metadata and output paths without downloading:

```sh
./bandcamper --dry-run https://artist.bandcamp.com/album/example-album
```

## Documentation

See the [documentation index](docs/README.md) for the complete guides:

- [User guide](docs/user-guide.md): installation, supported URLs, usage, output behavior, and troubleshooting.
- [Configuration reference](docs/configuration.md): CLI flags, TOML settings, templates, and precedence.
- [Development guide](docs/development.md): repository layout, tests, builds, and release targets.
- [Implementation plan](docs/plan.md): the original project plan and design direction.

## Highlights

- Concurrent downloads with retry handling and cancellation.
- Atomic file storage with existing-file protection.
- ID3v2 tags, lyrics, embedded artwork, and saved cover art.
- M3U, PLS, WPL, and ZPL playlist generation.
- Configurable output directories and filename templates.
- Native binaries for Linux, macOS, and Windows on `amd64` and `arm64`.

Bandcamper does not bypass authentication, DRM, purchase restrictions, or other access controls. Downloads remain subject to Bandcamp's terms and the rights held by the artist or label.

## License

MIT License. See [LICENSE](LICENSE) for details.

## Credits

Bandcamper was inspired by the feature set and overall idea of [Otiel/BandcampDownloader](https://github.com/Otiel/BandcampDownloader). Credit and thanks to its author, [Otiel](https://github.com/Otiel), for the original project.
Bandcamper can be configured using a `config.toml` file.
