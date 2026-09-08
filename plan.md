# Bandcamp Downloader — Go Implementation Plan

## 1. Project Goal

Build a modern, cross-platform, fully featured Bandcamp downloader in Go, inspired by the feature set of `Otiel/BandcampDownloader`.

Reference project:

* https://github.com/Otiel/BandcampDownloader

The application should be a lightweight, single-binary CLI capable of resolving Bandcamp artist, album, and track URLs, downloading the available audio streams, embedding complete metadata and artwork, and generating playlists.

The implementation must be modular so the downloader engine can later be reused by a GUI application.

---

# 2. Core Requirements

The application must support:

* Bandcamp artist URLs
* Bandcamp album URLs
* Bandcamp track URLs
* Album/release metadata extraction
* Track metadata extraction
* Audio downloading
* Concurrent downloads
* Retry handling
* Download cancellation
* Progress reporting
* Existing-file detection
* ID3 metadata embedding
* Lyrics embedding when available
* Embedded cover artwork
* Cover artwork saved alongside music
* Configurable directory structure
* Configurable filename templates
* Playlist generation
* M3U playlists
* PLS playlists
* WPL playlists
* ZPL playlists
* Configuration file
* Dry-run mode
* Verbose/debug logging
* Clean CLI UX
* Cross-platform builds
* Unit and integration tests

The downloader must not attempt to bypass authentication, DRM, purchase restrictions, or other access controls.

---

# 3. Technology

Use:

* Go
* `net/http`
* `context`
* `encoding/json`
* `net/url`
* `html`
* `image`
* `os`
* `path/filepath`
* `sync`
* `time`

Keep external dependencies minimal.

Use established Go libraries for ID3/audio metadata handling rather than implementing the ID3 specification from scratch.

Do not introduce a web framework.

---

# 4. Repository Structure

Create the following structure:

```text
bandcamp-dl/
├── cmd/
│   └── bandcamp-dl/
│       └── main.go
│
├── internal/
│   ├── bandcamp/
│   │   ├── client.go
│   │   ├── parser.go
│   │   ├── resolver.go
│   │   ├── models.go
│   │   ├── album.go
│   │   ├── track.go
│   │   └── artist.go
│   │
│   ├── downloader/
│   │   ├── downloader.go
│   │   ├── worker.go
│   │   ├── retry.go
│   │   └── progress.go
│   │
│   ├── metadata/
│   │   ├── metadata.go
│   │   ├── id3.go
│   │   ├── artwork.go
│   │   └── lyrics.go
│   │
│   ├── playlist/
│   │   ├── playlist.go
│   │   ├── m3u.go
│   │   ├── pls.go
│   │   ├── wpl.go
│   │   └── zpl.go
│   │
│   ├── storage/
│   │   ├── paths.go
│   │   ├── filenames.go
│   │   └── atomic.go
│   │
│   └── config/
│       └── config.go
│
├── tests/
│   ├── fixtures/
│   ├── bandcamp/
│   ├── metadata/
│   ├── downloader/
│   └── playlist/
│
├── docs/
│
├── go.mod
├── go.sum
├── README.md
├── LICENSE
└── plan.md
```

Keep `cmd/` thin.

All business logic belongs in `internal/`.

---

# 5. Data Model

Create a normalized internal representation independent of the HTML parser.

## Release

```go
type Release struct {
    URL          string
    Artist       string
    Album        string
    AlbumArtist  string
    Description  string
    Genre        string
    ReleaseDate  time.Time
    ArtworkURL   string
    Tracks       []Track
}
```

## Track

```go
type Track struct {
    Number       int
    Title        string
    Artist       string
    Album        string
    Duration     time.Duration
    StreamURL    string
    Lyrics       string
}
```

## Metadata

Create a separate metadata representation:

```go
type Metadata struct {
    Title        string
    Artist       string
    Album        string
    AlbumArtist  string
    TrackNumber  int
    TrackTotal   int
    Year         int
    Genre        string
    Lyrics       string
    Comment      string
    Artwork      []byte
    ArtworkMIME  string
}
```

The rest of the application must not depend directly on parsed Bandcamp HTML.

---

# 6. Bandcamp URL Resolver

