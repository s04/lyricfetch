import json
import subprocess
from unittest.mock import Mock

import pytest
from lyricfetch import LyricfetchError, Status, search


def test_metadata_uses_stdin(monkeypatch):
    payload = {
        "lyrics": {"text": "[00:01]Fixture", "synced": True, "source": "kugou"},
        "attempts": [{"source": "kugou", "status": "found"}],
    }
    run = Mock(return_value=Mock(returncode=0, stdout=json.dumps(payload)))
    monkeypatch.setattr(subprocess, "run", run)
    result = search("$(never-execute)", "Artist", providers=("kugou",), budget=5)
    assert result.lyrics.synced
    assert result.attempts[0].status is Status.FOUND
    assert "$(never-execute)" not in run.call_args.args[0]
    assert json.loads(run.call_args.kwargs["input"])["title"] == "$(never-execute)"
    assert run.call_args.kwargs["timeout"] == 7
    assert "shell" not in run.call_args.kwargs


def test_not_found_is_a_result(monkeypatch):
    monkeypatch.setattr(
        subprocess,
        "run",
        lambda *a, **kw: Mock(
            returncode=1,
            stdout='{"lyrics":null,"attempts":[{"source":"lrclib","status":"not_found"}]}',
        ),
    )
    assert search("Song", "Artist").lyrics is None


@pytest.mark.parametrize(
    "error",
    [FileNotFoundError(), subprocess.TimeoutExpired("binary", 2), PermissionError()],
)
def test_process_errors_sanitized(monkeypatch, error):
    monkeypatch.setattr(subprocess, "run", Mock(side_effect=error))
    with pytest.raises(LyricfetchError):
        search("Song", "Artist")


@pytest.mark.parametrize(
    "code,body",
    [
        (2, "secret"),
        (0, "bad"),
        (0, "null"),
        (0, '{"lyrics":null,"attempts":[]}'),
        (1, '{"lyrics":null,"attempts":[{"source":"a","status":"bad"}]}'),
    ],
)
def test_invalid_protocol(monkeypatch, code, body):
    monkeypatch.setattr(subprocess, "run", Mock(return_value=Mock(returncode=code, stdout=body)))
    with pytest.raises(LyricfetchError):
        search("Song", "Artist")


@pytest.mark.parametrize(
    "kwargs",
    [
        {"budget": 0},
        {"duration": -1},
        {"budget": float("inf")},
        {"providers": ()},
        {"providers": ("lrclib,qqmusic",)},
    ],
)
def test_invalid_options(kwargs):
    with pytest.raises(ValueError):
        search("Song", "Artist", **kwargs)
