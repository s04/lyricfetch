package lyricfetch

import (
	"context"
	"net/url"
)

func (c *Client) lyricsOVH(ctx context.Context, t Track) (*Lyrics, error) {
	if t.Duration > 0 {
		return nil, nil
	}
	endpoint := "https://api.lyrics.ovh/v1/" + url.PathEscape(t.Artist) + "/" + url.PathEscape(t.Title)
	data, found, err := getJSON[struct {
		Lyrics string `json:"lyrics"`
	}](c, ctx, endpoint, nil, true)
	if err != nil || !found {
		return nil, err
	}
	return makeLyrics(data.Lyrics, LyricsOVH, "", false)
}
