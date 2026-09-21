package lyricfetch

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type qqSearch struct {
	Code *int `json:"code"`
	Data struct {
		Song struct {
			Items []struct {
				ID     string `json:"id"`
				MID    string `json:"mid"`
				Name   string `json:"name"`
				Singer string `json:"singer"`
			} `json:"itemlist"`
		} `json:"song"`
	} `json:"data"`
}

type qqLyrics struct {
	Code     *int `json:"code"`
	Response struct {
		Code *int `json:"code"`
		Data struct {
			SongID int64  `json:"songID"`
			Lyric  string `json:"lyric"`
		} `json:"data"`
	} `json:"req_0"`
}

func (c *Client) qqmusic(ctx context.Context, t Track) (*Lyrics, error) {
	// Public smartbox omits recording duration. Do not silently weaken requested matching.
	if t.Duration > 0 {
		return nil, nil
	}
	data, _, err := getJSON[qqSearch](c, ctx, "https://c.y.qq.com/splcloud/fcgi-bin/smartbox_new.fcg", url.Values{"key": {t.Title + " " + t.Artist}, "format": {"json"}}, false)
	if err != nil {
		return nil, err
	}
	if err = apiStatus(data.Code, 0); err != nil {
		return nil, err
	}
	for _, song := range data.Data.Song.Items {
		if !matches(t, song.Name, song.Singer, 0) {
			continue
		}
		if song.MID == "" {
			return nil, failure(InvalidResponse, "missing song MID")
		}
		body := map[string]any{"comm": map[string]string{"format": "json"}, "req_0": map[string]any{"module": "music.musichallSong.PlayLyricInfo", "method": "GetPlayLyricInfo", "param": map[string]any{"songMid": song.MID, "crypt": 0, "lrc_t": 0, "qrc": 0, "qrc_t": 0, "roma": 0, "roma_t": 0, "trans": 0, "trans_t": 0, "needSingingAnnotations": false, "type": 1}}}
		raw, _, err := c.request(ctx, http.MethodPost, "https://u.y.qq.com/cgi-bin/musicu.fcg", nil, body, nil)
		if err != nil {
			return nil, err
		}
		var result qqLyrics
		if err = decodeJSON(raw, &result); err != nil {
			return nil, err
		}
		if err = apiStatus(result.Code, 0); err != nil {
			return nil, err
		}
		if err = apiStatus(result.Response.Code, 0); err != nil {
			return nil, err
		}
		lyric := result.Response.Data
		if song.ID != "" && lyric.SongID != 0 && song.ID != strconv.FormatInt(lyric.SongID, 10) {
			return nil, failure(InvalidResponse, "returned song ID does not match")
		}
		text, err := decodeLyrics(lyric.Lyric)
		if err != nil {
			return nil, err
		}
		return makeLyrics(text, QQMusic, song.MID, true)
	}
	return nil, nil
}