Implement automatic URL classification.

Supported forms:

```text
https://artist.bandcamp.com/

https://artist.bandcamp.com/album/album-name

https://artist.bandcamp.com/track/track-name
```

Create:

```go
type PageType int

const (
    PageUnknown PageType = iota
    PageArtist
    PageAlbum
    PageTrack
)
```

The resolver must:

1. Parse the URL.
2. Validate that it is a Bandcamp URL.
3. Determine the page type.
4. Normalize the URL.
5. Pass it to the appropriate parser.

Reject unsupported URLs cleanly.

---

# 7. Bandcamp Parser

Implement a resilient parser.

Do not depend exclusively on visible HTML text.

Look for structured metadata embedded in Bandcamp pages, including JSON/state data where available.

The parser should extract:

### Release

* Artist
* Album
* Album artist
* Release date
* Genre
* Description
* Artwork URL
* Track count

### Track

* Track number
* Title
* Artist
* Duration
* Stream URL
* Lyrics when available

The parser must tolerate:

* Missing metadata
* Missing artwork
* Missing lyrics
* Single-track releases
* Bonus tracks
* Untitled tracks
* Unicode text
* HTML changes

Do not panic when optional fields are missing.

Return descriptive errors.

---

# 8. HTTP Client

Create a reusable Bandcamp HTTP client.

Requirements:

* Configurable timeout
* Context cancellation
* HTTP connection reuse
* Redirect support
* Appropriate User-Agent
* Response size limits where appropriate
* Proper status-code handling

Example API:

```go
type Client struct {
    HTTPClient *http.Client
    UserAgent  string
}
```

Implement:

```go
Get(ctx context.Context, url string) (*http.Response, error)
```

All network operations must accept a `context.Context`.

---

# 9. Downloader Engine

Create a concurrent worker pool.

Example configuration:

```go
type DownloadConfig struct {
    Workers       int
    RetryCount    int
    RetryDelay    time.Duration
    SkipExisting  bool
    Overwrite     bool
}
```

Pipeline:

```text
Resolve URL
     ↓
Parse release
     ↓
Create output directory
     ↓
Create track jobs
     ↓
Worker pool
     ↓
Download audio
     ↓
Write temporary file
     ↓
Embed metadata
     ↓
Embed artwork
     ↓
Atomic rename
     ↓
Playlist generation
```

Never write directly to the final filename.

Use:

```text
track.mp3.part
        ↓
metadata processing
        ↓
track.mp3
```

If a download fails, the partial file must be removed or safely retained according to future resume support.

---

# 10. Retry System

Implement exponential backoff.

Retry transient failures such as:

* HTTP 408
* HTTP 429
* HTTP 500
* HTTP 502
* HTTP 503
* HTTP 504
* temporary network failures

Do not retry permanent errors unnecessarily.

Example:

```text
attempt 1 → 1 second
attempt 2 → 2 seconds
attempt 3 → 4 seconds
attempt 4 → 8 seconds
```

Make retry count configurable.

---

# 11. Download Progress

Provide clear terminal progress.

Example:

```text
Album: Example Album
Artist: Example Artist

[1/10] Track One       ████████████████ 100%
[2/10] Track Two       ██████████░░░░░░  62%
[3/10] Track Three     waiting

Downloaded: 2 / 10
Speed: 4.2 MB/s
```

Do not make progress rendering part of the downloader engine.

Expose progress events through a channel/callback interface.

This allows a future GUI to consume the same events.

---

# 12. Metadata Embedding

Implement a dedicated metadata package.

For MP3 files embed:

* Title
* Artist
* Album
* Album Artist
* Track Number
* Total Tracks
* Year
* Genre
* Lyrics
* Comment
* Artwork

Use proper ID3v2 tags.

Artwork should use the appropriate attached-picture frame.

Preserve existing audio data.

Never re-encode the audio unnecessarily.

The downloader should perform:

```text
download MP3
    ↓
open MP3
    ↓
read existing tags
    ↓
update required tags
    ↓
add artwork
    ↓
write tags
```

---

# 13. Lyrics

If lyrics are available from Bandcamp:

* Extract them during parsing.
* Preserve Unicode.
* Strip unnecessary HTML.
* Normalize line endings.
* Embed them in the appropriate lyrics tag.

