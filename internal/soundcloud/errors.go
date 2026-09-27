package soundcloud

import "errors"

var (
	// ErrInvalidSoundCloudURL indicates the URL is not a recognized SoundCloud URL.
	ErrInvalidSoundCloudURL = errors.New("invalid SoundCloud URL")
	// ErrSoundCloudParse indicates hydration or page parsing failed.
	ErrSoundCloudParse = errors.New("failed to parse SoundCloud page")
	// ErrSoundCloudMediaUnavailable indicates no accessible media representation was found.
	ErrSoundCloudMediaUnavailable = errors.New("no publicly accessible media representation")
	// ErrSoundCloudUnsupportedFormat indicates the audio format is not supported.
	ErrSoundCloudUnsupportedFormat = errors.New("unsupported SoundCloud media format")
	// ErrSoundCloudNotFound indicates the resource does not exist on SoundCloud.
	ErrSoundCloudNotFound = errors.New("SoundCloud resource not found")
)
