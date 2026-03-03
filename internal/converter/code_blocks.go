package converter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// knownLanguages is a set of common programming language names used to detect
// bare class names on <code> elements (e.g. class="python").
var knownLanguages = map[string]bool{
	"bash": true, "c": true, "cpp": true, "csharp": true, "css": true,
	"dart": true, "diff": true, "dockerfile": true, "elixir": true,
	"erlang": true, "go": true, "golang": true, "graphql": true,
	"groovy": true, "haskell": true, "hcl": true, "html": true,
	"ini": true, "java": true, "javascript": true, "json": true,
	"jsx": true, "kotlin": true, "lua": true, "makefile": true,
	"markdown": true, "matlab": true, "nginx": true, "objectivec": true,
	"ocaml": true, "perl": true, "php": true, "plaintext": true,
	"powershell": true, "protobuf": true, "python": true, "r": true,
	"ruby": true, "rust": true, "scala": true, "scss": true,
	"sh": true, "shell": true, "sql": true, "swift": true,
	"terraform": true, "text": true, "toml": true, "ts": true,
	"tsx": true, "typescript": true, "vim": true, "xml": true,
	"yaml": true, "yml": true, "zig": true, "zsh": true,
}

// lineNumberClassRegex matches common line-number class patterns.
var lineNumberClassRegex = regexp.MustCompile(
	`(?i)^(ln|line-?number|linenumber|lnos|line-?nos?)$`,
)

// lineNumberTableClassRegex matches line-number table/cell patterns.
var lineNumberTableClassRegex = regexp.MustCompile(
	`(?i)(lntable|ln-table|linenumber-?table|lntd|line-?number-?cell)`,
)

// emptyCodeBlockRegex matches empty fenced code blocks in markdown output.
var emptyCodeBlockRegex = regexp.MustCompile("(?m)^```[a-zA-Z]*\\s*\n\\s*\n```\\s*$")

// PreserveCodeLanguages scans all <pre><code> elements and copies language
// information into a data-repodocs-lang attribute. This survives the
// Readability extraction which strips class attributes.
func PreserveCodeLanguages(sel *goquery.Selection) {
	findWithRoot(sel, "pre code").Each(func(_ int, code *goquery.Selection) {
		lang := detectLanguage(code)
		if lang != "" {
			code.SetAttr("data-repodocs-lang", lang)
		}
	})
}

// RestoreCodeLanguages reads back the data-repodocs-lang attribute and sets
// the class to "language-X" so the markdown converter can pick it up.
func RestoreCodeLanguages(sel *goquery.Selection) {
	findWithRoot(sel, "pre code[data-repodocs-lang]").Each(func(_ int, code *goquery.Selection) {
		lang, exists := code.Attr("data-repodocs-lang")
		if !exists || lang == "" {
			return
		}
		code.SetAttr("class", "language-"+lang)
		code.RemoveAttr("data-repodocs-lang")
	})
}

// NormalizeCodeLanguages ensures all <code> elements have a class="language-X"
// attribute when language info is available from non-standard sources:
//   - data-language="X" or data-lang="X" attributes
//   - bare class names matching known languages (e.g. class="python")
//   - hljs combined classes (e.g. class="hljs python")
func NormalizeCodeLanguages(sel *goquery.Selection) {
	findWithRoot(sel, "code").Each(func(_ int, code *goquery.Selection) {
		// Skip if already has language- or lang- prefix
		if class, exists := code.Attr("class"); exists {
			if strings.Contains(class, "language-") || strings.Contains(class, "lang-") {
				return
			}
		}

		lang := detectLanguageFromAttributes(code)
		if lang != "" {
			existing, _ := code.Attr("class")
			if existing != "" {
				code.SetAttr("class", existing+" language-"+lang)
			} else {
				code.SetAttr("class", "language-"+lang)
			}
		}
	})
}

// StripLineNumbers removes common line-number elements from code blocks.
// Handles patterns like <span class="ln">1</span>, line-number table cells,
// and similar constructs.
func StripLineNumbers(sel *goquery.Selection) {
	// Remove span-based line numbers inside <pre> elements
	findWithRoot(sel, "pre span").Each(func(_ int, span *goquery.Selection) {
		class, _ := span.Attr("class")
		if lineNumberClassRegex.MatchString(class) {
			span.Remove()
		}
	})

	// Remove line-number table cells (keep only the code cell)
	findWithRoot(sel, "table").Each(func(_ int, table *goquery.Selection) {
		class, _ := table.Attr("class")
		if !lineNumberTableClassRegex.MatchString(class) {
			return
		}
		// In line-number tables, the code is typically in the last <td>
		// containing a <pre>. Remove the line-number <td>.
		table.Find("td").Each(func(_ int, td *goquery.Selection) {
			tdClass, _ := td.Attr("class")
			if lineNumberTableClassRegex.MatchString(tdClass) {
				td.Remove()
				return
			}
			// If no pre inside, it's probably a line-number cell
			if td.Find("pre").Length() == 0 && td.Find("code").Length() == 0 {
				text := strings.TrimSpace(td.Text())
				if isLineNumberBlock(text) {
					td.Remove()
				}
			}
		})
	})
}

