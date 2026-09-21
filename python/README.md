# lyricfetch for Python

A dependency-free wrapper around the Go CLI. Requires Python 3.11+ and a separately built `lyricfetch` executable on PATH (or `executable=`).

```sh
# From the repository root:
go build -o bin/lyricfetch ./cmd/lyricfetch
python -m pip install ./python
```

```python
from lyricfetch import search

result = search("Imagine", "John Lennon", executable="./bin/lyricfetch")
if result.lyrics:
    print(result.lyrics.source, result.lyrics.synced)
for attempt in result.attempts:
    print(attempt.source, attempt.status)
```

Provider failures appear in `attempts`. Invalid input raises `ValueError`; missing binaries, process timeouts and malformed process output raise `LyricfetchError`. Each call has a process deadline in addition to Go's HTTP/context deadlines. This package does not bundle or download executables, run a server, or implement a second set of provider APIs.
