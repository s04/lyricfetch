package lyricfetch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type adapter func(context.Context, Track) (*Lyrics, error)

// Client is safe for concurrent use. It keeps only a temporary Musixmatch token
// in memory. Close releases idle network connections; no files are written.
type Client struct {
	options     Options
	http        *http.Client
	adapters    map[Source]adapter
	tokenMu     sync.Mutex
	token       string
	tokenExpiry time.Time
}

// New constructs a client. Invalid configuration is rejected before any network request.
func New(options Options) (*Client, error) {
	if options.RequestTimeout < 0 || options.Budget < 0 {
		return nil, errors.New("timeouts must not be negative")
	}
	if options.RequestTimeout == 0 {
		options.RequestTimeout = 8 * time.Second
	}
	if options.Budget == 0 {
		options.Budget = 25 * time.Second
	}
	if len(options.Providers) == 0 {
		options.Providers = []Source{LRCLIB, NetEase, Kugou, QQMusic, LyricsOVH}
	}
	options.Providers = append([]Source(nil), options.Providers...)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	c := &Client{options: options, http: &http.Client{Transport: transport, Timeout: options.RequestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	c.adapters = map[Source]adapter{LRCLIB: c.lrclib, NetEase: c.netease, Kugou: c.kugou, QQMusic: c.qqmusic, LyricsOVH: c.lyricsOVH, Musixmatch: c.musixmatch}
	seen := map[Source]bool{}
	for _, source := range options.Providers {
		if _, ok := c.adapters[source]; !ok {
			c.Close()
			return nil, fmt.Errorf("unknown provider %q", source)
		}
		if seen[source] {
			c.Close()
			return nil, fmt.Errorf("duplicate provider %q", source)
		}
		seen[source] = true
	}
	return c, nil
}

// Close releases idle connections. It does not cancel searches already in progress.
func (c *Client) Close() { c.http.CloseIdleConnections() }

// Search respects the caller's cancellation and the configured overall budget.
// Its error is reserved for invalid input. Operational outcomes are in Attempts.
func (c *Client) Search(ctx context.Context, track Track) (Result, error) {
	result := Result{Attempts: []Attempt{}}
	if err := track.validate(); err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.options.Budget)
	defer cancel()
	var plain *Lyrics
	for _, source := range c.options.Providers {
		if err := ctx.Err(); err != nil {
			result.Attempts = append(result.Attempts, contextAttempt(source, err))
			break
		}
		lyric, err := c.adapters[source](ctx, track)
		attempt := Attempt{Source: source, Status: NotFound}
		if err != nil {
			var pe *providerError
			if errors.As(err, &pe) {
				attempt.Status = pe.status
				attempt.Detail = pe.detail
			} else {
				attempt.Status = Unavailable
				attempt.Detail = "provider failed"
			}
		} else if lyric != nil {
			attempt.Status = Found
		}
		result.Attempts = append(result.Attempts, attempt)
		if attempt.Status == Instrumental {
			return result, nil
		}
		if lyric != nil {
			if lyric.Synced {
				result.Lyrics = lyric
				return result, nil
			}
			if plain == nil {
				plain = lyric
			}
		}
	}
	if !c.options.SyncedOnly {
		result.Lyrics = plain
	}
	return result, nil
}

func contextAttempt(source Source, err error) Attempt {
	status := Canceled
	if errors.Is(err, context.DeadlineExceeded) {
		status = TimedOut
	}
	return Attempt{Source: source, Status: status, Detail: "search deadline or cancellation"}
}
