// showcase: a tour of everything a rhun extension can do. Each command uses one part of the
// protocol (https://rhun.app/extensions/v1/): read and change text as one undo step, select,
// toast and status text, open files, marks with severities, hover text, the edits stream,
// per-file options, pick lists and questions, terminal commands the user runs, rhun's own
// commands, commands added while running, and the extension's own settings. A theme and a
// grammar in its folder show the data-file side.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/rhun-editor/extension-protocol/go/rhunext"
)

// a document rhun told us about
type doc struct {
	path    string
	lang    string
	version int64
	cursor  rhunext.Pos
	anchor  rhunext.Pos // equal to cursor without a selection
	marked  bool        // our marks are on it
}

type showcase struct {
	ext      *rhunext.Ext
	project  string
	settings rhunext.Settings
	docs     map[rhunext.Doc]*doc
	active   rhunext.Doc
	edits    int  // edit records seen since the start (the edits stream)
	added    bool // the runtime command is registered
}

// the tour's demos, in the order the pick list shows them
var demos = []struct{ command, title string }{
	{"greet", "Insert a greeting at the cursor (insert)"},
	{"shout", "Change the case of the selection (range, replace)"},
	{"wrap", "Wrap the selection in brackets (range, replace)"},
	{"stats", "Count lines and words (text, toast)"},
	{"marks", "Mark TODO and FIXME (diagnostics)"},
	{"clear", "Clear the marks (diagnostics)"},
	{"next", "Go to the next TODO (select)"},
	{"hello", "Ask for a name and greet it (ask, insert)"},
	{"readme", "Open the project's README (open)"},
	{"terminal", "Suggest a command for the terminal (term)"},
	{"wordwrap", "Toggle word wrap (cmd)"},
	{"indent", "Set this file's tab width (option)"},
	{"more", "Add a command while running (register)"},
}

func (s *showcase) Ready(ext *rhunext.Ext, project string) {
	s.ext = ext
	s.project = project
	s.docs = map[rhunext.Doc]*doc{}
}

func (s *showcase) Event(ext *rhunext.Ext, ev rhunext.Event) {
	if s.settings.Update(ev) {
		key, _, _ := rhunext.ParseSetting(ev)
		switch key {
		case "live_marks":
			for id, d := range s.docs {
				if s.settings.Bool("live_marks", true) {
					s.markTodos(id, d)
				} else if d.marked {
					s.clearMarks(id, d)
				}
			}
		case "status_words":
			s.refreshStatus()
		}
		return
	}
	switch ev.Name {
	case "opened":
		o, ok := rhunext.ParseOpened(ev)
		if !ok {
			return
		}
		d := &doc{path: o.Path, lang: o.Lang, version: o.Version, cursor: rhunext.Pos{Line: 1, Col: 1}}
		d.anchor = d.cursor
		s.docs[o.Doc] = d
		if s.settings.Bool("live_marks", true) {
			s.markTodos(o.Doc, d)
		}
	case "active":
		id, _ := rhunext.ParseDoc(ev)
		s.active = id
		s.refreshStatus()
	case "changed":
		id, version, ok := rhunext.ParseVersion(ev)
		d := s.docs[id]
		if !ok || d == nil {
			return
		}
		d.version = version
		if s.settings.Bool("live_marks", true) {
			s.markTodos(id, d)
		}
		if id == s.active {
			s.refreshStatus()
		}
	case "cursor":
		if id, at, ok := rhunext.ParseCursor(ev); ok {
			if d := s.docs[id]; d != nil {
				d.cursor, d.anchor = at, at
			}
		}
	case "selection":
		if id, rg, ok := rhunext.ParseSelection(ev); ok {
			if d := s.docs[id]; d != nil {
				d.anchor, d.cursor = rg.Start, rg.End
			}
		}
	case "edits":
		if records, err := rhunext.ParseEdits(ev.Body); err == nil {
			s.edits += len(records)
		}
	case "closed":
		id, _ := rhunext.ParseDoc(ev)
		delete(s.docs, id)
	case "hover":
		id, at, ok := rhunext.ParseHover(ev)
		if ok {
			s.hover(id, at)
		}
	}
}

