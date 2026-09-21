# Prior art and protocol references

Research checked September 21–22, 2026. lyricfetch is a small Go library and CLI with a Python standard-library wrapper. Its implementation targets Go 1.27.1 and uses the Go standard library. It does not introduce a new lyrics catalog or claim to be the first multi-provider lyrics library.

The projects below informed provider selection, protocol investigation and failure handling. They are references, not vendored dependencies. lyricfetch's Go transport, matching, provider adapters and Python wrapper were written for this project; whole upstream modules were not imported. Protocol details and their sources are identified explicitly below. This document does not substitute for reviewing attribution when incorporating third-party code later.

## Libraries, clients and services

| Project | Relevant capability | What lyricfetch learns or implements |
| --- | --- | --- |
| [moehmeni/syncedlyrics](https://github.com/moehmeni/syncedlyrics) — Python, MIT | Multi-provider synchronized/plain lyrics retrieval. Latest release observed: 1.0.1, July 28, 2024. | Provider coverage and result parsing informed research. lyricfetch uses its own typed result/failure model, bounded requests and structured matching. An adapter existing upstream is not evidence its remote service still works. |
| [yanus/syncedlyrics](https://github.com/moehmeni/syncedlyrics/compare/main...yanus:main) — Python fork | Three September 18, 2026 commits: synced-result filtering, mobile Musixmatch endpoint, richer search parameters. | Reference for Musixmatch mobile endpoint behavior and structured searches. Recent commits alone do not prove catalog-wide reliability. |
| [OneTagger](https://github.com/Marekkon5/onetagger) — Rust, GPL-3.0 | Music tagging application with a Musixmatch adapter and rich synchronization handling. | Establishes the specific Rust lineage described below. lyricfetch does not embed OneTagger or copy its Rust modules. |
| [NTag/lyrics.ovh](https://github.com/NTag/lyrics.ovh) — JavaScript, MIT | Hosted plain-text API backed by six website scrapers. | lyricfetch calls the documented hosted API as a plain fallback. It does not bundle the Express server or port its six scraper implementations. |
| [LRCLIB](https://github.com/tranxuanthang/lrclib) — Rust server, MIT | Lyrics catalog with metadata lookup, search, synchronized/plain lyrics and instrumental status. | Direct HTTP adapter; no server runtime dependency. [API documentation](https://lrclib.net/docs). |
| [LRCGET](https://github.com/tranxuanthang/lrcget) — Tauri/Rust desktop app, MIT | Official LRCLIB client for local-library lyrics retrieval. | Useful reference for the ecosystem and user workflows. It accesses the same service, not another independent catalog. |
| [L-1124/QQMusicApi](https://github.com/L-1124/QQMusicApi) — Python | Maintained QQ Music request definitions and transport. | Reference for public smartbox search and `GetPlayLyricInfo`. Our narrow adapter uses the verified credential-free request path, without importing its account/session stack. |
| [CharlesPikachu/musicdl](https://github.com/CharlesPikachu/musicdl) — Python | Multi-service music client with NetEase and Kugou protocol implementations. | Reference for NetEase POST cloud search and Kugou candidate/download shapes. No audio-download machinery, copied account cookies or third-party proxy services are needed by our lyrics adapters. |
| [dddevid/syncedlyrics](https://github.com/moehmeni/syncedlyrics/compare/main...dddevid:main) — TypeScript fork | October 2025 port with test and package-release changes. | Relevant cross-language prior art. A language port alone does not repair blocked providers; its tests were not independently run during this research. |

Upstream software licenses concern code, not blanket rights to all lyrics those services return. Availability and published behavior can change; lyricfetch must report unavailable providers honestly.

## The precise Rust lineage

syncedlyrics' **Musixmatch provider**, specifically, credits OneTagger's Rust implementation and describes a Rust-to-Python conversion. Its comment links OneTagger commit `0654131188c4df2b4b171ded7cdb927a4369746e`. This does not establish that the entire syncedlyrics package was derived from Rust, nor that LRCLIB or LRCGET was its origin. [Attributed Python source](https://github.com/moehmeni/syncedlyrics/blob/3c8a318d9a8df26855bdc3a5d23f7fd2b99ade2e/syncedlyrics/providers/musixmatch.py), [linked Rust source](https://github.com/Marekkon5/onetagger/blob/0654131188c4df2b4b171ded7cdb927a4369746e/crates/onetagger-platforms/src/musixmatch.rs).

OneTagger declares GPL-3.0, while syncedlyrics declares MIT. That is a concrete provenance detail to consider before redistributing copied provider code. We have not established whether separate permissions exist and make no infringement claim. The newly written adapter follows observed HTTP request/response behavior; a future direct code port would require its own license review. [OneTagger license](https://github.com/Marekkon5/onetagger/blob/master/LICENSE), [syncedlyrics license](https://github.com/moehmeni/syncedlyrics/blob/main/LICENSE).

An important reliability lesson survives across languages: both inspected Musixmatch implementations recursively retry token acquisition after certain authentication failures. A network timeout alone does not bound a recursive retry loop. A small library should instead propagate a typed failure within its caller's deadline.

## What lyrics.ovh actually aggregates

We inspected exact commit **`44cf73f2576ab0c6857076de75d6bb4a13175e34`**. Its six plain-text sources are **Genius, AZLyrics, Paroles.net, LyricsMania, Letras.mus.br and Lyrics.com**. Deezer powers `/suggest`, not the returned lyrics. The core races all sources, returns the first successful text, and caches it; source identity is not returned by the public `/v1/{artist}/{title}` API. [Pinned source](https://github.com/NTag/lyrics.ovh/blob/44cf73f2576ab0c6857076de75d6bb4a13175e34/lyrics.js), [HTTP wrapper](https://github.com/NTag/lyrics.ovh/blob/44cf73f2576ab0c6857076de75d6bb4a13175e34/index.js).

Consequently, lyricfetch labels an aggregate result `lyricsovh`; it cannot truthfully label the underlying website. Source inspection also showed why a direct port is not automatically an improvement: unrestricted first-success races can favor weak matches, original and translated paragraphs can be mixed, and blanket redirect rejection can discard valid canonical song pages. Fresh small probes found a usable LyricsMania container and a valid Letras song reached through a same-host canonical redirect. These are possible future direct adapters, not a claim they are already implemented here.

NTag's license is MIT with copyright Basile Bruneau. Calling its service is different from copying its scraper code. If substantial scraper logic is ported later, retain the required notice and record the source commit. [Pinned license](https://github.com/NTag/lyrics.ovh/blob/44cf73f2576ab0c6857076de75d6bb4a13175e34/LICENSE).

## Verified protocol details behind the new adapters

### NetEase: POST cloud search without historical cookies

The old `GET /api/search/pc` path returned application error `-462` in our fresh client. The reference implementation instead uses `POST https://music.163.com/api/cloudsearch/pc` with form fields `s`, `type=1`, `limit`, and `offset`. Search metadata uses `ar[].name`, `dt` in milliseconds and `al.name`. Lyrics remain available from `GET https://music.163.com/api/song/lyric?id={id}&lv=1`. [Reference search implementation](https://github.com/CharlesPikachu/musicdl/blob/master/musicdl/modules/sources/netease.py).

Small public tests with user agent `lyricfetch/0.1`, no supplied cookies and no login returned successful search/lyric responses for three songs:

| Track | Matching ID | Duration (ms) | Retrieved lines |
| --- | --- | --- | --- |
| Imagine — John Lennon | 1476431 | 185173 | 28 |
| Hello — Adele | 35847388 | 295502 | 53 |
| Alors on danse — Stromae | 19086497 | 206066 | 40 |

Every search and lyric response in that matrix was HTTP 200/application code 200. A separate homepage visit established zero cookies, and a fresh request without that visit also succeeded. There was no need to copy syncedlyrics' large historical cookie header. These observations establish three examples, not regional or catalog-wide guarantees. Machine-readable implementation probes are recorded in [netease-cloudsearch-live.json](netease-cloudsearch-live.json).

### QQ Music: current CGI instead of the legacy lyric endpoint

The maintained reference defines smartbox discovery at `https://c.y.qq.com/splcloud/fcgi-bin/smartbox_new.fcg` and CGI module `music.musichallSong.PlayLyricInfo`, method `GetPlayLyricInfo`, sent to `https://u.y.qq.com/cgi-bin/musicu.fcg`. [Search definition](https://github.com/L-1124/QQMusicApi/blob/main/qqmusic_api/modules/search.py), [lyric definition](https://github.com/L-1124/QQMusicApi/blob/main/qqmusic_api/modules/lyric.py), [transport](https://github.com/L-1124/QQMusicApi/blob/main/qqmusic_api/core/executor.py).

For the public song MID `000Eq2fc2uW9hE`, the old lyric endpoint returned application code `-1310`. The current CGI with `crypt=0` returned base64 UTF-8 LRC: 37 total lines, 32 timestamped. No account cookie, session acquisition or signature was needed in the successful experiment. Both the top-level and subrequest status must be checked; HTTP 200 alone is insufficient.

### Kugou: search-issued candidate keys

The public search endpoint returns candidate IDs, title/artist metadata and a candidate-specific `accesskey`; the download endpoint accepts that returned pair and `fmt=lrc`, yielding base64 UTF-8 text. HTTPS search/download successfully returned 32 timestamped lines for Imagine. The adapter uses service-issued candidate keys rather than hard-coding or guessing account credentials. [Reference implementation](https://github.com/CharlesPikachu/musicdl/blob/master/musicdl/modules/sources/kugou.py).

## Scope and future work

The benefit sought here is a small dependency surface, predictable deadlines, clear result provenance and testable matching—not novelty. Deterministic fixtures should cover malformed responses, mismatched recordings, body-level errors inside HTTP 200, encoded content, timeouts and synchronization preference. Keep live checks opt-in and report metadata/counts rather than committing full third-party lyrics.

Unverified adapters and blocked providers should remain distinguishable from demonstrated working paths. More providers are useful only when they improve correctly matched coverage without making ordinary lookups slow or unreliable.
