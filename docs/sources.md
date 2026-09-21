# Source protocols and evidence

Research: September 21–22, 2026. This records public protocol observations, not supported API contracts unless the service documents them. We inspected source implementations and made bounded normal requests; no account cookies, challenge bypasses or encrypted payload decoders are required by the default adapters.

| Adapter | Public flow | Primary references |
| --- | --- | --- |
| LRCLIB | GET `/api/get` with title/artist/optional album/duration; on 404 GET `/api/search`. Inspect identity, duration, instrumental and embedded lyric fields. | [Official API](https://lrclib.net/docs) |
| NetEase | POST `https://music.163.com/api/cloudsearch/pc`, form `s`, `type=1`, `limit=10`, `offset=0`; candidates `id`, `name`, `ar[].name`, `dt` milliseconds; GET `/api/song/lyric?id=...&lv=1`. | [Maintained musicdl protocol reference](https://github.com/CharlesPikachu/musicdl/blob/master/musicdl/modules/sources/netease.py) |
| Kugou | GET `https://lyrics.kugou.com/search?ver=1&man=yes&client=pc&keyword={artist - title}`; pass selected candidate's returned id/accesskey to `/download?ver=1&client=pc&fmt=lrc&charset=utf8`; base64 UTF-8 content. Check application `status=200`. | [Protocol reference](https://github.com/CharlesPikachu/musicdl/blob/master/musicdl/modules/sources/kugou.py) |
| QQ Music | GET public `smartbox_new.fcg?key=...&format=json`; POST current `musicu.fcg` with module `music.musichallSong.PlayLyricInfo`, method `GetPlayLyricInfo`, selected `songMid`, `crypt=0`, and LRC flags. Both application codes must be 0; decode base64 UTF-8 and compare returned song ID. | [Search reference](https://github.com/L-1124/QQMusicApi/blob/main/qqmusic_api/modules/search.py), [lyric parameters](https://github.com/L-1124/QQMusicApi/blob/main/qqmusic_api/modules/lyric.py), [transport](https://github.com/L-1124/QQMusicApi/blob/main/qqmusic_api/core/executor.py) |
| lyrics.ovh | GET `https://api.lyrics.ovh/v1/{artist}/{title}`; each component independently URL encoded. 404 is a miss, other failures are not. | [Source](https://github.com/NTag/lyrics.ovh), [API documentation](https://lyricsovh.docs.apiary.io) |
| Musixmatch | Experimental mobile `/ws/1.1/token.get`, `track.search`, `track.subtitle.get`; app ID `mac-ios-v2.0`; structured title/artist parameters; one token request, memory-only reuse. | [Recent MIT fork](https://github.com/yanus/syncedlyrics), [original adapter and Rust attribution](https://github.com/moehmeni/syncedlyrics/blob/main/syncedlyrics/providers/musixmatch.py) |

## Failures that informed the implementation

* Old NetEase `/api/search/pc` without inherited cookies returned API `-462`. Switching to the public cloudsearch POST route, supported by a maintained reference, returned all three test tracks and synchronized lyrics. No cookie bootstrap is necessary. Search metadata uses **`ar`/`dt`**, not legacy `artists`/`duration`.
* Old QQ lyric CGI returned `-1310`. Its current public module returned LRC with `crypt=0`; no decryption or login is involved.
* Original Musixmatch desktop search returned unrelated tracks in this environment. A mobile endpoint repair in the separate syncedlyrics checkout produced all three samples, but token requests sometimes return 401. This Go implementation remains experimental: its first probe had a strict-match miss and two 401s. No claim that authentication failures are fixed.
* Direct Genius returned 403. Megalobiz returned 500/503 and later refused connections. Neither is advertised as a working adapter here.

## lyrics.ovh is an aggregator, not a synced source

Its six underlying plain-text scrapers are Genius, AZLyrics, Paroles.net, LyricsMania, Letras and Lyrics.com. Deezer supplies suggestions only. See [pinned implementation](https://github.com/NTag/lyrics.ovh/blob/44cf73f2576ab0c6857076de75d6bb4a13175e34/lyrics.js).

Small direct probes found candidate lyric containers on LyricsMania and Letras. Letras legitimately redirects to a same-host numeric canonical path, which upstream's blanket redirect rejection misses. They are **research candidates**, not implemented adapters: a reliable HTML parser, original-vs-translated text handling, metadata verification and same-origin redirect tests deserve a separate change. We do not parse arbitrary HTML with regex to preserve a nominal zero-dependency claim.

## Evidence files

`live-results.json` is the initial Go matrix, including failed legacy NetEase search. `netease-cloudsearch-live.json` records the successful repaired adapter. Results contain status and counts only; lyric bodies and temporary tokens are excluded. Fixtures are synthetic original text.