func (s *showcase) Run(ext *rhunext.Ext, command string, id rhunext.Doc) {
	name := strings.TrimPrefix(command, "showcase.")
	if name == "tour" {
		titles := make([]string, len(demos))
		for i, d := range demos {
			titles[i] = d.title
		}
		n, err := ext.Choose("Showcase: pick a demo", titles)
		if err != nil {
			return // dismissed, or another prompt is open
		}
		name = demos[n].command
	}
	d := s.docs[id]
	needsDoc := name != "readme" && name != "terminal" && name != "wordwrap" && name != "more"
	if needsDoc && d == nil {
		ext.Toast("Showcase: open a file first")
		return
	}
	switch name {
	case "greet":
		s.insert(id, s.settings.String("greeting", "Hello from the Showcase extension!"))
	case "shout":
		s.transformSelection(id, d, func(t string) string { return changeCase(t, s.settings.String("case", "upper")) })
	case "wrap":
		pair := s.settings.String("brackets", "()")
		s.transformSelection(id, d, func(t string) string { return wrap(t, pair) })
	case "stats":
		text, _, err := ext.Text(id)
		if err != nil {
			return
		}
		lines, words, chars := stats(string(text))
		ext.Toast(fmt.Sprintf("%d lines, %d words, %d characters; %d edits seen since rhun started",
			lines, words, chars, s.edits))
	case "marks":
		s.markTodos(id, d)
	case "clear":
		s.clearMarks(id, d)
	case "next":
		text, _, err := ext.Text(id)
		if err != nil {
			return
		}
		if rg, ok := nextTodo(string(text), d.cursor); ok {
			ext.Select(id, rg)
		} else {
			ext.Toast("Showcase: no TODO after the cursor")
		}
	case "hello":
		who, err := ext.Ask("Who should the Showcase greet?")
		if err != nil || strings.TrimSpace(who) == "" {
			return
		}
		s.insert(id, "Hello, "+strings.TrimSpace(who)+"!")
	case "readme":
		for _, n := range []string{"README.md", "README", "readme.md", "README.txt"} {
			if s.project == "" {
				break
			}
			if _, err := os.Stat(filepath.Join(s.project, n)); err == nil {
				ext.Open(n, rhunext.Pos{Line: 1, Col: 1})
				return
			}
		}
		ext.Toast("Showcase: this project has no README")
	case "terminal":
		ext.Term(termDemo())
	case "wordwrap":
		if err := ext.Cmd("toggle_word_wrap"); err == nil {
			ext.Toast("Showcase: word wrap toggled through rhun's own command")
		}
	case "indent":
		width := s.settings.Int("tab_width", 2)
		if ext.Option(id, "tab_width", fmt.Sprint(width)) == nil && ext.Option(id, "insert_spaces", "yes") == nil {
			ext.Toast(fmt.Sprintf("Showcase: this file now indents with %d spaces", width))
		}
	case "more":
		if s.added {
			ext.Toast("Showcase: Insert the Time is already in the palette")
			return
		}
		if err := ext.Register("time", "", "Insert the Time"); err == nil {
			s.added = true
			ext.Toast("Showcase: added Insert the Time (Showcase) to the command palette")
		}
	case "time":
		if d != nil {
			s.insert(id, time.Now().Format("15:04"))
		}
	}
}

// insert types text at the cursor of the document, at its current version
func (s *showcase) insert(id rhunext.Doc, text string) {
	_, version, err := s.ext.Text(id)
	if err != nil {
		return
	}
	s.ext.Insert(id, version, []byte(text))
}

// transformSelection replaces the selection with f of it, as one undo step
func (s *showcase) transformSelection(id rhunext.Doc, d *doc, f func(string) string) {
	start, end := d.anchor, d.cursor
	if before(end, start) {
		start, end = end, start
	}
	if start == end {
		s.ext.Toast("Showcase: select some text first")
		return
	}
	rg := rhunext.Range{Start: start, End: end}
	text, version, err := s.ext.RangeText(id, rg)
	if err != nil {
		return
	}
	s.ext.Replace(id, version, rg, []byte(f(string(text))))
}

func (s *showcase) markTodos(id rhunext.Doc, d *doc) {
	text, version, err := s.ext.Text(id)
	if err != nil {
		return
	}
	marks := todoMarks(string(text))
	if len(marks) == 0 && !d.marked {
		return
	}
	if s.ext.Diagnostics(id, version, marks) == nil {
		d.marked = len(marks) > 0
	}
}

func (s *showcase) clearMarks(id rhunext.Doc, d *doc) {
	_, version, err := s.ext.Text(id)
	if err != nil {
		return
	}
	if s.ext.Diagnostics(id, version, nil) == nil {
		d.marked = false
	}
}

func (s *showcase) refreshStatus() {
	if !s.settings.Bool("status_words", true) || s.docs[s.active] == nil {
		s.ext.Status("")
		return
	}
	text, _, err := s.ext.Text(s.active)
	if err != nil {
		return
	}
	_, words, _ := stats(string(text))
	s.ext.Status(fmt.Sprintf("Showcase: %d words", words))
}

func (s *showcase) hover(id rhunext.Doc, at rhunext.Pos) {
	text, _, err := s.ext.Text(id)
	if err != nil {
		return
	}
	word := wordAt(string(text), at)
	if word == "" {
		return
	}
	n := countWord(string(text), word)
	times := "times"
	if n == 1 {
		times = "time"
	}
	s.ext.Hover(id, at, fmt.Sprintf("“%s” appears %d %s in this file", word, n, times))
}

// ---- the text work, kept pure for the tests ----

func before(a, b rhunext.Pos) bool {
	return a.Line < b.Line || a.Line == b.Line && a.Col < b.Col
}

