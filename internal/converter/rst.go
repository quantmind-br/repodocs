package converter

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
)

// ConvertRST transforms reStructuredText bytes into Markdown bytes.
// Best-effort: unsupported constructs degrade to plain text rather than failing.
func ConvertRST(input []byte) ([]byte, error) {
	c := newRSTConverter(input)
	return c.convert(), nil
}

type rstConverter struct {
	lines      []string
	i          int
	out        *bytes.Buffer
	headingMap map[byte]int
	nextLevel  int
}

func newRSTConverter(input []byte) *rstConverter {
	text := strings.ReplaceAll(string(input), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return &rstConverter{
		lines:      strings.Split(text, "\n"),
		out:        &bytes.Buffer{},
		headingMap: make(map[byte]int),
	}
}

var (
	admonitionRE  = regexp.MustCompile(`^(\s*)\.\. (note|warning|tip|important|caution|hint|attention|danger|error|admonition|seealso)::\s*(.*)$`)
	codeBlockRE   = regexp.MustCompile(`^(\s*)\.\. (?:code-block|sourcecode|code)::\s*(\S*)\s*$`)
	imageRE       = regexp.MustCompile(`^(\s*)\.\. (?:image|figure)::\s*(\S+)\s*$`)
	dropDirectRE  = regexp.MustCompile(`^(\s*)\.\. (?:toctree|autodoc|automodule|autoclass|autofunction|automethod|autoattribute|autosummary|include|index|highlight|default-domain|module|currentmodule|deprecated|versionadded|versionchanged|graphviz|raw|epigraph|sectionauthor|moduleauthor|rubric|topic|contents|meta|todo|todolist|glossary|productionlist|tabularcolumns|only)::`)
	linkTargetRE  = regexp.MustCompile(`^(\s*)\.\. _[^:]+:.*$`)
	commentRE     = regexp.MustCompile(`^(\s*)\.\.(?:\s|$)`)
	bulletRE      = regexp.MustCompile(`^(\s*)([*+\-])\s+(.*)$`)
	enumRE        = regexp.MustCompile(`^(\s*)(\d+|#)\.\s+(.*)$`)
	fieldListRE   = regexp.MustCompile(`^(\s*):[^:]+:\s+.*$`)
	headerRuneSet = "=-~^\"*+#`:.'_"
)

var (
	inlineLiteralRE = regexp.MustCompile("``([^`]+)``")
	hyperlinkRE     = regexp.MustCompile("`([^`<]+?)\\s*<([^`>]+)>`_+")
	roleRE          = regexp.MustCompile(":([a-zA-Z][a-zA-Z0-9_+-]*):`([^`]+)`")
	simpleRefRE     = regexp.MustCompile("`([^`]+)`_+")
)

var admonitionTitle = map[string]string{
	"note":       "NOTE",
	"warning":    "WARNING",
	"tip":        "TIP",
	"important":  "IMPORTANT",
	"caution":    "CAUTION",
	"hint":       "TIP",
	"attention":  "IMPORTANT",
	"danger":     "WARNING",
	"error":      "WARNING",
	"admonition": "NOTE",
	"seealso":    "NOTE",
}

func (c *rstConverter) convert() []byte {
	for c.i < len(c.lines) {
		if c.tryHeading() {
			continue
		}
		if c.tryDirective() {
			continue
		}
		if c.tryLiteralBlock() {
			continue
		}
		if c.tryList() {
			continue
		}
		c.emitParagraphLine()
	}
	out := c.out.Bytes()
	out = collapseBlankLines(out)
	return out
}

// tryHeading looks for a non-empty line followed by an underline of one
// of the section-adornment characters, with length >= text length - 1.
// It also tolerates the over-and-underline form where the text is sandwiched
// between two identical adornment lines.
func (c *rstConverter) tryHeading() bool {
	if c.i >= len(c.lines) {
		return false
	}
	line := c.lines[c.i]
	trimmed := strings.TrimRight(line, " \t")

	// Over+underline form: adornment / text / adornment
	if isAdornmentLine(trimmed) && c.i+2 < len(c.lines) {
		text := strings.TrimRight(c.lines[c.i+1], " \t")
		under := strings.TrimRight(c.lines[c.i+2], " \t")
		if text != "" && isAdornmentLine(under) && trimmed[0] == under[0] {
			level := c.headingLevel(trimmed[0])
			c.writeHeading(level, strings.TrimSpace(text))
			c.i += 3
			c.skipBlanks()
			return true
		}
	}

	if trimmed == "" {
		c.out.WriteByte('\n')
		c.i++
		return true
	}

	if c.i+1 >= len(c.lines) {
		return false
	}
	next := strings.TrimRight(c.lines[c.i+1], " \t")
	if !isAdornmentLine(next) {
		return false
	}
	if len(next) < len([]rune(trimmed))-1 {
		return false
	}
	level := c.headingLevel(next[0])
	c.writeHeading(level, strings.TrimSpace(trimmed))
	c.i += 2
	c.skipBlanks()
	return true
}

func (c *rstConverter) writeHeading(level int, text string) {
	if level > 6 {
		level = 6
	}
	c.out.WriteString(strings.Repeat("#", level))
	c.out.WriteByte(' ')
	c.out.WriteString(processInline(text))
	c.out.WriteString("\n\n")
}

func (c *rstConverter) headingLevel(ch byte) int {
	if lvl, ok := c.headingMap[ch]; ok {
		return lvl
	}
	c.nextLevel++
	c.headingMap[ch] = c.nextLevel
	return c.nextLevel
}

func (c *rstConverter) tryDirective() bool {
	line := c.lines[c.i]

	if m := codeBlockRE.FindStringSubmatch(line); m != nil {
		base := len(m[1])
		lang := m[2]
		c.i++
		c.captureOptions()
		body := c.captureIndentedBody(base)
		c.out.WriteString("```" + lang + "\n")
		for _, b := range body {
			c.out.WriteString(b)
			c.out.WriteByte('\n')
		}
		c.out.WriteString("```\n\n")
		return true
	}

	if m := admonitionRE.FindStringSubmatch(line); m != nil {
		base := len(m[1])
		kind := strings.ToLower(m[2])
		title := admonitionTitle[kind]
		if title == "" {
			title = "NOTE"
		}
		first := strings.TrimSpace(m[3])
		c.i++
		c.captureOptions()
		body := c.captureIndentedBody(base)
		var blocks []string
		if first != "" {
			blocks = append(blocks, first)
		}
		blocks = append(blocks, joinBody(body))
		c.out.WriteString("> [!" + title + "]\n")
		for idx, block := range blocks {
			if idx > 0 {
				c.out.WriteString(">\n")
			}
			for ln := range strings.SplitSeq(block, "\n") {
				if ln == "" {
					c.out.WriteString(">\n")
				} else {
					c.out.WriteString("> ")
					c.out.WriteString(processInline(ln))
					c.out.WriteByte('\n')
				}
			}
		}
		c.out.WriteString("\n")
		return true
	}

	if m := imageRE.FindStringSubmatch(line); m != nil {
		base := len(m[1])
		path := m[2]
		c.i++
		options := c.captureOptions()
		c.captureIndentedBody(base)
		alt := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		for _, opt := range options {
			if rest, ok := strings.CutPrefix(opt, ":alt:"); ok {
				alt = strings.TrimSpace(rest)
			}
		}
		c.out.WriteString("![" + alt + "](" + path + ")\n\n")
		return true
	}

	if m := dropDirectRE.FindStringSubmatch(line); m != nil {
		base := len(m[1])
		c.i++
		c.captureOptions()
		c.captureIndentedBody(base)
		return true
	}

	if m := linkTargetRE.FindStringSubmatch(line); m != nil {
		base := len(m[1])
		c.i++
		c.captureOptions()
		c.captureIndentedBody(base)
		return true
	}

	// Generic comment / unknown directive — drop it and any indented body.
	if m := commentRE.FindStringSubmatch(line); m != nil {
		base := len(m[1])
		c.i++
		c.captureOptions()
		c.captureIndentedBody(base)
		return true
	}

	return false
}

// tryLiteralBlock detects a paragraph ending in `::` followed by an indented
// block, emitting the leading text (with one colon stripped) and a fenced
// code block.
func (c *rstConverter) tryLiteralBlock() bool {
	line := c.lines[c.i]
	trimmed := strings.TrimRight(line, " \t")
	if !strings.HasSuffix(trimmed, "::") {
		return false
	}
	// Must be followed (after blank lines) by an indented block.
	saved := c.i
	c.i++
	skipBlanks(&c.i, c.lines)
	if c.i >= len(c.lines) {
		c.i = saved
		return false
	}
	bodyIndent := indentOf(c.lines[c.i])
	if bodyIndent <= indentOf(line) {
		c.i = saved
		return false
	}
	// Emit leading text (replace `::` with `:` if there's text before it).
	head := strings.TrimSuffix(trimmed, "::")
	headTrim := strings.TrimRight(head, " \t")
	if headTrim != "" {
		c.out.WriteString(processInline(headTrim) + ":\n\n")
	}
	// Capture lines until indent drops back to <= original indent.
	origIndent := indentOf(line)
	var body []string
	for c.i < len(c.lines) {
		l := c.lines[c.i]
		if strings.TrimSpace(l) == "" {
			body = append(body, "")
			c.i++
			continue
		}
		if indentOf(l) <= origIndent {
			break
		}
		if len(l) >= bodyIndent {
			body = append(body, l[bodyIndent:])
		} else {
			body = append(body, strings.TrimLeft(l, " \t"))
		}
		c.i++
	}
	for len(body) > 0 && body[len(body)-1] == "" {
		body = body[:len(body)-1]
	}
	c.out.WriteString("```\n")
	for _, b := range body {
		c.out.WriteString(b)
		c.out.WriteByte('\n')
	}
	c.out.WriteString("```\n\n")
	return true
}

func (c *rstConverter) tryList() bool {
	line := c.lines[c.i]
	if m := bulletRE.FindStringSubmatch(line); m != nil {
		c.out.WriteString(m[1] + "- " + processInline(m[3]) + "\n")
		c.i++
		return true
	}
	if m := enumRE.FindStringSubmatch(line); m != nil {
		marker := m[2]
		if marker == "#" {
			marker = "1"
		}
		c.out.WriteString(m[1] + marker + ". " + processInline(m[3]) + "\n")
		c.i++
		return true
	}
	return false
}

func (c *rstConverter) emitParagraphLine() {
	line := c.lines[c.i]
	if strings.TrimSpace(line) == "" {
		c.out.WriteByte('\n')
		c.i++
		return
	}
	// Drop standalone field-list lines (`:field: value`) outside admonitions/directives;
	// they are RST metadata noise (`:author:`, `:date:`, etc.) at the document top.
	if fieldListRE.MatchString(line) && c.out.Len() == 0 {
		c.i++
		return
	}
	c.out.WriteString(processInline(line))
	c.out.WriteByte('\n')
	c.i++
}

// captureOptions consumes and returns the `:option: value` lines that
// directly follow a directive. It must be called before captureIndentedBody.
func (c *rstConverter) captureOptions() []string {
	var opts []string
	for c.i < len(c.lines) {
		l := c.lines[c.i]
		if strings.TrimSpace(l) == "" {
			break
		}
		t := strings.TrimLeft(l, " \t")
		if !strings.HasPrefix(t, ":") {
			break
		}
		rest := t[1:]
		idx := strings.Index(rest, ":")
		if idx <= 0 || strings.ContainsAny(rest[:idx], " \t") {
			break
		}
		opts = append(opts, t)
		c.i++
	}
	return opts
}

func (c *rstConverter) captureIndentedBody(baseIndent int) []string {
	skipBlanks(&c.i, c.lines)
	if c.i >= len(c.lines) {
		return nil
	}
	first := indentOf(c.lines[c.i])
	if first <= baseIndent {
		return nil
	}
	var body []string
	for c.i < len(c.lines) {
		l := c.lines[c.i]
		if strings.TrimSpace(l) == "" {
			body = append(body, "")
			c.i++
			continue
		}
		if indentOf(l) <= baseIndent {
			break
		}
		if len(l) >= first {
			body = append(body, l[first:])
		} else {
			body = append(body, strings.TrimLeft(l, " \t"))
		}
		c.i++
	}
	for len(body) > 0 && body[len(body)-1] == "" {
		body = body[:len(body)-1]
	}
	return body
}

func (c *rstConverter) skipBlanks() {
	skipBlanks(&c.i, c.lines)
}

func skipBlanks(i *int, lines []string) {
	for *i < len(lines) && strings.TrimSpace(lines[*i]) == "" {
		*i++
	}
}

func indentOf(s string) int {
	n := 0
	for n < len(s) && (s[n] == ' ' || s[n] == '\t') {
		n++
	}
	return n
}

func isAdornmentLine(s string) bool {
	if len(s) < 3 {
		return false
	}
	ch := s[0]
	if !strings.ContainsRune(headerRuneSet, rune(ch)) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] != ch {
			return false
		}
	}
	return true
}

func joinBody(lines []string) string {
	return strings.Join(lines, "\n")
}

// collapseBlankLines limits consecutive blank lines to at most one and trims
// leading/trailing blanks, mirroring typical Markdown formatting.
func collapseBlankLines(input []byte) []byte {
	out := &bytes.Buffer{}
	parts := bytes.Split(input, []byte{'\n'})
	blanks := 0
	started := false
	for _, p := range parts {
		if len(bytes.TrimSpace(p)) == 0 {
			blanks++
			continue
		}
		if started && blanks > 0 {
			out.WriteByte('\n')
			out.WriteByte('\n')
		} else if started {
			out.WriteByte('\n')
		}
		out.Write(p)
		started = true
		blanks = 0
	}
	if started {
		out.WriteByte('\n')
	}
	return out.Bytes()
}

func processInline(s string) string {
	s = inlineLiteralRE.ReplaceAllString(s, "`$1`")
	s = hyperlinkRE.ReplaceAllString(s, "[$1]($2)")
	s = roleRE.ReplaceAllString(s, "$2")
	s = simpleRefRE.ReplaceAllString(s, "$1")
	return s
}
