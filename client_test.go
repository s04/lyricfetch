package lyricfetch

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func testClient(t *testing.T, sources ...Source) *Client {
	t.Helper()
	c, err := New(Options{Providers: sources})
	if err != nil {
		t.Fatal(err)
	}
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected network request: %s", r.URL.Path)
		return nil, errors.New("network prohibited")
	})
	t.Cleanup(c.Close)
	return c
}
func reply(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

var sample = Track{Title: "Song", Artist: "Artist"}

func TestProviders(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("[00:01]Original fixture"))
	cases := []struct {
		name      Source
		responses []string
	}{
		{LRCLIB, []string{`{"id":1,"trackName":"Song","artistName":"Artist","syncedLyrics":"[00:01]Original fixture"}`}},
		{NetEase, []string{`{"code":200,"result":{"songs":[{"id":2,"name":"Song","ar":[{"name":"Artist"}]}]}}`, `{"code":200,"lrc":{"lyric":"[00:01]Original fixture"}}`}},
		{Kugou, []string{`{"status":200,"candidates":[{"id":"3","song":"Song","singer":"Artist","accesskey":"synthetic-only"}]}`, fmt.Sprintf(`{"status":200,"content":%q}`, encoded)}},
		{QQMusic, []string{`{"code":0,"data":{"song":{"itemlist":[{"id":"4","mid":"testmid","name":"Song","singer":"Artist"}]}}}`, fmt.Sprintf(`{"code":0,"req_0":{"code":0,"data":{"songID":4,"lyric":%q}}}`, encoded)}},
		{LyricsOVH, []string{`{"lyrics":"Original fixture"}`}},
		{Musixmatch, []string{`{"message":{"header":{"status_code":200},"body":{"user_token":"synthetic-token"}}}`, `{"message":{"header":{"status_code":200},"body":{"track_list":[{"track":{"track_id":5,"track_name":"Song","artist_name":"Artist"}}]}}}`, `{"message":{"header":{"status_code":200},"body":{"subtitle":{"subtitle_body":"[00:01]Original fixture"}}}}`}},
	}
	for _, tc := range cases {
		t.Run(string(tc.name), func(t *testing.T) {
			c := testClient(t, tc.name)
			calls := 0
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if calls >= len(tc.responses) {
					t.Fatal("unexpected retry")
				}
				if tc.name == Kugou && calls == 1 && r.URL.Query().Get("accesskey") != "synthetic-only" {
					t.Error("must use returned candidate key")
				}
				if tc.name == QQMusic && calls == 1 && r.Method != http.MethodPost {
					t.Error("QQ CGI requires POST")
				}
				response := reply(200, tc.responses[calls])
				calls++
				return response, nil
			})
			result, err := c.Search(context.Background(), sample)
			if err != nil || result.Lyrics == nil || result.Lyrics.Source != tc.name {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if result.Lyrics.Synced != (tc.name != LyricsOVH) {
				t.Error("wrong timing classification")
			}
			if calls != len(tc.responses) {
				t.Fatalf("calls=%d", calls)
			}
		})
	}
}

func TestInputValidation(t *testing.T) {
	for _, track := range []Track{{}, {Title: "Song"}, {Title: "Song", Artist: "Artist", Duration: math.NaN()}, {Title: "Song", Artist: "Artist", Duration: -1}, {Title: strings.Repeat("a", 2001), Artist: "Artist"}} {
		c := testClient(t, LRCLIB)
		if _, err := c.Search(context.Background(), track); err == nil {
			t.Error("accepted invalid track")
		}
	}
	for _, options := range []Options{{Providers: []Source{"bad"}}, {Providers: []Source{LRCLIB, LRCLIB}}, {Budget: -1}, {RequestTimeout: -1}} {
		if _, err := New(options); err == nil {
			t.Error("accepted invalid config")
		}
	}
}

func TestConservativeMatching(t *testing.T) {
	cases := []struct {
		track         Track
		title, artist string
		duration      float64
		want          bool
	}{
		{sample, " song ", "ARTIST", 0, true},
		{sample, "Song (Demo)", "Artist", 0, false},
		{sample, "Song", "Other", 0, false},
		{Track{Title: "Song", Artist: "Artist", Duration: 180}, "Song", "Artist", 182, true},
		{Track{Title: "Song", Artist: "Artist", Duration: 180}, "Song", "Artist", 184, false},
		{Track{Title: "Song", Artist: "Artist", Duration: 180}, "Song", "Artist", 0, false},
	}
	for _, tc := range cases {
		if got := matches(tc.track, tc.title, tc.artist, tc.duration); got != tc.want {
			t.Errorf("%+v: got %v", tc, got)
		}
	}
}

