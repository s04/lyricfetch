"""Opt-in, small live matrix. Records only status/counts, never lyrics or tokens."""

import argparse
import json
import subprocess
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("--binary", default="./bin/lyricfetch")
parser.add_argument("--output", default="docs/live-results.json")
parser.add_argument(
    "--providers",
    nargs="+",
    default=["lrclib", "netease", "kugou", "qqmusic", "lyricsovh", "musixmatch"],
)
args = parser.parse_args()
results = []
for provider in args.providers:
    for title, artist in [
        ("Imagine", "John Lennon"),
        ("Hello", "Adele"),
        ("Alors on danse", "Stromae"),
    ]:
        process = subprocess.run(
            [args.binary, "--stdin", "--providers", provider, "--budget", "10s"],
            input=json.dumps({"title": title, "artist": artist}),
            capture_output=True,
            text=True,
            timeout=12,
            check=False,
        )
        result = {
            "title": title,
            "artist": artist,
            "provider": provider,
            "exit_code": process.returncode,
            **json.loads(process.stdout),
        }
        results.append(result)
        print(json.dumps(result), flush=True)
Path(args.output).write_text(json.dumps(results, indent=2) + "\n", encoding="utf-8")