func changeCase(t, how string) string {
	switch how {
	case "lower":
		return strings.ToLower(t)
	case "title":
		out := []rune(t)
		start := true
		for i, r := range out {
			if unicode.IsLetter(r) {
				if start {
					out[i] = unicode.ToUpper(r)
				} else {
					out[i] = unicode.ToLower(r)
				}
				start = false
			} else {
				start = unicode.IsSpace(r)
			}
		}
		return string(out)
	case "reverse":
		r := []rune(t)
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
		return string(r)
	}
	return strings.ToUpper(t)
}

func wrap(t, pair string) string {
	r := []rune(pair)
	if len(r) != 2 {
		return "(" + t + ")"
	}
	return string(r[0]) + t + string(r[1])
}

func stats(t string) (lines, words, chars int) {
	if t != "" {
		lines = strings.Count(t, "\n")
		if !strings.HasSuffix(t, "\n") {
			lines++
		}
	}
	return lines, len(strings.Fields(t)), len([]rune(t))
}

// todoMarks: FIXME is an error, TODO a warning, NOTE information, IDEA a hint; each mark runs from
// the word to the end of its line, and says what follows the word
func todoMarks(t string) []rhunext.Diagnostic {
	kinds := []struct {
		word string
		sev  rhunext.Severity
	}{{"FIXME", rhunext.SevError}, {"TODO", rhunext.SevWarning}, {"NOTE", rhunext.SevInfo}, {"IDEA", rhunext.SevHint}}
	var out []rhunext.Diagnostic
	for n, line := range strings.Split(t, "\n") {
		for _, k := range kinds {
			i := strings.Index(line, k.word)
			if i < 0 {
				continue
			}
			rest := strings.TrimLeft(line[i+len(k.word):], ":- \t")
			msg := k.word
			if rest != "" {
				msg += ": " + rest
			}
			out = append(out, rhunext.Diagnostic{
				Range:    rhunext.Range{Start: rhunext.Pos{Line: n + 1, Col: i + 1}, End: rhunext.Pos{Line: n + 1, Col: len(line) + 1}},
				Severity: k.sev, Message: msg,
			})
			break
		}
	}
	return out
}

// nextTodo: the first TODO or FIXME after the cursor (wrapping to the top), selected
func nextTodo(t string, from rhunext.Pos) (rhunext.Range, bool) {
	lines := strings.Split(t, "\n")
	find := func(line string, after int) int {
		best := -1
		for _, w := range []string{"TODO", "FIXME"} {
			if i := strings.Index(line[min(after, len(line)):], w); i >= 0 {
				i += min(after, len(line))
				if best < 0 || i < best {
					best = i
				}
			}
		}
		return best
	}
	for pass := 0; pass < 2; pass++ {
		for n := range lines {
			line := lines[n]
			after := 0
			if pass == 0 {
				if n+1 < from.Line {
					continue
				}
				if n+1 == from.Line {
					after = from.Col // past the cursor
				}
			}
			if i := find(line, after); i >= 0 {
				w := 4
				if strings.HasPrefix(line[i:], "FIXME") {
					w = 5
				}
				return rhunext.Range{Start: rhunext.Pos{Line: n + 1, Col: i + 1}, End: rhunext.Pos{Line: n + 1, Col: i + 1 + w}}, true
			}
		}
	}
	return rhunext.Range{}, false
}

func isWordByte(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= 0x80
}

// wordAt: the word under a position (byte columns)
func wordAt(t string, at rhunext.Pos) string {
	lines := strings.Split(t, "\n")
	if at.Line < 1 || at.Line > len(lines) {
		return ""
	}
	line := lines[at.Line-1]
	i := at.Col - 1
	if i < 0 || i >= len(line) || !isWordByte(line[i]) {
		return ""
	}
	a, b := i, i
	for a > 0 && isWordByte(line[a-1]) {
		a--
	}
	for b < len(line) && isWordByte(line[b]) {
		b++
	}
	return line[a:b]
}

// countWord: whole-word occurrences
func countWord(t, word string) int {
	n := 0
	for i := 0; ; {
		j := strings.Index(t[i:], word)
		if j < 0 {
			return n
		}
		j += i
		end := j + len(word)
		if (j == 0 || !isWordByte(t[j-1])) && (end == len(t) || !isWordByte(t[end])) {
			n++
		}
		i = end
	}
}

func main() {
	if err := rhunext.Run(&showcase{}); err != nil {
		os.Exit(1)
	}
}

// termDemo is the command the Terminal demo suggests: the arithmetic shows that the shell ran
// it. rhun's terminal is a POSIX shell on Linux and macOS and PowerShell on Windows.
func termDemo() string {
	if runtime.GOOS == "windows" {
		return `echo "Showcase ran this: $(6 * 7)"`
	}
	return `echo "Showcase ran this: $((6 * 7))"`
}
