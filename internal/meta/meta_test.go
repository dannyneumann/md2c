package meta

import (
	"strings"
	"testing"
)

func TestExtractShortCommaForm(t *testing.T) {
	t.Parallel()
	in := `<!-- space:PSE,path:Decisions+Konzepte,title:Draft: Stagingkonzept-Telematik-Anwendungen -->
# Inhalt
`
	got, rest := Extract(in)
	if got.Space != "PSE" || got.Path != "Decisions+Konzepte" {
		t.Fatalf("got %+v", got)
	}
	if got.Title != "Draft: Stagingkonzept-Telematik-Anwendungen" {
		t.Fatalf("title %q", got.Title)
	}
	if dest := got.Destination(); dest != "Decisions+Konzepte/Draft: Stagingkonzept-Telematik-Anwendungen" {
		t.Fatalf("destination %q", dest)
	}
	if !strings.HasPrefix(rest, "# Inhalt") {
		t.Fatalf("rest %q", rest)
	}
}

func TestExtractLegacyMetadataPrefix(t *testing.T) {
	t.Parallel()
	in := `<-- metadata.Space: PSE; metadata.Path: Decisions+Konzepte; metaData.Title: Draft: Stagingkonzept-Telematik-Anwendungen -->
# Inhalt

Hallo.
`
	got, rest := Extract(in)
	if got.Space != "PSE" {
		t.Fatalf("space %q", got.Space)
	}
	if got.Path != "Decisions+Konzepte" {
		t.Fatalf("path %q", got.Path)
	}
	if got.Title != "Draft: Stagingkonzept-Telematik-Anwendungen" {
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
	in := `<!-- space:PSE,path:Decisions+Konzepte,title: Draft: Stgingkonzept für die Applikationen in der Telematik -->
# x
`
	got, _ := Extract(in)
	want := "Draft: Stgingkonzept für die Applikationen in der Telematik"
	if got.Title != want {
		t.Fatalf("title %q want %q", got.Title, want)
	}
	if got.Destination() != "Decisions+Konzepte/"+want {
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