If lyrics are unavailable:

* Do not fail the download.
* Leave the lyrics field empty.

---

# 14. Artwork

Artwork processing must support:

* JPEG
* PNG
* Other formats supported by the image decoder

Validate downloaded image data.

Store artwork as:

```text
cover.jpg
```

by default.

Embed artwork into every track.

Do not download the same artwork once for every track.

Use a release-level artwork cache.

Example:

```text
Album/
├── cover.jpg
├── 01 - Track One.mp3
├── 02 - Track Two.mp3
└── 03 - Track Three.mp3
```

---

# 15. Filesystem Layout

Default layout:

```text
{output}/{artist}/{album}/{tracknumber} - {title}.mp3
```

Example:

```text
Music/
└── Artist Name/
    └── Album Name/
        ├── cover.jpg
        ├── 01 - First Track.mp3
        ├── 02 - Second Track.mp3
        └── 03 - Third Track.mp3
```

Make this configurable.

Supported placeholders:

```text
{artist}
{album}
{albumartist}
{title}
{tracknumber}
{tracktotal}
{year}
{genre}
```

Sanitize filenames for:

* Linux
* macOS
* Windows

Handle reserved filenames and problematic Unicode safely.

---

# 16. Existing Files

Support:

```text
--skip-existing
--overwrite
```

Default behavior should be safe and avoid destroying existing files.

Before downloading:

1. Determine final path.
2. Check whether it exists.
3. Skip if configured.
4. Otherwise download to a temporary path.
5. Replace atomically only after successful processing.

---

# 17. Playlist Generation

Create a generic playlist interface:

```go
type Writer interface {
    Write(path string, tracks []string) error
}
```

Implement:

```text
M3U
PLS
WPL
ZPL
```

Playlist generation occurs only after downloads finish.

Only include successfully downloaded/skipped-valid tracks.

Respect relative paths where possible.

---

# 18. Configuration

Provide a configuration file.

Possible location:

```text
Linux:
~/.config/bandcamp-dl/config.toml

macOS:
~/Library/Application Support/bandcamp-dl/config.toml

Windows:
%APPDATA%\bandcamp-dl\config.toml
```

Configuration should include:

```toml
output = "~/Music"
workers = 4
retry_count = 3
skip_existing = true
overwrite = false

filename = "{tracknumber} - {title}.mp3"
directory = "{artist}/{album}"

embed_artwork = true
save_artwork = true
embed_lyrics = true

playlist = "m3u"
```

CLI arguments must override configuration values.

---

# 19. CLI

Command:

```bash
bandcamp-dl <URL>
```

Support multiple URLs:

```bash
bandcamp-dl <URL1> <URL2> <URL3>
```

Support URL files:

```bash
bandcamp-dl --file urls.txt
```

Options:

```text
-o, --output
-j, --jobs
--retry
--format
--filename
--directory

--skip-existing
--overwrite

--no-tags
--no-artwork
--no-lyrics
--no-save-artwork

--playlist
--dry-run

-v, --verbose
-q, --quiet

--config
--version
--help
```

Example:

```bash
bandcamp-dl \
  --output ~/Music \
  --jobs 4 \
  --playlist m3u \
  https://artist.bandcamp.com/album/example
```

---

# 20. Dry Run

Implement:

```bash
bandcamp-dl --dry-run URL
```

It should parse the page and show:

```text
Artist: Example Artist
Album: Example Album
Tracks: 10

Output:
~/Music/Example Artist/Example Album/

01 - Track One.mp3
02 - Track Two.mp3
...
```

No files should be downloaded.

This is also useful for debugging parser changes.

---

# 21. Logging

Use structured internal logging.

Levels:

```text
ERROR
WARN
INFO
DEBUG
```

Normal output should remain clean.

Verbose mode should expose:

* HTTP requests
* parser decisions
* extracted metadata
* download retries
* file paths
* metadata operations

Do not log sensitive headers or credentials.

---

# 22. Error Handling

Never panic for normal runtime failures.

Errors should provide useful context:

```text
failed to parse album:
    https://artist.bandcamp.com/album/example

failed to download track:
    Track Name

failed to write metadata:
    /Music/Artist/Album/01 - Track.mp3
```

