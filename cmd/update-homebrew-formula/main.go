package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"md2confluence/internal/homebrew"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("update-homebrew-formula", flag.ContinueOnError)
	fs.SetOutput(stderr)
	version := fs.String("version", "", "release tag, e.g. v0.2.2")
	sumsPath := fs.String("sums", "", "path to SHA256SUMS")
	out := fs.String("out", "Formula/md2c.rb", "output formula path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version == "" || *sumsPath == "" {
		fmt.Fprintln(stderr, "usage: update-homebrew-formula -version vX.Y.Z -sums SHA256SUMS [-out Formula/md2c.rb]")
		return 2
	}
	f, err := os.Open(*sumsPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer f.Close()
	sums, err := homebrew.ParseSHA256SUMS(f)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := homebrew.WriteFormula(*out, *version, sums); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
