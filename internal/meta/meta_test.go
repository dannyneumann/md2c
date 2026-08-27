package meta

import (
	"strings"
	"testing"
)

func TestExtractShortCommaForm(t *testing.T) {
	t.Parallel()
	in := `<!-- space:DOC,path:Guides,title:Getting started -->
# Inhalt
`
	got, rest := Extract(in)
	if got.Space != "DOC" || got.Path != "Guides" {
		t.Fatalf("got %+v", got)
	}
	if got.Title != "Getting started" {
		t.Fatalf("title %q", got.Title)
	}
	if dest := got.Destination(); dest != "Guides/Getting started" {
		t.Fatalf("destination %q", dest)
	}
	if !strings.HasPrefix(rest, "# Inhalt") {
		t.Fatalf("rest %q", rest)
	}
}

func TestExtractLegacyMetadataPrefix(t *testing.T) {
	t.Parallel()
	in := `<-- metadata.Space: DOC; metadata.Path: Guides; metaData.Title: Getting started -->
# Inhalt

Hallo.
`
	got, rest := Extract(in)
	if got.Space != "DOC" {
		t.Fatalf("space %q", got.Space)
	}
	if got.Path != "Guides" {
		t.Fatalf("path %q", got.Path)
	}
	if got.Title != "Getting started" {
		t.Fatalf("title %q", got.Title)
	}
	if rest != "# Inhalt\n\nHallo.\n" {
		t.Fatalf("rest %q", rest)
	}
}

func TestExtractHTMLCommentAndMultiline(t *testing.T) {
	t.Parallel()
	in := `<!--
space: DEV
path: Engineering/Docs
title: Onboarding
-->
Text
`
	got, rest := Extract(in)
	if got.Space != "DEV" || got.Path != "Engineering/Docs" || got.Title != "Onboarding" {
		t.Fatalf("got %+v", got)
	}
	if rest != "Text\n" {
		t.Fatalf("rest %q", rest)
	}
}

func TestExtractNone(t *testing.T) {
	t.Parallel()
	in := "# Title\n\nBody.\n"
	got, rest := Extract(in)
	if got != (Meta{}) {
		t.Fatalf("got %+v", got)
	}
	if rest != in {
		t.Fatalf("rest changed: %q", rest)
	}
}

func TestExtractTitleKeepsSpacesAndColon(t *testing.T) {
	t.Parallel()
	in := `<!-- space:DOC,path:Guides,title: Draft: Leitfaden für neue Anwendungen -->
# x
`
	got, _ := Extract(in)
	want := "Draft: Leitfaden für neue Anwendungen"
	if got.Title != want {
		t.Fatalf("title %q want %q", got.Title, want)
	}
	if got.Destination() != "Guides/"+want {
		t.Fatalf("destination %q", got.Destination())
	}
}

func TestDestinationTitleOnly(t *testing.T) {
	t.Parallel()
	m := Meta{Title: "Root Page"}
	if got := m.Destination(); got != "Root Page" {
		t.Fatalf("got %q", got)
	}
}