Use wrapped errors:

```go
fmt.Errorf("download track %q: %w", track.Title, err)
```

Define errors for important cases:

```go
ErrInvalidURL
ErrUnsupportedPage
ErrParseFailed
ErrDownloadFailed
ErrArtworkFailed
ErrMetadataFailed
```

---

# 23. Testing

Testing is mandatory.

## Unit Tests

Test:

* URL classification
* URL normalization
* filename sanitization
* path templates
* metadata mapping
* playlist generation
* retry behavior
* configuration loading
* configuration overrides

## Parser Tests

Store representative Bandcamp HTML/JSON responses as fixtures.

Test:

* albums
* tracks
* artist pages
* Unicode artists
* missing artwork
* missing lyrics
* single-track releases
* releases with many tracks

Do not make normal unit tests depend on live Bandcamp.

## Downloader Tests

Use `httptest.Server`.

Test:

* successful download
* failed download
* retries
* HTTP errors
* cancellation
* concurrent workers
* existing files
* atomic writes

## Metadata Tests

Verify generated MP3 files contain:

```text
Title
Artist
Album
Album Artist
Track Number
Year
Lyrics
Artwork
```

## Playlist Tests

Verify valid:

```text
.m3u
.pls
.wpl
.zpl
```

output.

---

# 24. Concurrency Safety

Run:

```bash
go test -race ./...
```

No data races are acceptable.

Ensure:

* progress state is synchronized
* artwork cache is synchronized
* shared configuration is immutable after initialization
* worker shutdown is deterministic

Use `context.Context` for cancellation.

---

# 25. Security and Robustness

The application must:

* Validate URLs.
* Avoid path traversal through metadata.
* Sanitize filenames.
* Limit excessive response sizes where appropriate.
* Avoid arbitrary filesystem writes.
* Never execute downloaded content.
* Never trust HTML metadata as filesystem paths.
* Handle malformed server responses safely.

Do not implement mechanisms intended to bypass Bandcamp access controls.

---

# 26. CLI Exit Codes

Use meaningful exit codes.

```text
0 = success

1 = general failure
2 = invalid command/arguments
3 = invalid Bandcamp URL
4 = parsing failure
5 = download failure
6 = metadata failure
```

For multiple URLs, continue processing independent URLs where possible and return a non-zero exit code if any operation failed.

---

# 27. Performance Goals

The application should remain lightweight.

Target:

* Single native binary
* No runtime dependency on Python/Node/.NET
* Low memory usage
* Connection reuse
* Concurrent downloads
* Streaming downloads
* No unnecessary audio re-encoding
* Artwork downloaded once per release
* Minimal external dependencies

Do not over-engineer the initial implementation.

---

# 28. Architecture Rules

Follow these rules throughout development:

1. Keep Bandcamp parsing isolated.
2. Keep HTTP handling isolated from parsing.
3. Keep downloading isolated from metadata processing.
4. Keep metadata handling isolated from filesystem handling.
5. Keep playlist generation independent.
6. CLI code must not contain business logic.
7. All network operations must support context cancellation.
8. All filesystem writes must be safe and atomic where appropriate.
9. Optional metadata must never cause the entire download to fail.
10. Avoid global mutable state.
11. Prefer interfaces only where they provide actual extensibility.
12. Do not add unnecessary abstractions.
13. Keep the core library usable without the CLI.

---

# 29. Development Phases

## Phase 1 — Project Bootstrap

Create:

* Go module
* Repository structure
* CLI entry point
* Configuration system
* Logging
* Version information

Deliverable:

```bash
bandcamp-dl --help
bandcamp-dl --version
```

---

## Phase 2 — URL Resolver

Implement:

* Bandcamp URL validation
* Page type detection
* URL normalization

Add tests.

Deliverable:

```bash
bandcamp-dl --dry-run URL
```

can identify the page type.

---

## Phase 3 — Bandcamp Parser

Implement:

* Release parser
* Track parser
* Artist parser
* Metadata normalization
* Artwork extraction
* Lyrics extraction

Add fixture-based tests.

Do not implement downloading yet.

Deliverable:

