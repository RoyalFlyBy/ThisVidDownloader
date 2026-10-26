package utils

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/RoyalFlyBy/go-thisvid/api"
)

// FileSize returns the file's size. A missing path is reported as false. The
// threaded task creates the target as an empty placeholder while it downloads,
// so callers must compare the size rather than treating existence as complete.
func FileSize(location string) (int64, bool) {
	info, err := os.Stat(location)
	if err != nil || info.IsDir() {
		return 0, false
	}
	return info.Size(), true
}

var fileNameReplacer = strings.NewReplacer(
	`\`, ``, `/`, ``, `?`, ``, `*`, ``, `:`, ``, `<`, ``, `>`, ``, `|`, ``, `"`, ``,
)

// SanitizeFileName removes the characters Windows and Unix reject in a file
// name, plus control characters, and trims trailing dots and spaces.
func SanitizeFileName(fileName string) string {
	cleaned := fileNameReplacer.Replace(fileName)
	cleaned = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, cleaned)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return strings.TrimRight(strings.TrimSpace(cleaned), ". ")
}

// SelectStream picks the best stream, or the stream at exactly index when it is
// not negative. An index past the end clamps to the last stream.
func SelectStream(streams []*api.VideoStream, index int) *api.VideoStream {
	if index < 0 {
		return api.HighestQuality(streams)
	}

	valid := make([]*api.VideoStream, 0, len(streams))
	for _, stream := range streams {
		if stream != nil && len(stream.URL) > 0 {
			valid = append(valid, stream)
		}
	}
	if len(valid) == 0 {
		return nil
	}
	if index >= len(valid) {
		index = len(valid) - 1
	}
	return valid[index]
}

// StreamExtension returns the stream's extension, falling back to the resolved
// content type and then to .mp4.
func StreamExtension(stream *api.VideoStream, contentType string) string {
	if stream != nil && len(stream.Extension) > 0 {
		return stream.Extension
	}
	if strings.Contains(strings.ToLower(contentType), "webm") {
		return ".webm"
	}
	return ".mp4"
}

// VideoBaseName builds "<uploader> - <title>", optionally prefixed with the
// video id. It falls back to the slug when the page has no title.
func VideoBaseName(video *api.Video, prefixID bool) string {
	name := ""
	if video.Uploader != nil {
		name = strings.TrimSpace(video.Uploader.Name)
	}
	title := strings.TrimSpace(video.Title)
	switch {
	case len(name) > 0 && len(title) > 0:
		name = name + " - " + title
	case len(title) > 0:
		name = title
	case len(name) > 0:
		name = name + " - " + video.Slug
	default:
		name = video.Slug
	}

	name = SanitizeFileName(name)
	if len(name) == 0 {
		name = "thisvid"
	}
	if prefixID {
		id := video.Slug
		if video.ID > 0 {
			id = strconv.FormatInt(video.ID, 10)
		}
		name = "[" + id + "] " + name
	}
	return name
}

// EnsureDir creates the parent directory of a file location.
func EnsureDir(fileLocation string) error {
	dir := filepath.Dir(fileLocation)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, os.ModePerm)
}