// CleanEmptyCodeBlocks removes empty fenced code blocks from markdown output.
func CleanEmptyCodeBlocks(markdown string) string {
	return emptyCodeBlockRegex.ReplaceAllString(markdown, "")
}

// detectLanguage extracts language info from a <code> element using all
// available sources: class prefixes, data attributes, and bare class names.
func detectLanguage(code *goquery.Selection) string {
	// 1. Standard class prefixes: language-X, lang-X
	if class, exists := code.Attr("class"); exists {
		if lang := extractLangFromClass(class); lang != "" {
			return lang
		}
	}

	// 2. Data attributes
	if lang := detectLanguageFromAttributes(code); lang != "" {
		return lang
	}

	// 3. Check parent <pre> for language info
	parent := code.Parent()
	if goquery.NodeName(parent) == "pre" {
		if class, exists := parent.Attr("class"); exists {
			if lang := extractLangFromClass(class); lang != "" {
				return lang
			}
		}
		if lang := detectLanguageFromAttributes(parent); lang != "" {
			return lang
		}
	}

	return ""
}

// detectLanguageFromAttributes checks data-language, data-lang, and bare
// class names for language identification.
func detectLanguageFromAttributes(el *goquery.Selection) string {
	// data-language attribute (Hugo, Gatsby)
	if lang, exists := el.Attr("data-language"); exists && lang != "" {
		return strings.ToLower(strings.TrimSpace(lang))
	}

	// data-lang attribute
	if lang, exists := el.Attr("data-lang"); exists && lang != "" {
		return strings.ToLower(strings.TrimSpace(lang))
	}

	// Bare class names matching known languages
	if class, exists := el.Attr("class"); exists {
		return detectBareLanguageClass(class)
	}

	return ""
}

// extractLangFromClass extracts language from class="language-X" or class="lang-X".
func extractLangFromClass(class string) string {
	for part := range strings.FieldsSeq(class) {
		lower := strings.ToLower(part)
		if lang, ok := strings.CutPrefix(lower, "language-"); ok {
			return lang
		}
		if lang, ok := strings.CutPrefix(lower, "lang-"); ok {
			return lang
		}
	}
	return ""
}

// detectBareLanguageClass looks for known language names in class values.
// For example: class="python", class="hljs python", class="highlight go".
func detectBareLanguageClass(class string) string {
	for part := range strings.FieldsSeq(class) {
		lower := strings.ToLower(part)
		// Skip non-language classes
		if lower == "hljs" || lower == "highlight" || lower == "code" ||
			lower == "codehilite" || lower == "sourceCode" || lower == "source" {
			continue
		}
		if knownLanguages[lower] {
			return lower
		}
	}
	return ""
}

// isLineNumberBlock checks if text content is just a sequence of line numbers.
func isLineNumberBlock(text string) bool {
	if text == "" {
		return false
	}
	lines := strings.Split(text, "\n")
	numCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		isNum := true
		for _, ch := range trimmed {
			if ch < '0' || ch > '9' {
				isNum = false
				break
			}
		}
		if !isNum {
			return false
		}
		numCount++
	}
	return numCount > 0
}

// ReparentCodeBlocks extracts <pre> elements from a selection that is about
// to be removed, inserting them before the parent element in the DOM tree.
// Returns the extracted pre elements so the caller can decide what to do.
func ReparentCodeBlocks(sel *goquery.Selection) {
	sel.Each(func(_ int, el *goquery.Selection) {
		pres := el.Find("pre")
		if pres.Length() == 0 {
			return
		}
		// Move each <pre> before the element being removed
		pres.Each(func(_ int, pre *goquery.Selection) {
			// Wrap in a div to avoid breaking document flow
			wrapper := fmt.Sprintf("<div class=\"repodocs-preserved-code\">%s</div>", outerHTML(pre))
			el.BeforeHtml(wrapper)
		})
	})
}

// outerHTML returns the outer HTML of a selection.
func outerHTML(sel *goquery.Selection) string {
	html, err := goquery.OuterHtml(sel)
	if err != nil {
		return ""
	}
	return html
}
