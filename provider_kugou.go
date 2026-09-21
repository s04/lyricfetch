package lyricfetch

import (
	"context"
	"net/url"
)

type kugouSearch struct {
	Status     *int `json:"status"`
	Candidates []struct {
		ID       string  `json:"id"`
		Key      string  `json:"accesskey"`
		Song     string  `json:"song"`
		Singer   string  `json:"singer"`
		Duration float64 `json:"duration"`
	} `json:"candidates"`
}

type kugouDownload struct {
	Status  *int   `json:"status"`
	Content string `json:"content"`
}

func (c *Client) kugou(ctx context.Context, t Track) (*Lyrics, error) {
	data, _, err := getJSON[kugouSearch](c, ctx, "https://lyrics.kugou.com/search", url.Values{"ver": {"1"}, "man": {"yes"}, "client": {"pc"}, "keyword": {t.Artist + " - " + t.Title}}, false)
	if err != nil {
		return nil, err
	}
	if err = apiStatus(data.Status, 200); err != nil {
		return nil, err
	}
	for _, candidate := range data.Candidates {
		if !matches(t, candidate.Song, candidate.Singer, candidate.Duration/1000) {
			continue
		}
		if candidate.ID == "" || candidate.Key == "" {
			return nil, failure(InvalidResponse, "missing candidate ID or key")
		}
		data, _, err := getJSON[kugouDownload](c, ctx, "https://lyrics.kugou.com/download", url.Values{"ver": {"1"}, "client": {"pc"}, "id": {candidate.ID}, "accesskey": {candidate.Key}, "fmt": {"lrc"}, "charset": {"utf8"}}, false)
		if err != nil {
			return nil, err
		}
		if err = apiStatus(data.Status, 200); err != nil {
			return nil, err
		}
		text, err := decodeLyrics(data.Content)
		if err != nil {
			return nil, err
		}
		return makeLyrics(text, Kugou, candidate.ID, true)
	}
	return nil, nil
}
