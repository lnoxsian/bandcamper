# Bandcamper Documentation

Bandcamper is a Go command-line downloader for publicly available Bandcamp artist, album, and track pages. It resolves page metadata, downloads MP3 streams, writes ID3v2 tags, handles artwork and lyrics, and can create playlists.

## Documentation

- [User guide](user-guide.md): installation, supported URLs, common commands, output layout, and troubleshooting.
- [Configuration reference](configuration.md): command-line flags, TOML settings, templates, and precedence rules.
- [Development guide](development.md): repository layout, build recipes, tests, release builds, and implementation notes.
- [Implementation plan](plan.md): the original project plan and intended design direction.

## Quick start

Build the binary from the repository root:

```sh
go build -o bandcamper ./cmd/bandcamper
```

Download a release:

```sh
./bandcamper https://artist.bandcamp.com/album/example-album
```

Preview the resolved release and generated filenames without downloading files:

```sh
./bandcamper --dry-run https://artist.bandcamp.com/album/example-album
```

The default output directory is `~/Music`. The default directory and filename templates are:

```text
{artist}/{album}
{tracknumber} - {title}.mp3
```

## Scope and access

Bandcamper works with the public page and stream information available to it. It does not bypass authentication, DRM, purchase restrictions, or other access controls. Downloads remain subject to Bandcamp's terms and the rights held by the artist or label.

## Credits

Bandcamper was inspired by [Otiel/BandcampDownloader](https://github.com/Otiel/BandcampDownloader). Credit and thanks to [Otiel](https://github.com/Otiel) for the original project and inspiration.
