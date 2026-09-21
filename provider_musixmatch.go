package lyricfetch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type mxmMessage struct {
	Message struct {
		Header struct {
			Status *int `json:"status_code"`
		} `json:"header"`
		Body json.RawMessage `json:"body"`
	} `json:"message"`
}

func (c *Client) mxmCall(ctx context.Context, action string, params url.Values, token string, target any) error {
	params.Set("app_id", "mac-ios-v2.0")
	if token != "" {
		params.Set("usertoken", token)
	}
	headers := map[string]string{"X-Cookie": "x-mxm-token-guid=", "x-mxm-app-version": "10.1.1", "X-User-Agent": "Musixmatch/2025120901 CFNetwork/3860.300.31 Darwin/25.2.0", "Accept-Language": "en-US,en;q=0.9"}
	data, _, err := c.request(ctx, http.MethodGet, "https://apic-appmobile.musixmatch.com/ws/1.1/"+action, params, nil, headers)
	if err != nil {
		return err
	}
	var response mxmMessage
	if err = decodeJSON(data, &response); err != nil {
		return err
	}
	if err = apiStatus(response.Message.Header.Status, 200); err != nil {
		return err
	}
	return decodeJSON(response.Message.Body, target)
}

func (c *Client) mxmToken(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	token, valid := c.token, time.Now().Before(c.tokenExpiry)
	c.tokenMu.Unlock()
	if token != "" && valid {
		return token, nil
	}
	var body struct {
		Token string `json:"user_token"`
	}
	if err := c.mxmCall(ctx, "token.get", url.Values{"user_language": {"en"}}, "", &body); err != nil {
		return "", err
	}
	if body.Token == "" {
		return "", failure(InvalidResponse, "missing temporary token")
	}
	c.tokenMu.Lock()
	c.token = body.Token
	c.tokenExpiry = time.Now().Add(10 * time.Minute)
	c.tokenMu.Unlock()
	return body.Token, nil
}

func (c *Client) musixmatch(ctx context.Context, t Track) (*Lyrics, error) {
	token, err := c.mxmToken(ctx)
	if err != nil {
		return nil, err
	}
	var body struct {
		Tracks []struct {
			Track struct {
				ID       int64   `json:"track_id"`
				Title    string  `json:"track_name"`
				Artist   string  `json:"artist_name"`
				Duration float64 `json:"track_length"`
			} `json:"track"`
		} `json:"track_list"`
	}
	err = c.mxmCall(ctx, "track.search", url.Values{"q_track": {t.Title}, "q_artist": {t.Artist}, "page_size": {"10"}, "page": {"1"}, "f_has_subtitle": {"1"}}, token, &body)
	if err != nil {
		c.tokenMu.Lock()
		if c.token == token {
			c.token = ""
		}
		c.tokenMu.Unlock()
		return nil, err
	}
	for _, wrapper := range body.Tracks {
		song := wrapper.Track
		if !matches(t, song.Title, song.Artist, song.Duration) {
			continue
		}
		id := strconv.FormatInt(song.ID, 10)
		var lyric struct {
			Subtitle struct {
				Text string `json:"subtitle_body"`
			} `json:"subtitle"`
		}
		if err = c.mxmCall(ctx, "track.subtitle.get", url.Values{"track_id": {id}, "subtitle_format": {"lrc"}}, token, &lyric); err != nil {
			return nil, err
		}
		return makeLyrics(lyric.Subtitle.Text, Musixmatch, id, true)
	}
	return nil, nil
}