```text
URL
 ↓
Release model
```

---

## Phase 4 — Downloader

Implement:

* HTTP client
* Worker pool
* Streaming downloads
* Retry system
* Context cancellation
* Progress events
* Temporary files
* Atomic finalization

Deliverable:

```bash
bandcamp-dl URL
```

downloads audio successfully.

---

## Phase 5 — Metadata

Implement:

* ID3 tags
* Lyrics
* Artwork
* Cover image
* Metadata validation

Deliverable:

Downloaded MP3 files contain complete metadata.

---

## Phase 6 — Storage

Implement:

* Directory templates
* Filename templates
* Filename sanitization
* Existing-file handling
* Overwrite/skip behavior

Deliverable:

Clean music-library output.

---

## Phase 7 — Playlists

Implement:

* M3U
* PLS
* WPL
* ZPL

Deliverable:

```bash
bandcamp-dl --playlist m3u URL
```

creates a valid playlist.

---

## Phase 8 — CLI Polish

Add:

* Multiple URLs
* URL files
* Dry run
* Quiet mode
* Verbose mode
* Better progress UI
* Configuration overrides
* Useful error messages

---

## Phase 9 — Testing

Complete:

```bash
go test ./...
go test -race ./...
go vet ./...
```

Add integration tests and parser fixtures.

---

## Phase 10 — Release

Build for:

```text
Linux amd64
Linux arm64
macOS amd64
macOS arm64
Windows amd64
Windows arm64
```

Produce:

```text
.tar.gz
.zip
checksums
```

Document installation and usage.

---

# 30. Definition of Done

The project is considered complete when:

* [x] Artist URLs work.
* [x] Album URLs work.
* [x] Track URLs work.
* [x] Album metadata is extracted correctly.
* [x] Track metadata is extracted correctly.
* [x] Audio downloads reliably.
* [x] Multiple tracks download concurrently.
* [x] Failed downloads retry.
* [x] Downloads can be cancelled.
* [x] Existing files are handled safely.
* [x] MP3 ID3 metadata is embedded.
* [x] Lyrics are embedded when available.
* [x] Artwork is embedded.
* [x] Artwork is saved to the album directory.
* [x] Directory templates work.
* [x] Filename templates work.
* [x] Unicode filenames work.
* [x] M3U works.
* [x] PLS works.
* [x] WPL works.
* [x] ZPL works.
* [x] Configuration works.
* [x] CLI overrides configuration.
* [x] Dry-run works.
* [x] Multiple URLs work.
* [x] URL files work.
* [x] Unit tests pass.
* [x] Integration tests pass.
* [x] Race detector passes.
* [x] `go vet` passes.
* [x] Cross-platform builds succeed.
* [x] README contains complete usage documentation.

---

# 31. Implementation Principle

Do not attempt to implement every feature simultaneously.

Build vertically:

```text
URL
 ↓
Parse
 ↓
Download ONE track
 ↓
Tag ONE track
 ↓
Embed artwork
 ↓
Save ONE correct file
```

Once the complete single-track pipeline works, expand it to:

```text
single track
    ↓
album
    ↓
artist
    ↓
multiple URLs
    ↓
playlists
```

This prevents the project from becoming a collection of partially implemented subsystems.

The final architecture should make this possible:

```text
                 ┌──────────────┐
                 │     CLI      │
                 └──────┬───────┘
                        │
                 ┌──────▼───────┐
                 │    Service   │
                 └──────┬───────┘
                        │
        ┌───────────────┼────────────────┐
        │               │                │
 ┌──────▼──────┐ ┌──────▼──────┐ ┌──────▼──────┐
 │  Bandcamp   │ │  Downloader  │ │   Storage   │
 │   Parser    │ │    Engine    │ │             │
 └─────────────┘ └──────┬───────┘ └─────────────┘
                        │
                 ┌──────▼───────┐
                 │   Metadata   │
                 │ ID3/Artwork  │
                 └──────────────┘
                        │
                 ┌──────▼───────┐
                 │   Playlist   │
                 └──────────────┘
```

**Start with Phase 1 and proceed sequentially. Do not skip tests. Do not introduce GUI functionality until the CLI/download engine is stable.**
