# lyricfetch

A small **Go library and CLI** for synchronized and plain lyrics, with a **Python wrapper**. Go 1.27.1; standard-library runtime only in both languages. No server, account credentials, persisted lyrics, or embedded browser.

```sh
go build -o bin/lyricfetch ./cmd/lyricfetch
./bin/lyricfetch --title 'Imagine' --artist 'John Lennon'
```

The CLI prints source, timing type, line count and provider attempts. Add `--include-text` to include lyrics. Exit codes: `0` found, `1` no accepted result, `2` invalid input/process failure. Use `--stdin` to send a Track JSON object without putting song metadata in process arguments.

## Go

```go
client, err := lyricfetch.New(lyricfetch.Options{})
if err != nil { return err }
defer client.Close()
result, err := client.Search(ctx, lyricfetch.Track{
    Title: "Imagine", Artist: "John Lennon",
})
```

Import `github.com/s04/lyricfetch`. `Result.Lyrics` preserves text, source, source ID and whether timestamps exist. `Result.Attempts` distinguishes `found`, `not_found`, `instrumental`, `unavailable`, `timeout`, `canceled`, and `invalid_response`. Only invalid input returns a Search error; inspect attempts for operational failures. Clients are safe for concurrent searches.

## Python

```sh
python -m pip install ./python
```

```python
from lyricfetch import search
result = search("Imagine", "John Lennon", executable="./bin/lyricfetch")
if result.lyrics:
    print(result.lyrics.source, result.lyrics.synced)
```

The wrapper requires Python 3.11+ and a separately installed/built Go executable. It uses JSON stdin/stdout, never a shell, with a process deadline. It does not duplicate provider implementations. [Python API](python/README.md).

## Sources and observed availability

Default order: LRCLIB → NetEase → Kugou → QQ Music → lyrics.ovh. Plain results are held while later synchronized sources are checked. Sources run sequentially within a shared 25-second context deadline and 8-second request timeout. Caller cancellation applies to the complete operation; bodies are capped at 2 MB, redirects are rejected, and errors omit URLs/tokens/bodies.

| Source | Type | Small live checks, September 21–22, 2026 |
| --- | --- | --- |
| LRCLIB | Synced/plain | 3/3 found; one exact-lookup result was plain |
| NetEase | Synced | 3/3 after switching to public POST cloudsearch; no cookies |
| Kugou | Synced | 3/3 |
| QQ Music | Synced | 3/3 using current CGI and `crypt=0` |
| lyrics.ovh | Plain | 3/3; underlying scraper is not identified by its response |
| Musixmatch | Synced, experimental opt-in | Mobile endpoint works in repaired syncedlyrics probes; new core observed intermittent 401 and a strict-match miss |

Use `Options.Providers` or `--providers kugou,qqmusic` to choose/order providers. Musixmatch is excluded from defaults. It obtains at most one temporary token per attempt and keeps it only in memory; no recursive retries or shipped cookies.

Matching requires title/artist equality after case/whitespace normalization. **Version labels are significant.** If a duration is supplied, candidates must provide a duration within three seconds. QQ smartbox and lyrics.ovh cannot verify duration, so those adapters yield no match when duration is requested. Unicode accents are preserved; canonical Unicode equivalence and fuzzy matching are intentionally not claimed. These conservative choices can miss legitimate catalog variants.

This sample matrix does not establish catalog-wide coverage or exact synchronization with every recording. A `synced` result means valid LRC timestamps are present, not that their timing has been manually verified.

## Development

```sh
go test -race -cover ./...
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
go test -run '^$' -fuzz '^FuzzLyrics$' -fuzztime=5s .
go test -run '^$' -fuzz '^FuzzResponse$' -fuzztime=5s .
python -m pip install pytest==9.1.1 ruff==0.16.8
PYTHONPATH=python python -m pytest -q python/tests
ruff check python tools
ruff format --check python tools
```

Offline tests use synthetic original lyrics and a transport that rejects unexpected requests. CI never calls live lyric services. Optional `python tools/probe.py` makes a small live matrix and saves only metadata/status/counts. See [source protocols](docs/sources.md), [prior art and inspiration](docs/prior-art.md), and [verification](docs/verification.md).

MIT applies to this code, not to provider catalogs or retrieved lyrics. Protocol implementations are independently written; no OneTagger, musicdl or lyrics.ovh implementation is bundled.
