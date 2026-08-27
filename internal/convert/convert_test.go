package convert

import (
	"strings"
	"testing"
)

func TestConvert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "heading paragraph emphasis",
			in:   "# Title\n\nHello **world** and *italics*.\n",
			want: "<h1>Title</h1><p>Hello <strong>world</strong> and <em>italics</em>.</p>",
		},
		{
			name: "link and inline code",
			in:   "See [docs](https://example.com) and `code`.",
			want: `<p>See <a href="https://example.com">docs</a> and <code>code</code>.</p>`,
		},
		{
			name: "unordered list",
			in:   "- a\n- b\n",
			want: "<ul><li>a</li><li>b</li></ul>",
		},
		{
			name: "ordered list start",
			in:   "3. third\n4. fourth\n",
			want: `<ol start="3"><li>third</li><li>fourth</li></ol>`,
		},
		{
			name: "fenced code",
			in:   "```go\nfmt.Println(\"hi\")\n```\n",
			want: `<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">go</ac:parameter><ac:plain-text-body><![CDATA[fmt.Println("hi")
]]></ac:plain-text-body></ac:structured-macro>`,
		},
		{
			name: "js language alias",
			in:   "```js\ntrue\n```\n",
			want: `<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">javascript</ac:parameter><ac:plain-text-body><![CDATA[true
]]></ac:plain-text-body></ac:structured-macro>`,
		},
		{
			name: "blockquote hr strikethrough",
			in:   "> quote\n\n---\n\n~~old~~\n",
			want: "<blockquote><p>quote</p></blockquote><hr /><p><del>old</del></p>",
		},
		{
			name: "table",
			in:   "| A | B |\n| --- | --- |\n| 1 | 2 |\n",
			want: "<table><tbody><tr><th>A</th><th>B</th></tr><tr><td>1</td><td>2</td></tr></tbody></table>",
		},
		{
			name: "xml escape",
			in:   "Use <tag> & \"quotes\".\n",
			want: "<p>Use &lt;tag&gt; &amp; \"quotes\".</p>",
		},
		{
			name: "remote image",
			in:   "![Logo](https://example.com/logo.png)\n",
			want: `<p><ac:image ac:alt="Logo"><ri:url ri:value="https://example.com/logo.png" /></ac:image></p>`,
		},
		{
			name: "local image placeholder",
			in:   "![Logo](./logo.png)\n",
			want: "<p>[image: Logo (./logo.png)]</p>",
		},
		{
			name: "empty",
			in:   "   \n",
			want: "<p></p>",
		},
		{
			name: "task list",
			in:   "- [x] done\n- [ ] todo\n",
			want: "<ul><li>☑ done</li><li>☐ todo</li></ul>",
		},
		{
			name: "toc paragraph",
			in:   "[TOC]\n\n# Hello\n",
			want: `<ac:structured-macro ac:name="toc"></ac:structured-macro><h1>Hello</h1>`,
		},
		{
			name: "toc heading",
			in:   "## [TOC]\n\n# Hello\n",
			want: `<ac:structured-macro ac:name="toc"></ac:structured-macro><h1>Hello</h1>`,
		},
		{
			name: "toc compact heading",
			in:   "##[TOC]\n\n# Hello\n",
			want: `<ac:structured-macro ac:name="toc"></ac:structured-macro><h1>Hello</h1>`,
		},
		{
			name: "toc lowercase",
			in:   "[toc]\n",
			want: `<ac:structured-macro ac:name="toc"></ac:structured-macro>`,
		},
		{
			name: "toc not in sentence",
			in:   "See [TOC] in the docs.\n",
			want: "<p>See [TOC] in the docs.</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Convert(tt.in)
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestInfoMacro(t *testing.T) {
	t.Parallel()
	got := InfoMacro("Don't edit <this>")
	if !strings.Contains(got, "ac:name=\"info\"") {
		t.Fatalf("missing info macro: %s", got)
	}
	if !strings.Contains(got, "Don't edit &lt;this&gt;") {
		t.Fatalf("prefix not escaped: %s", got)
	}
	if InfoMacro("  ") != "" {
		t.Fatal("empty prefix should yield empty string")
	}
}

func TestCDATAFence(t *testing.T) {
	t.Parallel()
	got, err := Convert("```\nfoo]]>bar\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "foo]]>bar") {
		t.Fatalf("unescaped CDATA terminator: %s", got)
	}
	if !strings.Contains(got, "]]]]><![CDATA[>") {
		t.Fatalf("expected CDATA split, got: %s", got)
	}
}
