package lyricfetch

import (
	"context"
	"net/url"
	"strconv"
)

type neteaseSearch struct {
	Code   *int `json:"code"`
	Result struct {
		Songs []struct {
			ID       int64   `json:"id"`
			Name     string  `json:"name"`
			Duration float64 `json:"dt"`
			Artists  []struct {
				Name string `json:"name"`
			} `json:"ar"`
		} `json:"songs"`
	} `json:"result"`
}

type neteaseLyrics struct {
	Code *int `json:"code"`
	LRC  struct {
		Lyric string `json:"lyric"`
	} `json:"lrc"`
}

func (c *Client) netease(ctx context.Context, t Track) (*Lyrics, error) {
	encoded, _, err := c.request(ctx, "POST", "https://music.163.com/api/cloudsearch/pc", nil, url.Values{"s": {t.Title + " " + t.Artist}, "type": {"1"}, "limit": {"10"}, "offset": {"0"}}, nil)
	var data neteaseSearch
	if err == nil {
		err = decodeJSON(encoded, &data)
	}
	if err != nil {
		return nil, err
	}
	if err = apiStatus(data.Code, 200); err != nil {
		return nil, err
	}
	for _, song := range data.Result.Songs {
		matched := false
		for _, artist := range song.Artists {
			matched = matched || matches(t, song.Name, artist.Name, song.Duration/1000)
		}
		if !matched {
			continue
		}
		id := strconv.FormatInt(song.ID, 10)
		lyric, _, err := getJSON[neteaseLyrics](c, ctx, "https://music.163.com/api/song/lyric", url.Values{"id": {id}, "lv": {"1"}}, false)
		if err != nil {
			return nil, err
		}
		if err = apiStatus(lyric.Code, 200); err != nil {
			return nil, err
		}
		return makeLyrics(lyric.LRC.Lyric, NetEase, id, false)
	}
	return nil, nil
}
