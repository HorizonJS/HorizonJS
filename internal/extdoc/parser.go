package extdoc

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var classNamePattern = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_.$]*$`)
var tagPattern = regexp.MustCompile(`(?:^|\s)(@\S*)`)
var codeWord = regexp.MustCompile(`[\p{L}\p{N}._]+`)

// Preserve the Java scanner's observed input order. It affects merged class
// comments, unnamed legacy members, overrides, and duplicate source filenames.
// New files are still scanned, in deterministic order after these known paths.
//
//go:embed legacy-source-order.json
var legacySourceOrder []byte

type tag struct{ Name, Text string }
type comment struct {
	Description string
	Tags        []tag
}

func (c comment) get(name string) (string, bool) {
	for _, t := range c.Tags {
		if t.Name == name {
			return t.Text, true
		}
	}
	return "", false
}
func (c comment) has(name string) bool { _, ok := c.get(name); return ok }

func parseComment(content string) comment {
	// Strip leading stars but retain indentation of unstarred code-example lines.
	var clean, gap strings.Builder
	leading, star := true, false
	for _, r := range content {
		if leading {
			if unicode.IsSpace(r) || r == '*' {
				gap.WriteRune(r)
				star = star || r == '*'
				continue
			}
			if !star {
				clean.WriteString(gap.String())
			}
			gap.Reset()
			leading = false
		}
		clean.WriteRune(r)
		if r == '\n' {
			leading, star = true, false
		}
	}
	inner := clean.String()
	positions := tagPattern.FindAllStringSubmatchIndex(inner, -1)
	c := comment{}
	if len(positions) == 0 {
		c.Description = strings.TrimSpace(inner)
		return c
	}
	c.Description = strings.TrimSpace(inner[:positions[0][0]])
	for i, position := range positions {
		end := len(inner)
		if i+1 < len(positions) {
			end = positions[i+1][0]
		}
		c.Tags = append(c.Tags, tag{inner[position[2]:position[3]], strings.TrimSpace(inner[position[3]:end])})
	}
	return c
}

func fields(text string, count int) []string {
	result := make([]string, count)
	remaining := strings.TrimLeftFunc(text, unicode.IsSpace)
	for i := 0; i < count-1; i++ {
		at := strings.IndexFunc(remaining, unicode.IsSpace)
		if at < 0 {
			result[i] = remaining
			return result
		}
		result[i] = remaining[:at]
		remaining = strings.TrimLeftFunc(remaining[at:], unicode.IsSpace)
	}
	result[count-1] = remaining
	return result
}

func parse(ctx context.Context, root string) (*catalog, error) {
	c := &catalog{ByName: map[string]*Class{}}
	names := map[string]int{}
	paths, err := sourcePaths(ctx, root)
	if err != nil {
		return nil, err
	}
	var order []string
	if err := json.Unmarshal(legacySourceOrder, &order); err != nil {
		return nil, err
	}
	ranks := map[string]int{}
	for i, name := range order {
		ranks[name] = i
	}
	sort.SliceStable(paths, func(i, j int) bool {
		a, _ := filepath.Rel(root, paths[i])
		b, _ := filepath.Rel(root, paths[j])
		ar, aKnown := ranks[filepath.ToSlash(a)]
		br, bKnown := ranks[filepath.ToSlash(b)]
		if aKnown != bKnown {
			return aKnown
		}
		if aKnown {
			return ar < br
		}
		return a < b
	})
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		name := filepath.Base(path)
		target := strings.TrimSuffix(name, ".js")
		if names[name] > 0 {
			target += fmt.Sprint(names[name])
		}
		names[name]++
		file := &sourceFile{Name: name, Target: target + ".html", Content: string([]rune(string(data)))}
		c.Files = append(c.Files, file)
		if err := c.parseFile(file); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	if len(c.Classes) == 0 {
		return nil, fmt.Errorf("no documented classes in %s", root)
	}
	if err := c.inherit(); err != nil {
		return nil, err
	}
	return c, nil
}

func sourcePaths(ctx context.Context, root string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var paths []string
	// Docker's source extraction visited directories before sibling files.
	// Make that legacy order deterministic on every operating system.
	for _, directories := range []bool{true, false} {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") || entry.IsDir() != directories {
				continue
			}
			path := filepath.Join(root, entry.Name())
			if entry.Type()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("documentation input is a symlink: %s", path)
			}
			if directories {
				children, err := sourcePaths(ctx, path)
				if err != nil {
					return nil, err
				}
				paths = append(paths, children...)
			} else if filepath.Ext(path) == ".js" {
				paths = append(paths, path)
			}
		}
	}
	return paths, nil
}

func (c *catalog) parseFile(file *sourceFile) error {
	content := file.Content
	for position := 0; position < len(content); {
		start := strings.Index(content[position:], "/**")
		if start < 0 {
			break
		}
		start += position
		end := strings.Index(content[start+3:], "*/")
		if end < 0 {
			return fmt.Errorf("unterminated documentation comment at byte %d", start)
		}
		end += start + 3
		next := strings.Index(content[end+2:], "/**")
		limit := len(content)
		if next >= 0 {
			limit = end + 2 + next
		}
		words := codeWord.FindAllString(content[end+2:limit], 2)
		first, second := "", ""
		if len(words) > 0 {
			first = words[0]
		}
		if len(words) > 1 {
			second = words[1]
		}
		// Match the CRLF source used by the Java reference for its UTF-16
		// description truncation, independently of Git's checkout line endings.
		commentText := strings.ReplaceAll(content[start+3:end], "\r\n", "\n")
		commentText = strings.ReplaceAll(commentText, "\n", "\r\n")
		if err := c.addComment(parseComment(commentText), first, second, file, start); err != nil {
			return err
		}
		position = end + 2
	}
	return nil
}

func (c *catalog) addComment(doc comment, first, second string, file *sourceFile, offset int) error {
	if classTag, ok := doc.get("@class"); ok {
		parts := fields(classTag, 2)
		name := parts[0]
		if !classNamePattern.MatchString(name) {
			return fmt.Errorf("invalid class name %q", name)
		}
		cls := c.ByName[name]
		if cls == nil {
			pkg, short := splitPackage(name)
			if namespace, ok := doc.get("@namespace"); ok {
				pkg, short = namespace, lastPart(name)
			}
			cls = &Class{Name: name, Package: pkg, ShortName: short, Href: file.Target + "#cls-" + name}
			c.ByName[name] = cls
			c.Classes = append(c.Classes, cls)
			file.Anchors = append(file.Anchors, anchor{offset, "cls-" + name})
		}
		c.current = cls
		cls.DefinedIn = append(cls.DefinedIn, file.Name)
		cls.Singleton = cls.Singleton || doc.has("@singleton")
		cls.Hide = cls.Hide || doc.has("@private") || doc.has("@ignore")
		extends, hasExtends := doc.get("@extends")
		if cls.ParentName == "" && hasExtends {
			cls.ParentName = fields(extends, 2)[0]
		}
		if !cls.HasConstructor && doc.has("@constructor") {
			constructor, _ := doc.get("@constructor")
			cls.HasConstructor = true
			cls.ConstructorDescription = describe(constructor, cls.Name, true)
			cls.Params = parseParams(doc, cls.Name)
		}
		if cls.Description == "" {
			description := parts[1]
			if description == "" && hasExtends {
				description = fields(extends, 2)[1]
			}
			cls.Description = describe(description, cls.Name, false).Long
		}
		cls.Custom = append(cls.Custom, customTags(doc)...)
		for _, t := range doc.Tags {
			if t.Name == "@cfg" {
				c.addMember(configMember(t.Text, cls), file, offset)
			}
		}
		return nil
	}
	if doc.has("@private") || doc.has("@ignore") {
		return nil
	}
	if c.current == nil {
		return fmt.Errorf("public member %q appears before a class", first)
	}
	cls := c.current
	m := &Member{Name: lastPart(first), Type: "Object", Owner: cls.Name, ShortOwner: cls.ShortName, Hide: doc.has("@hide"), Custom: customTags(doc)}
	switch {
	case doc.has("@event"):
		text, _ := doc.get("@event")
		parts := fields(text, 2)
		m.Kind, m.Name = "event", parts[0]
		m.Description = describe(parts[1], cls.Name, true)
		m.Params = parseParams(doc, cls.Name)
	case doc.has("@cfg"):
		text, _ := doc.get("@cfg")
		cfg := configMember(text, cls)
		cfg.Hide, cfg.Custom = m.Hide, m.Custom
		m = cfg
	case doc.has("@param") || doc.has("@return") || doc.has("@method") || (!doc.has("@type") && !doc.has("@property") && second == "function"):
		m.Kind = "method"
		_, m.Name = splitPackage(first)
		if text, ok := doc.get("@method"); ok && text != "" {
			m.Name = text
		}
		if text, ok := doc.get("@member"); ok {
			parts := fields(text, 2)
			m.Owner = parts[0]
			_, m.ShortOwner = splitPackage(m.Owner)
			if parts[1] != "" {
				m.Name = parts[1]
			}
		}
		m.Static = doc.has("@static")
		m.Description = describe(doc.Description, cls.Name, true)
		m.Params = parseParams(doc, cls.Name)
		if text, ok := doc.get("@return"); ok {
			parts := fields(text, 2)
			m.ReturnType = strings.Trim(parts[0], "{}")
			m.ReturnDescription = parts[1]
		}
	default:
		m.Kind = "prop"
		description := doc.Description
		if text, ok := doc.get("@property"); ok {
			parts := fields(text, 2)
			if parts[0] != "" {
				m.Name = parts[0]
			}
			if parts[1] != "" {
				description = parts[1]
			}
		}
		if text, ok := doc.get("@type"); ok {
			m.Type = strings.Trim(text, "{}")
		}
		m.Description = describe(description, cls.Name, false)
	}
	// The original source has adjacent property comments with no following
	// declaration. Extdoc emits empty-name rows for these; retain that behavior.
	if m.Name == "" && m.Kind != "prop" {
		return fmt.Errorf("empty %s name in %s", m.Kind, cls.Name)
	}
	c.addMember(m, file, offset)
	return nil
}

func configMember(text string, cls *Class) *Member {
	parts := fields(text, 3)
	optional := isOptional(parts[2])
	description := parts[2]
	if optional {
		// Extdoc removes the marker and exactly one following character,
		// including a CR when the description starts on the next line.
		if len(description) > 11 {
			description = description[11:]
		} else {
			description = ""
		}
	}
	return &Member{Kind: "cfg", Name: parts[1], Type: strings.Trim(parts[0], "{}"), Description: describe(description, cls.Name, false), Optional: optional, Owner: cls.Name, ShortOwner: cls.ShortName}
}

func (c *catalog) addMember(m *Member, file *sourceFile, offset int) {
	id := m.Kind + "-" + m.Owner + "-" + m.Name
	m.Href = file.Target + "#" + id
	if m.Kind == "method" {
		for i, old := range c.Members {
			if old.Kind == m.Kind && old.Owner == m.Owner && old.Name == m.Name {
				if strings.HasPrefix(old.Href, file.Target+"#") {
					for j, oldAnchor := range file.Anchors {
						if oldAnchor.ID == id {
							file.Anchors = append(file.Anchors[:j], file.Anchors[j+1:]...)
							break
						}
					}
				}
				c.Members = append(c.Members[:i], c.Members[i+1:]...)
				break
			}
		}
	}
	c.Members = append(c.Members, m)
	file.Anchors = append(file.Anchors, anchor{offset, id})
}

func parseParams(doc comment, cls string) []Param {
	var params []Param
	for _, t := range doc.Tags {
		if t.Name == "@param" {
			parts := fields(t.Text, 3)
			params = append(params, Param{Name: parts[1], Type: strings.Trim(parts[0], "{}"), Description: describe(parts[2], cls, false).Long, Optional: isOptional(parts[2])})
		}
	}
	return params
}

func customTags(doc comment) []CustomTag {
	var result []CustomTag
	for _, definition := range []struct{ Name, Title, Format string }{{"@author", "Author", ""}, {"@version", "Version", ""}, {"@note", "", "<i>NOTE: {0}</i>"}, {"@demo", "Demo", `<a href="{0}">{0}</a>`}, {"@xtype", "xtype", ""}, {"@brick", "brick-icon", ""}} {
		if text, ok := doc.get(definition.Name); ok {
			if definition.Format != "" {
				text = strings.ReplaceAll(definition.Format, "{0}", text)
			}
			result = append(result, CustomTag{definition.Title, text})
		}
	}
	return result
}

func (c *catalog) inherit() error {
	for _, m := range c.Members {
		if owner := c.ByName[m.Owner]; owner != nil {
			owner.Members = append(owner.Members, m)
		} else {
			return fmt.Errorf("member %s belongs to undocumented class %s", m.Name, m.Owner)
		}
	}
	for _, cls := range c.Classes {
		if parent := c.ByName[cls.ParentName]; parent != nil {
			parent.Subclasses = append(parent.Subclasses, cls)
		}
	}
	// Match Extdoc's ordered inheritance pass, including when a parent has
	// already had its hidden overrides removed before a descendant is visited.
	for _, cls := range c.Classes {
		seen := map[string]bool{cls.Name: true}
		for parent := c.ByName[cls.ParentName]; parent != nil; parent = c.ByName[parent.ParentName] {
			if seen[parent.Name] {
				return fmt.Errorf("inheritance cycle involving %s", parent.Name)
			}
			seen[parent.Name] = true
			cls.Ancestors = append(cls.Ancestors, parent)
			cls.Component = cls.Component || parent.Name == "Ext.Component"
			for _, m := range parent.Members {
				if m.Static {
					continue
				}
				overridden := false
				for _, own := range cls.Members {
					if m.Name == "" {
						break
					}
					if own.Kind == m.Kind && lastPart(own.Name) == lastPart(m.Name) {
						overridden = true
						break
					}
				}
				if !overridden {
					cls.Members = append(cls.Members, m)
				}
			}
		}
		visible := cls.Members[:0]
		for _, m := range cls.Members {
			if !m.Hide {
				visible = append(visible, m)
			}
		}
		cls.Members = visible
		sort.SliceStable(cls.Members, func(i, j int) bool { return cls.Members[i].Name < cls.Members[j].Name })
		for i, j := 0, len(cls.Ancestors)-1; i < j; i, j = i+1, j-1 {
			cls.Ancestors[i], cls.Ancestors[j] = cls.Ancestors[j], cls.Ancestors[i]
		}
		sort.SliceStable(cls.Subclasses, func(i, j int) bool { return cls.Subclasses[i].Name < cls.Subclasses[j].Name })
	}
	visible := c.Classes[:0]
	for _, cls := range c.Classes {
		if !cls.Hide {
			visible = append(visible, cls)
		}
	}
	c.Classes = visible
	return nil
}

func startsUpper(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}
func lastPart(name string) string {
	if at := strings.LastIndex(name, "."); at >= 0 {
		return name[at+1:]
	}
	return name
}
func isOptional(text string) bool {
	return strings.HasPrefix(text, "(optional)") || strings.HasPrefix(text, "(Optional)")
}
