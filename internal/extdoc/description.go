package extdoc

import (
	"html"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf16"
)

var shortTags = regexp.MustCompile(`<\S*?>`)
var codeBlocks = regexp.MustCompile(`(?s)<code>(.*?)(</code>|$)`)

func describe(text, currentClass string, alwaysShort bool) Description {
	text = codeBlocks.ReplaceAllStringFunc(text, func(block string) string {
		body := strings.TrimPrefix(block, "<code>")
		suffix := ""
		if strings.HasSuffix(body, "</code>") {
			body = strings.TrimSuffix(body, "</code>")
			suffix = "</code>"
		}
		return "<code>" + highlight(strings.TrimSpace(body)) + suffix
	})
	long, plain := expandLinks(text, currentClass)
	plain = shortTags.ReplaceAllString(plain, "")
	// Java's limit counts UTF-16 code units rather than Unicode code points.
	units := utf16.Encode([]rune(plain))
	description := Description{Long: long, HasShort: alwaysShort || len(units) > 117}
	if description.HasShort {
		description.Short = plain
		if len(units) > 117 {
			description.Short = string(utf16.Decode(units[:117])) + "..."
		}
	}
	return description
}

func expandLinks(text, currentClass string) (string, string) {
	var long, plain strings.Builder
	for {
		at := strings.Index(text, "{@link")
		if at < 0 || at+6 == len(text) {
			long.WriteString(text)
			plain.WriteString(text)
			break
		}
		long.WriteString(text[:at])
		plain.WriteString(text[:at])
		// The legacy scanner consumes the first character after the marker,
		// normally a space. Preserve its handling of incomplete inline links.
		text = text[at+7:]
		end := strings.IndexByte(text, '}')
		if end < 0 {
			long.WriteString(text)
			plain.WriteString(text)
			break
		}
		class, member, label := linkParts(text[:end], currentClass)
		if member == "" {
			long.WriteString(`<a href="output/` + html.EscapeString(class) + `.html" ext:cls="` + html.EscapeString(class) + `">` + label + `</a>`)
		} else {
			long.WriteString(`<a href="output/` + html.EscapeString(class) + `.html#` + html.EscapeString(class+"-"+member) + `" ext:member="` + html.EscapeString(member) + `" ext:cls="` + html.EscapeString(class) + `">` + label + `</a>`)
		}
		plain.WriteString(label)
		text = text[end+1:]
	}
	return long.String(), plain.String()
}

func linkParts(value, currentClass string) (class, member, label string) {
	parts := fields(value, 2)
	reference := parts[0]
	label = parts[1]
	class, member, _ = strings.Cut(reference, "#")
	if member != "" {
		if label == "" {
			if class == "" {
				label = member
			} else {
				label = class + "." + member
			}
		}
		if class == "" {
			class = currentClass
		}
	} else if label == "" {
		label = class
	}
	return
}

func highlight(text string) string {
	// Preserve the original highlighter's order: comments, keywords, quotes.
	var comments strings.Builder
	inComment := false
	for i, r := range text {
		if !inComment && r == '/' && strings.HasPrefix(text[i:], "//") {
			comments.WriteString("<i>")
			inComment = true
		}
		if inComment && r == '\n' {
			comments.WriteString("</i>")
			inComment = false
		}
		comments.WriteRune(r)
	}
	if inComment {
		comments.WriteString("</i>")
	}
	keywords := map[string]bool{}
	for _, word := range strings.Fields("break continue do for import new void case default else function in return typeof while comment delete export if label switch var with") {
		keywords[word] = true
	}
	runes := []rune(comments.String())
	var marked strings.Builder
	for i := 0; i < len(runes); {
		j := i + 1
		letters := unicode.IsLetter(runes[i])
		for j < len(runes) && unicode.IsLetter(runes[j]) == letters {
			j++
		}
		word := string(runes[i:j])
		if letters && keywords[word] {
			marked.WriteString("<b>" + word + "</b>")
		} else {
			marked.WriteString(word)
		}
		i = j
	}
	var quoted strings.Builder
	single, double := false, false
	for _, r := range marked.String() {
		switch r {
		case '\'':
			if single {
				quoted.WriteString("'</em>")
			} else {
				quoted.WriteString("<em>'")
			}
			single = !single
		case '"':
			if double {
				quoted.WriteString(`"</em>`)
			} else {
				quoted.WriteString(`<em>"`)
			}
			double = !double
		default:
			quoted.WriteRune(r)
		}
	}
	return quoted.String()
}
