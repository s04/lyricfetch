package lyricfetch

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestLRCLIBSearchPrefersSameRecording(t *testing.T) {
	c := testClient(t, LRCLIB)
	calls := 0
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return reply(404, "missing"), nil
		}
		if r.URL.Path != "/api/search" {
			t.Error(r.URL.Path)
		}
		return reply(200, `[{"id":1,"trackName":"Song","artistName":"Artist","duration":180,"plainLyrics":"plain"},{"id":2,"trackName":"Song (Live)","artistName":"Artist","duration":180,"syncedLyrics":"[00:01]wrong"},{"id":3,"trackName":"Song","artistName":"Artist","duration":181,"syncedLyrics":"[00:01]correct"}]`), nil
	})
	result, _ := c.Search(context.Background(), Track{Title: "Song", Artist: "Artist", Duration: 180})
	if result.Lyrics == nil || result.Lyrics.SourceID != "3" || calls != 2 {
		t.Fatal(result, calls)
	}
}

func TestWrongQQSongID(t *testing.T) {
	c := testClient(t, QQMusic)
	calls := 0
	c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return reply(200, `{"code":0,"data":{"song":{"itemlist":[{"id":"42","mid":"x","name":"Song","singer":"Artist"}]}}}`), nil
		}
		return reply(200, `{"code":0,"req_0":{"code":0,"data":{"songID":43,"lyric":""}}}`), nil
	})
	result, _ := c.Search(context.Background(), sample)
	if result.Attempts[0].Status != InvalidResponse {
		t.Fatal(result)
	}
}

func TestMissingAndApplicationFailures(t *testing.T) {
	for _, tc := range []struct {
		source Source
		body   string
		want   Status
	}{
		{Kugou, `{"status":500}`, Unavailable},
		{QQMusic, `{"code":-1310}`, Unavailable},
		{NetEase, `{"code":403}`, Unavailable},
		{Kugou, `{}`, InvalidResponse},
		{Kugou, `{"status":200,"candidates":[]}`, NotFound},
		{QQMusic, `{"code":0,"data":{}}`, NotFound},
		{NetEase, `{"code":200,"result":{}}`, NotFound},
	} {
		t.Run(fmt.Sprintf("%s/%s", tc.source, tc.want), func(t *testing.T) {
			c := testClient(t, tc.source)
			c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return reply(200, tc.body), nil })
			result, _ := c.Search(context.Background(), sample)
			if result.Attempts[0].Status != tc.want {
				t.Fatal(result)
			}
		})
	}
}

func TestEncodingAndTimestampClassification(t *testing.T) {
	for _, bad := range []string{"not base64!", base64.StdEncoding.EncodeToString([]byte{255})} {
		if _, err := decodeLyrics(bad); err == nil {
			t.Error("accepted invalid encoding")
		}
	}
	for _, tc := range []struct {
		text   string
		synced bool
	}{{"Short plain lyrics", false}, {"[Chorus]\nPlain lyrics", false}, {"[00:01]First", true}, {"[00:01.25]First", true}, {"[00:99]Invalid", false}, {"\ufeff[00:01]First", true}} {
		result, err := makeLyrics(tc.text, Kugou, "", false)
		if err != nil || result.Synced != tc.synced {
			t.Fatal(tc, result, err)
		}
	}
	if _, err := makeLyrics("plain", Kugou, "", true); err == nil {
		t.Fatal("accepted unsynced response")
	}
}

func TestDurationUnsupportedSourcesDoNotGuess(t *testing.T) {
	c := testClient(t, QQMusic, LyricsOVH)
	result, _ := c.Search(context.Background(), Track{Title: "Song", Artist: "Artist", Duration: 180})
	if result.Lyrics != nil || len(result.Attempts) != 2 {
		t.Fatal(result)
	}
}

func TestLyricsOVHPathAndMiss(t *testing.T) {
	c := testClient(t, LyricsOVH)
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.Contains(r.URL.EscapedPath(), "A%2FB%3F") {
			t.Error(r.URL.EscapedPath())
		}
		return reply(404, "not found"), nil
	})
	result, _ := c.Search(context.Background(), Track{Title: "A/B?", Artist: "Été & moi"})
	if result.Attempts[0].Status != NotFound {
		t.Fatal(result)
	}
}

func TestMusixmatchConcurrentTokenCache(t *testing.T) {
	c := testClient(t, Musixmatch)
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/ws/1.1/token.get" {
			return reply(200, `{"message":{"header":{"status_code":200},"body":{"user_token":"synthetic"}}}`), nil
		}
		if r.URL.Query().Get("usertoken") != "synthetic" {
			t.Error("missing token")
		}
		return reply(200, `{"message":{"header":{"status_code":200},"body":{"track_list":[]}}}`), nil
	})
	var group sync.WaitGroup
	for range 20 {
		group.Go(func() {
			result, err := c.Search(context.Background(), sample)
			if err != nil || result.Attempts[0].Status != NotFound {
				t.Error("unexpected outcome")
			}
		})
	}
	group.Wait()
}
