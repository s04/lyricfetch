package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestInvalidInput(t *testing.T) {
	for _, args := range [][]string{{}, {"--unknown"}, {"extra"}, {"--title", "Song", "--artist", "Artist", "--providers", "bad"}, {"--stdin"}} {
		var out, errOut bytes.Buffer
		if code := run(args, strings.NewReader("not JSON"), &out, &errOut); code != 2 {
			t.Errorf("args %v code %d", args, code)
		}
		if out.Len() != 0 {
			t.Error("unexpected stdout")
		}
	}
}
