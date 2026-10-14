package buildtool

import (
	_ "embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// This frozen inventory contains only entries already absent from the original
// checkout. Never derive it from the current filesystem: that would hide new
// missing dependencies. See legacy-missing.json and the build-tool documentation.
//
//go:embed legacy-missing.json
var legacyMissingJSON []byte

type project struct {
	XMLName   xml.Name      `xml:"project"`
	Name      string        `xml:"name,attr"`
	Author    string        `xml:"author,attr"`
	Version   string        `xml:"version,attr"`
	Copyright string        `xml:"copyright,attr"`
	Source    string        `xml:"source,attr"`
	SourceDir string        `xml:"source-dir,attr"`
	Minify    string        `xml:"minify,attr"`
	MinDir    string        `xml:"min-dir,attr"`
	Files     []projectFile `xml:"file"`
	Targets   []target      `xml:"target"`
}

type projectFile struct {
	Name string `xml:"name,attr"`
	Path string `xml:"path,attr"`
}

type target struct {
	Name      string    `xml:"name,attr"`
	File      string    `xml:"file,attr"`
	Debug     string    `xml:"debug,attr"`
	Shorthand string    `xml:"shorthand,attr"`
	Includes  []include `xml:"include"`
}

type include struct {
	Name string `xml:"name,attr"`
}
type sourceFile struct {
	source, minified []byte
	parsed           bool
}

var blockComments = regexp.MustCompile(`(?s)/\*.*?\*/`)

func buildProject(manifest, out string, allowedMissing []string, patches []sourcePatch, log io.Writer) error {
	data, err := os.ReadFile(manifest)
	if err != nil {
		return err
	}
	var p project
	if err := xml.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("parse %s: %w", manifest, err)
	}
	source, err := strconv.ParseBool(p.Source)
	if err != nil {
		return fmt.Errorf("%s: invalid source attribute: %w", manifest, err)
	}
	minify, err := strconv.ParseBool(p.Minify)
	if err != nil {
		return fmt.Errorf("%s: invalid minify attribute: %w", manifest, err)
	}
	if len(p.Files) == 0 || len(p.Targets) == 0 {
		return fmt.Errorf("%s: project has no files or targets", manifest)
	}
	allow := make(map[string]bool, len(allowedMissing))
	for _, name := range allowedMissing {
		allow[name] = true
	}
	base := filepath.Dir(manifest)
	out, err = filepath.Abs(out)
	if err != nil {
		return err
	}
	applyVars := func(value string) string {
		return strings.NewReplacer("$author", p.Author, "$version", p.Version, "$output", filepath.ToSlash(out), "$projectName", p.Name, "$projectDir", filepath.ToSlash(base)).Replace(strings.ReplaceAll(value, `\`, "/"))
	}
	header := applyVars(p.Copyright)
	if header != "" {
		header = "/*\r\n * " + strings.ReplaceAll(header, "\n", "\n * ") + "\r\n */\r\n\r\n"
	}
	sourceDir, err := pathWithin(out, applyVars(p.SourceDir))
	if err != nil {
		return err
	}
	minDir, err := pathWithin(out, applyVars(p.MinDir))
	if err != nil {
		return err
	}
	files := make(map[string]sourceFile, len(p.Files))
	declared := make(map[string]bool, len(p.Files))
	missing := 0
	applied := make([]bool, len(patches))
	// Validate and parse every input before writing anything for this project.
	for _, entry := range p.Files {
		name := strings.ReplaceAll(entry.Name, `\`, "/")
		if declared[name] {
			return fmt.Errorf("%s: duplicate file %q", manifest, name)
		}
		declared[name] = true
		input, err := pathWithin(base, name)
		if err != nil {
			return err
		}
		info, err := os.Lstat(input)
		if os.IsNotExist(err) && allow[name] {
			missing++
			continue
		}
		if err != nil {
			return fmt.Errorf("%s: required input %q: %w", manifest, name, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s: input %q is not a regular file", manifest, name)
		}
		realInput, err := filepath.EvalSymlinks(input)
		if err != nil {
			return err
		}
		if _, err := pathWithin(base, realInput); err != nil {
			return err
		}
		bytes, err := os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("read %s: %w", input, err)
		}
		file := sourceFile{source: bytes}
		switch strings.ToLower(filepath.Ext(name)) {
		case ".js":
			// Match StreamReader's handling of UTF-8 BOMs in the previous builder.
			file.source = []byte(strings.TrimPrefix(string(bytes), "\uFEFF"))
			patched, err := applySourcePatches(string(file.source), name, patches, applied)
			if err != nil {
				return err
			}
			file.source = []byte(patched)
			result, err := minifyJS(string(file.source))
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			file.minified, file.parsed = []byte(result), true
		case ".css":
			file.source = []byte(strings.TrimPrefix(string(bytes), "\uFEFF"))
			file.minified, file.parsed = blockComments.ReplaceAll(file.source, nil), true
		}
		files[name] = file
	}
	if err := checkAppliedPatches(patches, applied); err != nil {
		return err
	}
	for _, t := range p.Targets {
		if strings.EqualFold(t.Shorthand, "true") {
			return fmt.Errorf("target %q: shorthand is unsupported", t.Name)
		}
		if len(t.Includes) == 0 {
			return fmt.Errorf("target %q has no includes", t.Name)
		}
		for _, inc := range t.Includes {
			name := strings.ReplaceAll(inc.Name, `\`, "/")
			if !declared[name] {
				return fmt.Errorf("target %q: undeclared include %q", t.Name, name)
			}
			file, exists := files[name]
			if !exists && allow[name] {
				continue
			}
			if !exists || !file.parsed {
				return fmt.Errorf("target %q: unusable include %q", t.Name, name)
			}
		}
	}
	if missing > 0 {
		fmt.Fprintf(log, "%s: skipped %d documented historical entries\n", filepath.Base(manifest), missing)
	}
	for _, entry := range p.Files {
		name := strings.ReplaceAll(entry.Name, `\`, "/")
		file, exists := files[name]
		if !exists {
			continue
		}
		if source {
			output, err := pathWithin(sourceDir, filepath.Join(filepath.FromSlash(strings.ReplaceAll(entry.Path, `\`, "/")), filepath.Base(name)))
			if err != nil {
				return err
			}
			content := file.source
			if file.parsed {
				content = append([]byte(header), content...)
			}
			if err := writeFile(output, content); err != nil {
				return err
			}
		}
		if minify && file.parsed {
			filename := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name)) + "-min" + filepath.Ext(name)
			output, err := pathWithin(minDir, filepath.Join(filepath.FromSlash(strings.ReplaceAll(entry.Path, `\`, "/")), filename))
			if err != nil {
				return err
			}
			if err := writeFile(output, append([]byte(header), file.minified...)); err != nil {
				return err
			}
		}
	}
	for _, t := range p.Targets {
		var compact, debug strings.Builder
		compact.WriteString(header)
		debug.WriteString(header)
		count := 0
		for _, inc := range t.Includes {
			file, exists := files[strings.ReplaceAll(inc.Name, `\`, "/")]
			if !exists {
				continue
			} // Only explicitly allowed historical entries reach here.
			compact.Write(file.minified)
			compact.WriteByte('\n')
			debug.Write(blockComments.ReplaceAll(file.source, nil))
			debug.WriteByte('\n')
			count++
		}
		if count == 0 {
			return fmt.Errorf("target %q has no usable includes", t.Name)
		}
		output, err := pathWithin(out, applyVars(t.File))
		if err != nil {
			return err
		}
		if err := writeFile(output, []byte(compact.String())); err != nil {
			return err
		}
		if strings.EqualFold(t.Debug, "true") {
			if filepath.Ext(output) == ".js" {
				output = strings.TrimSuffix(output, ".js") + "-debug.js"
			}
			if err := writeFile(output, []byte(debug.String())); err != nil {
				return err
			}
		}
	}
	return nil
}

func legacyMissing() (map[string][]string, error) {
	var result map[string][]string
	err := json.Unmarshal(legacyMissingJSON, &result)
	return result, err
}

// Resolve manifest paths without permitting traversal or absolute paths outside
// the designated tree. Files copied from the repository must also be regular.
func pathWithin(root, name string) (string, error) {
	name = filepath.FromSlash(strings.ReplaceAll(name, `\`, "/"))
	if !filepath.IsAbs(name) {
		name = filepath.Join(root, name)
	}
	resolved, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	base, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes %s", name, root)
	}
	return resolved, nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
