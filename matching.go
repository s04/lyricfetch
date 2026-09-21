package lyricfetch

import (
	"encoding/base64"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

var timestampPattern = regexp.MustCompile(`(?m)^\s*\[\d+:[0-5]\d(?:\.\d+)?\]`)

func normalize(value string) string { return strings.ToLower(strings.Join(strings.Fields(value), " ")) }
func matches(track Track, title, artist string, duration float64) bool {
	if normalize(track.Title) != normalize(title) || normalize(track.Artist) != normalize(artist) {
		return false
	}
	return track.Duration == 0 || (duration > 0 && !math.IsNaN(duration) && !math.IsInf(duration, 0) && math.Abs(duration-track.Duration) <= 3)
}

func makeLyrics(text string, source Source, id string, requireSynced bool) (*Lyrics, error) {
	text = strings.TrimSpace(strings.TrimPrefix(text, "\ufeff"))
	if text == "" {
		return nil, nil
	}
	if !utf8.ValidString(text) {
		return nil, failure(InvalidResponse, "invalid UTF-8 lyrics")
	}
	synced := timestampPattern.MatchString(text)
	if requireSynced && !synced {
		return nil, failure(InvalidResponse, "expected synchronized lyrics")
	}
	return &Lyrics{Text: text, Synced: synced, Source: source, SourceID: id}, nil
}

func decodeLyrics(encoded string) (string, error) {
	data, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || !utf8.Valid(data) {
		return "", failure(InvalidResponse, "invalid lyric encoding")
	}
	return string(data), nil
}
