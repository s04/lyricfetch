# Verification

Verified locally September 21–22, 2026 on macOS arm64 with Go **1.27.1** and Python **3.12**. [Official Go releases](https://go.dev/dl/?mode=json) identified 1.27.1 as stable at verification time.

* `go test -race -cover ./...`: passed. Core statement coverage **89.0%**, CLI **70.4%**. Includes all six provider success paths, source fallback, duration/version matching, request limits, cancellation, status/encoding failures, secret redaction, concurrent searches and concurrent temporary-token access.
* `go vet ./...` and Staticcheck **v0.8.1**: passed.
* govulncheck **v1.8.0**: no vulnerabilities found.
* Two five-second fuzz smoke runs passed: approximately **130,000** lyric-parser inputs and **49,000** response-parser inputs. This is bounded smoke evidence, not proof of exhaustive correctness.
* Python wrapper: **15 tests passed**, Ruff **0.16.8** lint/format passed.
* Runtime dependency check: Go core imports only standard-library packages; Python wrapper declares no runtime dependencies. Staticcheck/govulncheck are development tools, pinned in CI.

Live checks are separate from deterministic tests and include no committed lyric bodies or tokens. Initial matrix: [live-results.json](live-results.json). After replacing legacy NetEase search: [netease-cloudsearch-live.json](netease-cloudsearch-live.json). Three songs are a smoke sample, not catalog-wide coverage.

The repaired syncedlyrics fork is a separate experiment, not the Go core's runtime dependency. Its mobile Musixmatch adapter succeeded in a three-song run, but the Go core's stricter matching and intermittent token failures remain visible in its own evidence. Do not substitute one implementation's successes for another's.

GitHub Actions runs race tests, vet, pinned Staticcheck/govulncheck, bounded fuzz tests, and Python checks. No live provider tests or package-publication workflow runs automatically.
