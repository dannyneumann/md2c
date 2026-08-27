package convert

import (
	"strings"
	"testing"
)

func TestMermaidToPlantUML(t *testing.T) {
	t.Parallel()

	src := "```mermaid\nflowchart LR\n    subgraph DEV[\"Team\"]\n        A[\"Start\"]\n        B[\"Ende\"]\n        A --> B\n    end\n    C[\"Aussen\"]\n    B -->|ok| C\n    A -.->|hint| C\n```\n"
	got, err := Convert(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `ac:name="plantuml"`) {
		t.Fatalf("expected plantuml macro, got:\n%s", got)
	}
	if !strings.Contains(got, "!pragma layout smetana") {
		t.Fatalf("expected smetana layout (no Graphviz), got:\n%s", got)
	}
	if !strings.Contains(got, "left to right direction") {
		t.Fatalf("expected LR direction, got:\n%s", got)
	}
	if !strings.Contains(got, `rectangle "Team"`) {
		t.Fatalf("expected subgraph rectangle, got:\n%s", got)
	}
	if !strings.Contains(got, "A --> B") {
		t.Fatalf("expected edge, got:\n%s", got)
	}
	if !strings.Contains(got, "B --> C : ok") {
		t.Fatalf("expected labeled edge, got:\n%s", got)
	}
	if !strings.Contains(got, "A ..> C : hint") {
		t.Fatalf("expected dotted edge, got:\n%s", got)
	}
	if strings.Contains(got, `ac:name="code"`) {
		t.Fatalf("mermaid should not stay a code block:\n%s", got)
	}
}

func TestMermaidChainedEdges(t *testing.T) {
	t.Parallel()
	puml, ok := mermaidToPlantUML("flowchart TD\nCODE --> TESTS --> QG\n")
	if !ok {
		t.Fatal("expected conversion")
	}
	if !strings.Contains(puml, "CODE --> TESTS") || !strings.Contains(puml, "TESTS --> QG") {
		t.Fatalf("chained edges not split:\n%s", puml)
	}
}

func TestMermaidUnsupportedFallsBackToCode(t *testing.T) {
	t.Parallel()
	got, err := Convert("```mermaid\nsequenceDiagram\n    Alice->>Bob: hi\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `ac:name="code"`) {
		t.Fatalf("unsupported mermaid should stay code, got:\n%s", got)
	}
	if strings.Contains(got, `ac:name="plantuml"`) {
		t.Fatalf("sequenceDiagram must not become plantuml:\n%s", got)
	}
}

func TestMermaidInlineNodeOnEdge(t *testing.T) {
	t.Parallel()
	puml, ok := mermaidToPlantUML("flowchart LR\nQG -->|ok| G0[\"Anlieferung\"]\nG0 --> G1[\"ITU\"]\n")
	if !ok {
		t.Fatal("expected conversion")
	}
	if !strings.Contains(puml, `rectangle "Anlieferung" as G0`) {
		t.Fatalf("inline node missing:\n%s", puml)
	}
	if !strings.Contains(puml, "QG --> G0 : ok") {
		t.Fatalf("edge missing:\n%s", puml)
	}
}

func TestPlantUMLFence(t *testing.T) {
	t.Parallel()
	got, err := Convert("```plantuml\nAlice -> Bob: hi\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `ac:name="plantuml"`) {
		t.Fatalf("expected plantuml macro, got:\n%s", got)
	}
	if !strings.Contains(got, "@startuml") {
		t.Fatalf("expected @startuml wrapper, got:\n%s", got)
	}
}
