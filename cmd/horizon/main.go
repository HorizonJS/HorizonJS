package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"horizonjs/internal/buildtool"
)

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: horizon build [--tag main] [--docs go|bundled] [--root directory] [--out directory]")
	}
	if args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(stdout, "Usage: horizon build [--tag main] [--docs go|bundled] [--root directory] [--out directory]")
		return nil
	}
	if args[0] != "build" {
		return fmt.Errorf("unknown command %q", args[0])
	}
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var opts buildtool.Options
	flags.StringVar(&opts.Tag, "tag", "main", "release tag; main also assembles the website")
	flags.StringVar(&opts.Docs, "docs", "go", "go generates native docs; bundled copies the original docs")
	flags.StringVar(&opts.Root, "root", "", "repository root (auto-detected from the working directory)")
	flags.StringVar(&opts.Out, "out", "", "output directory (default: <root>/build)")
	if err := flags.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	opts.Log = stdout
	return buildtool.Build(ctx, opts)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "Build failed:", err)
		os.Exit(1)
	}
}
