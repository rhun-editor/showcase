package main

import (
	"testing"

	"github.com/rhun-editor/extension-protocol/go/rhunext"
)

func TestChangeCase(t *testing.T) {
	cases := map[string]string{"upper": "HELLO WORLD", "lower": "hello world", "title": "Hello World", "reverse": "dlrow olleh"}
	for how, want := range cases {
		if got := changeCase("hello world", how); got != want {
			t.Errorf("%s: %q, want %q", how, got, want)
		}
	}
}

func TestWrapAndStats(t *testing.T) {
	if got := wrap("x", "[]"); got != "[x]" {
		t.Errorf("wrap: %q", got)
	}
	if l, w, c := stats("one two\nthree\n"); l != 2 || w != 3 || c != 14 {
		t.Errorf("stats: %d %d %d", l, w, c)
	}
	if l, _, _ := stats("no newline"); l != 1 {
		t.Errorf("stats without a final newline: %d lines", l)
	}
}

func TestTodoMarks(t *testing.T) {
	marks := todoMarks("a\n// TODO: write it\nx := 1 // FIXME broken\n# NOTE look\n")
	if len(marks) != 3 {
		t.Fatalf("%d marks", len(marks))
	}
	if m := marks[0]; m.Severity != rhunext.SevWarning || m.Message != "TODO: write it" ||
		m.Range.Start != (rhunext.Pos{Line: 2, Col: 4}) || m.Range.End != (rhunext.Pos{Line: 2, Col: 18}) {
		t.Errorf("TODO mark: %+v", m)
	}
	if marks[1].Severity != rhunext.SevError || marks[2].Severity != rhunext.SevInfo {
		t.Errorf("severities: %v %v", marks[1].Severity, marks[2].Severity)
	}
}

func TestNextTodoAndWords(t *testing.T) {
	text := "TODO one\nplain\nx FIXME two\n"
	rg, ok := nextTodo(text, rhunext.Pos{Line: 1, Col: 2})
	if !ok || rg.Start != (rhunext.Pos{Line: 3, Col: 3}) || rg.End != (rhunext.Pos{Line: 3, Col: 8}) {
		t.Errorf("next after line 1: %+v %v", rg, ok)
	}
	rg, ok = nextTodo(text, rhunext.Pos{Line: 3, Col: 9})
	if !ok || rg.Start != (rhunext.Pos{Line: 1, Col: 1}) {
		t.Errorf("wraps to the top: %+v %v", rg, ok)
	}
	if w := wordAt("let foo_bar = 1", rhunext.Pos{Line: 1, Col: 6}); w != "foo_bar" {
		t.Errorf("wordAt: %q", w)
	}
	if n := countWord("foo food foo_x foo", "foo"); n != 2 {
		t.Errorf("countWord: %d", n)
	}
}
