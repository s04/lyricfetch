// Command lyricfetch exposes the library as a bounded JSON process for scripts and Python.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/s04/lyricfetch"
)

func run(args []string, in io.Reader, out, errOut io.Writer) int {
	flags := flag.NewFlagSet("lyricfetch", flag.ContinueOnError)
	flags.SetOutput(errOut)
	title := flags.String("title", "", "Song title")
	artist := flags.String("artist", "", "Artist")
	album := flags.String("album", "", "Album (optional)")
	duration := flags.Float64("duration", 0, "Recording duration in seconds")
	providers := flags.String("providers", "", "Comma-separated providers, in fallback order")
	budget := flags.Duration("budget", 25*time.Second, "Overall request budget")
	syncedOnly := flags.Bool("synced-only", false, "Do not return plain lyrics")
	full := flags.Bool("include-text", false, "Include full lyrics in JSON (default: metadata only)")
	stdin := flags.Bool("stdin", false, "Read Track JSON from stdin instead of command-line metadata")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(errOut, "unexpected positional arguments")
		return 2
	}
	track := lyricfetch.Track{Title: *title, Artist: *artist, Album: *album, Duration: *duration}
	if *stdin {
		decoder := json.NewDecoder(io.LimitReader(in, 65537))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&track); err != nil {
			fmt.Fprintln(errOut, "invalid track JSON")
			return 2
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			fmt.Fprintln(errOut, "unexpected trailing JSON")
			return 2
		}
	}
	options := lyricfetch.Options{Budget: *budget, SyncedOnly: *syncedOnly}
	if *providers != "" {
		for _, value := range strings.Split(*providers, ",") {
			options.Providers = append(options.Providers, lyricfetch.Source(strings.TrimSpace(value)))
		}
	}
	client, err := lyricfetch.New(options)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	defer client.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	result, err := client.Search(ctx, track)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	var lyric any
	if result.Lyrics != nil {
		if *full {
			lyric = result.Lyrics
		} else {
			lyric = struct {
				Source lyricfetch.Source `json:"source"`
				Synced bool              `json:"synced"`
				Lines  int               `json:"lines"`
			}{result.Lyrics.Source, result.Lyrics.Synced, len(strings.Split(result.Lyrics.Text, "\n"))}
		}
	}
	if err := json.NewEncoder(out).Encode(struct {
		Lyrics   any                  `json:"lyrics"`
		Attempts []lyricfetch.Attempt `json:"attempts"`
	}{lyric, result.Attempts}); err != nil {
		fmt.Fprintln(errOut, "cannot write result")
		return 2
	}
	if result.Lyrics == nil {
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
