"""Python interface to the Go lyricfetch executable; no Python runtime dependencies."""

import json
import math
import os
import subprocess
from dataclasses import dataclass
from enum import StrEnum


class Status(StrEnum):
    FOUND = "found"
    NOT_FOUND = "not_found"
    INSTRUMENTAL = "instrumental"
    UNAVAILABLE = "unavailable"
    TIMEOUT = "timeout"
    CANCELED = "canceled"
    INVALID_RESPONSE = "invalid_response"


@dataclass(frozen=True)
class Lyrics:
    text: str
    synced: bool
    source: str
    source_id: str = ""


@dataclass(frozen=True)
class Attempt:
    source: str
    status: Status
    detail: str = ""


@dataclass(frozen=True)
class Result:
    lyrics: Lyrics | None
    attempts: tuple[Attempt, ...]


class LyricfetchError(RuntimeError):
    """The local executable could not complete its JSON protocol."""


def search(
    title: str,
    artist: str,
    *,
    album: str = "",
    duration: float = 0,
    providers: tuple[str, ...] | None = None,
    synced_only: bool = False,
    budget: float = 25,
    executable: str | os.PathLike[str] = "lyricfetch",
) -> Result:
    """Run one bounded search. Provider failures are returned in result.attempts.

    Install/build the Go CLI separately. Track metadata goes through stdin,
    never shell interpolation or command-line arguments. Concurrent calls use
    separate processes; each search's temporary tokens die with its process.
    """
    if (
        not isinstance(title, str)
        or not isinstance(artist, str)
        or not title.strip()
        or not artist.strip()
    ):
        raise ValueError("title and artist must be nonempty strings")
    if (
        not isinstance(album, str)
        or max(len(value.encode()) for value in (title, artist, album)) > 2000
    ):
        raise ValueError("metadata fields must not exceed 2000 UTF-8 bytes")
    if not math.isfinite(duration) or duration < 0:
        raise ValueError("duration must be finite nonnegative seconds")
    if not math.isfinite(budget) or budget <= 0:
        raise ValueError("budget must be positive finite seconds")
    command = [
        os.fspath(executable),
        "--stdin",
        "--include-text",
        "--budget",
        f"{budget}s",
    ]
    if providers is not None:
        if not providers or any(
            not isinstance(value, str) or not value or "," in value for value in providers
        ):
            raise ValueError("providers must contain individual provider names")
        command.extend(["--providers", ",".join(providers)])
    if synced_only:
        command.append("--synced-only")
    try:
        process = subprocess.run(
            command,
            input=json.dumps(
                {"title": title, "artist": artist, "album": album, "duration": duration}
            ),
            text=True,
            encoding="utf-8",
            capture_output=True,
            timeout=budget + 2,
            check=False,
        )
    except FileNotFoundError:
        raise LyricfetchError(
            "lyricfetch executable not found; build/install the Go CLI first"
        ) from None
    except subprocess.TimeoutExpired:
        raise LyricfetchError("lyricfetch process exceeded its deadline") from None
    except OSError:
        raise LyricfetchError("could not run lyricfetch executable") from None
    if process.returncode not in (0, 1):
        raise LyricfetchError("lyricfetch rejected the request or failed")
    try:
        data = json.loads(process.stdout)
        raw = data["lyrics"]
        lyrics = Lyrics(**raw) if raw is not None else None
        attempts = tuple(
            Attempt(
                source=item["source"],
                status=Status(item["status"]),
                detail=item.get("detail", ""),
            )
            for item in data["attempts"]
        )
        if lyrics is not None and (
            not isinstance(lyrics.text, str)
            or not isinstance(lyrics.synced, bool)
            or not isinstance(lyrics.source, str)
        ):
            raise ValueError("invalid lyric types")
        if (process.returncode == 0) != (lyrics is not None):
            raise ValueError("exit status disagrees with payload")
        return Result(lyrics=lyrics, attempts=attempts)
    except (ValueError, KeyError, TypeError):
        raise LyricfetchError("invalid JSON response from lyricfetch") from None


__all__ = ["Attempt", "LyricfetchError", "Lyrics", "Result", "Status", "search"]
