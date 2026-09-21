package lyricfetch

import (
	"context"
	"net/url"
	"strconv"
)

type lrclibRecord struct {
	ID           int64   `json:"id"`
	Title        string  `json:"trackName"`
	Artist       string  `json:"artistName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	Synced       string  `json:"syncedLyrics"`
	Plain        string  `json:"plainLyrics"`
}

func (c *Client) lrclib(ctx context.Context, t Track) (*Lyrics, error) {
	params := url.Values{"track_name": {t.Title}, "artist_name": {t.Artist}}
	if t.Album != "" {
		params.Set("album_name", t.Album)
	}
	if t.Duration > 0 {
		params.Set("duration", strconv.FormatFloat(t.Duration, 'f', -1, 64))
	}
	item, found, err := getJSON[lrclibRecord](c, ctx, "https://lrclib.net/api/get", params, true)
	if err != nil {
		return nil, err
	}
	records := []lrclibRecord{item}
	if !found {
		params.Del("album_name")
		params.Del("duration")
		records, _, err = getJSON[[]lrclibRecord](c, ctx, "https://lrclib.net/api/search", params, false)
		if err != nil {
			return nil, err
		}
	}
	var plain *Lyrics
	for _, record := range records {
		if !matches(t, record.Title, record.Artist, record.Duration) {
			continue
		}
		if record.Instrumental {
			return nil, failure(Instrumental, "track identified as instrumental")
		}
		text := record.Synced
		if text == "" {
			text = record.Plain
		}
		lyric, err := makeLyrics(text, LRCLIB, strconv.FormatInt(record.ID, 10), record.Synced != "")
		if err != nil {
			return nil, err
		}
		if lyric != nil && lyric.Synced {
			return lyric, nil
		}
		if plain == nil {
			plain = lyric
		}
	}
	return plain, nil
}