func TestFallbackAndPlainPreference(t *testing.T) {
	c := testClient(t, LyricsOVH, LRCLIB)
	c.adapters[LyricsOVH] = func(context.Context, Track) (*Lyrics, error) { return &Lyrics{Text: "plain", Source: LyricsOVH}, nil }
	c.adapters[LRCLIB] = func(context.Context, Track) (*Lyrics, error) {
		return &Lyrics{Text: "[00:01]sync", Synced: true, Source: LRCLIB}, nil
	}
	result, _ := c.Search(context.Background(), sample)
	if result.Lyrics.Source != LRCLIB || len(result.Attempts) != 2 {
		t.Fatal(result)
	}
	c.adapters[LRCLIB] = func(context.Context, Track) (*Lyrics, error) { return nil, failure(Unavailable, "HTTP 503") }
	result, _ = c.Search(context.Background(), sample)
	if result.Lyrics.Source != LyricsOVH || result.Attempts[1].Status != Unavailable {
		t.Fatal(result)
	}
	c.options.SyncedOnly = true
	result, _ = c.Search(context.Background(), sample)
	if result.Lyrics != nil {
		t.Fatal("plain escaped synced-only mode")
	}
}

func TestInstrumentalStopsFallback(t *testing.T) {
	c := testClient(t, LRCLIB, Kugou)
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return reply(200, `{"trackName":"Song","artistName":"Artist","instrumental":true}`), nil
	})
	result, _ := c.Search(context.Background(), sample)
	if result.Lyrics != nil || len(result.Attempts) != 1 || result.Attempts[0].Status != Instrumental {
		t.Fatal(result)
	}
}

func TestMalformedAndBlockedResponses(t *testing.T) {
	for _, source := range Sources() {
		for _, tc := range []struct {
			status int
			body   string
			want   Status
		}{{403, "blocked", Unavailable}, {429, "rate limited", Unavailable}, {200, "<html>", InvalidResponse}, {200, "null", InvalidResponse}, {200, "[]", InvalidResponse}} {
			t.Run(fmt.Sprintf("%s/%d/%s", source, tc.status, tc.body), func(t *testing.T) {
				c := testClient(t, source)
				c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return reply(tc.status, tc.body), nil })
				result, _ := c.Search(context.Background(), sample)
				if result.Lyrics != nil || result.Attempts[0].Status != tc.want {
					t.Fatal(result)
				}
			})
		}
	}
}

func TestMusixmatchNoRecursiveTokenRetry(t *testing.T) {
	c := testClient(t, Musixmatch)
	calls := 0
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return reply(200, `{"message":{"header":{"status_code":401}}}`), nil
	})
	result, _ := c.Search(context.Background(), sample)
	if calls != 1 || result.Attempts[0].Status != Unavailable {
		t.Fatal(result, calls)
	}
}

func TestCanceledAndBudget(t *testing.T) {
	c := testClient(t, LRCLIB)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, _ := c.Search(ctx, sample)
	if result.Attempts[0].Status != Canceled {
		t.Fatal(result)
	}
	c.options.Budget = 20 * time.Millisecond
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })
	started := time.Now()
	result, _ = c.Search(context.Background(), sample)
	if result.Attempts[0].Status != TimedOut || time.Since(started) > time.Second {
		t.Fatal(result)
	}
}

func TestBodyLimitAndRedaction(t *testing.T) {
	c := testClient(t, LRCLIB)
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return reply(200, strings.Repeat("x", maxResponseBytes+1)), nil
	})
	result, _ := c.Search(context.Background(), sample)
	if result.Attempts[0].Status != InvalidResponse {
		t.Fatal(result)
	}
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("secret-token-must-not-leak") })
	result, _ = c.Search(context.Background(), sample)
	if strings.Contains(fmt.Sprint(result), "secret-token") {
		t.Fatal("secret leaked")
	}
}

func TestRedirectNotFollowed(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; http.Redirect(w, r, "/next", http.StatusFound) }))
	defer server.Close()
	c, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, status, err := c.request(context.Background(), http.MethodGet, server.URL, nil, nil, nil)
	if err == nil || status != 302 || hits != 1 {
		t.Fatalf("status %d hits %d err %v", status, hits, err)
	}
}

func TestClientConcurrent(t *testing.T) {
	c := testClient(t, LRCLIB)
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return reply(200, `{"trackName":"Song","artistName":"Artist","syncedLyrics":"[00:01]Fixture"}`), nil
	})
	var group sync.WaitGroup
	for range 20 {
		group.Go(func() {
			result, err := c.Search(context.Background(), sample)
			if err != nil || result.Lyrics == nil {
				t.Error("concurrent search failed")
			}
		})
	}
	group.Wait()
}

func FuzzLyrics(f *testing.F) {
	for _, seed := range []string{"[00:01]hello", "Plain line", "[99:99]invalid", "\xff", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		result, err := makeLyrics(text, LRCLIB, "", false)
		if err == nil && result != nil && strings.TrimSpace(result.Text) == "" {
			t.Fatal("empty lyrics")
		}
	})
}

func FuzzResponse(f *testing.F) {
	for _, seed := range []string{`{}`, `null`, `[]`, `{"message":{"header":{"status_code":401}}}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		for _, source := range Sources() {
			c := testClient(t, source)
			calls := 0
			c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				if calls > 3 {
					t.Fatal("unbounded requests")
				}
				return reply(200, body), nil
			})
			_, _ = c.Search(context.Background(), sample)
		}
	})
}
