// Package lyricfetch retrieves lyrics from public providers with bounded requests,
// conservative track matching, and explicit per-provider outcomes.
package lyricfetch

import (
	"errors"
	"math"
	"strings"
	"time"
)

// Source identifies a provider, not a guarantee of catalog availability.
type Source string

const (
	LRCLIB     Source = "lrclib"
	NetEase    Source = "netease"
	Kugou      Source = "kugou"
	QQMusic    Source = "qqmusic"
	LyricsOVH  Source = "lyricsovh"
	Musixmatch Source = "musixmatch"
)

// Sources returns all implemented providers. Musixmatch is opt-in and experimental.
func Sources() []Source { return []Source{LRCLIB, NetEase, Kugou, QQMusic, LyricsOVH, Musixmatch} }

// Status distinguishes a catalog miss from a service or parsing failure.
type Status string

const (
	Found           Status = "found"
	NotFound        Status = "not_found"
	Instrumental    Status = "instrumental"
	Unavailable     Status = "unavailable"
	TimedOut        Status = "timeout"
	Canceled        Status = "canceled"
	InvalidResponse Status = "invalid_response"
)

// Track contains recording metadata. Duration is in seconds; zero means unknown.
// Matching tolerates capitalization and whitespace, but does not discard version labels.
type Track struct {
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Album    string  `json:"album,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

func (t Track) validate() error {
	if strings.TrimSpace(t.Title) == "" || strings.TrimSpace(t.Artist) == "" {
		return errors.New("title and artist must be nonempty")
	}
	if len(t.Title) > 2000 || len(t.Artist) > 2000 || len(t.Album) > 2000 {
		return errors.New("metadata fields must not exceed 2000 UTF-8 bytes")
	}
	if math.IsNaN(t.Duration) || math.IsInf(t.Duration, 0) || t.Duration < 0 {
		return errors.New("duration must be finite nonnegative seconds")
	}
	return nil
}

// Lyrics preserves the provider and its record ID with the returned text.
type Lyrics struct {
	Text     string `json:"text"`
	Synced   bool   `json:"synced"`
	Source   Source `json:"source"`
	SourceID string `json:"source_id,omitempty"`
}

// Attempt contains a sanitized diagnostic; never a token, request URL, or response body.
type Attempt struct {
	Source Source `json:"source"`
	Status Status `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Result reports fallback attempts even when no lyrics could be retrieved.
type Result struct {
	Lyrics   *Lyrics   `json:"lyrics"`
	Attempts []Attempt `json:"attempts"`
}

// Options controls source order and deadlines. Zero values select safe defaults.
// Providers are tried sequentially. Plain lyrics are retained while synchronized
// sources are tried, and returned only when no synchronized result is found.
type Options struct {
	Providers      []Source
	RequestTimeout time.Duration
	Budget         time.Duration
	SyncedOnly     bool
}

type providerError struct {
	status Status
	detail string
}

func (e *providerError) Error() string           { return e.detail }
func failure(status Status, detail string) error { return &providerError{status, detail} }
