package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCLIArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown"}, {"build", "--unknown"}, {"build", "unexpected"}, {"build", "--docs", "unknown"}, {"build", "--docs", "docker"}, {"build", "--tag", "../escape"}} {
		var log bytes.Buffer
		if err := run(context.Background(), args, &log, &log); err == nil {
			t.Fatalf("%v: expected failure", args)
		}
		if strings.Contains(log.String(), "completed successfully") {
			t.Fatalf("%v: false success", args)
		}
	}
	for _, args := range [][]string{{"--help"}, {"build", "--help"}} {
		if err := run(context.Background(), args, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	}
}
