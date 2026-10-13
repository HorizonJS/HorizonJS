package extdoc

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

//go:embed class.html
var classTemplate string

type memberView struct {
	*Member
	Inherited bool
}
type classView struct {
	*Class
	Configs, Properties, Methods, Events []memberView
	Constructor                          memberView
}

// Generate renders the original Ext documentation UI with native Go parsing
// and templates. No Java, Ant, XSLT runtime, or external process is needed.
func Generate(ctx context.Context, source, assets, out string, log io.Writer) error {
	catalog, err := parse(ctx, source)
	if err != nil {
		return err
	}
	funcs := template.FuncMap{"esc": html.EscapeString, "join": strings.Join, "nbsp": func(n int) string { return strings.Repeat("&nbsp;", n*2) }, "add": func(a, b int) int { return a + b }, "last": func(classes []*Class) *Class { return classes[len(classes)-1] }}
	tpl, err := template.New("class").Funcs(funcs).Parse(classTemplate)
	if err != nil {
		return err
	}
	for _, name := range []string{"index.html", "welcome.html", "license.txt", "resources"} {
		if err := copyResources(filepath.Join(assets, name), filepath.Join(out, name)); err != nil {
			return err
		}
	}
	wrapper, err := os.ReadFile(filepath.Join(assets, "source-wrap.html"))
	if err != nil {
		return err
	}
	prefix, suffix, ok := strings.Cut(string(wrapper), "###SOURCE###")
	if !ok {
		return fmt.Errorf("source wrapper has no ###SOURCE### marker")
	}
	for _, file := range catalog.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Source code is escaped so literal HTML in JavaScript remains visible.
		var source strings.Builder
		source.WriteString(prefix)
		sort.SliceStable(file.Anchors, func(i, j int) bool { return file.Anchors[i].Offset < file.Anchors[j].Offset })
		last := 0
		for _, anchor := range file.Anchors {
			source.WriteString(html.EscapeString(file.Content[last:anchor.Offset]))
			source.WriteString(`<div id="` + html.EscapeString(anchor.ID) + `"></div>`)
			last = anchor.Offset
		}
		source.WriteString(html.EscapeString(file.Content[last:]))
		source.WriteString(suffix)
		if err := write(filepath.Join(out, "source", file.Target), []byte(source.String())); err != nil {
			return err
		}
	}
	for _, cls := range catalog.Classes {
		if err := ctx.Err(); err != nil {
			return err
		}
		view := classView{Class: cls, Constructor: memberView{Member: &Member{Name: cls.ShortName, Owner: cls.Name, ShortOwner: cls.ShortName, Href: cls.Href, Description: cls.ConstructorDescription, Params: cls.Params, Custom: cls.Custom}}}
		for _, member := range cls.Members {
			item := memberView{member, member.Owner != cls.Name}
			switch member.Kind {
			case "cfg":
				view.Configs = append(view.Configs, item)
			case "prop":
				view.Properties = append(view.Properties, item)
			case "method":
				view.Methods = append(view.Methods, item)
			case "event":
				view.Events = append(view.Events, item)
			}
		}
		var rendered bytes.Buffer
		if err := tpl.ExecuteTemplate(&rendered, "class", view); err != nil {
			return err
		}
		if err := write(filepath.Join(out, "output", cls.Name+".html"), rendered.Bytes()); err != nil {
			return err
		}
	}
	tree, icons := navigation(catalog.Classes)
	treeJSON, err := json.Marshal(tree)
	if err != nil {
		return err
	}
	iconsJSON, err := json.Marshal(icons)
	if err != nil {
		return err
	}
	data := append([]byte("Docs.classData = "), treeJSON...)
	data = append(data, []byte(";\nDocs.icons = ")...)
	data = append(data, iconsJSON...)
	data = append(data, []byte(";\n")...)
	if err := write(filepath.Join(out, "output", "tree.js"), data); err != nil {
		return err
	}
	if log != nil {
		fmt.Fprintf(log, "Generated %d class pages and %d source pages with Go\n", len(catalog.Classes), len(catalog.Files))
	}
	return nil
}

type treeNode struct {
	ID       string      `json:"id"`
	Text     string      `json:"text"`
	Icon     string      `json:"iconCls"`
	CSS      string      `json:"cls,omitempty"`
	Href     string      `json:"href,omitempty"`
	IsClass  bool        `json:"isClass,omitempty"`
	Leaf     bool        `json:"leaf,omitempty"`
	Expand   bool        `json:"singleClickExpand,omitempty"`
	Children []*treeNode `json:"children,omitempty"`
}

func navigation(classes []*Class) (*treeNode, map[string]string) {
	root := &treeNode{ID: "apidocs", Text: "API Documentation", Icon: "icon-docs", Expand: true}
	icons := map[string]string{}
	for _, cls := range classes {
		node := root
		full := ""
		if cls.Package != "" {
			for _, part := range strings.Split(cls.Package, ".") {
				if full != "" {
					full += "."
				}
				full += part
				var child *treeNode
				for _, existing := range node.Children {
					if existing.ID == "pkg-"+full {
						child = existing
						break
					}
				}
				if child == nil {
					child = &treeNode{ID: "pkg-" + full, Text: part, Icon: "icon-pkg", CSS: "package", Expand: true}
					node.Children = append(node.Children, child)
				}
				node = child
			}
		}
		icon := "icon-cls"
		if cls.Component {
			icon = "icon-cmp"
		}
		if cls.Singleton {
			icon = "icon-static"
		}
		for _, custom := range cls.Custom {
			if custom.Title == "brick-icon" {
				icon = custom.Value
				break
			}
		}
		icons[cls.Name] = icon
		node.Children = append(node.Children, &treeNode{ID: cls.Name, Text: cls.ShortName, Icon: icon, CSS: "cls", Href: "output/" + cls.Name + ".html", IsClass: true, Leaf: true})
	}
	var sortNodes func(*treeNode)
	sortNodes = func(node *treeNode) {
		sort.SliceStable(node.Children, func(i, j int) bool {
			a, b := node.Children[i], node.Children[j]
			if a.IsClass != b.IsClass {
				return !a.IsClass
			}
			return a.Text < b.Text
		})
		for _, child := range node.Children {
			sortNodes(child)
		}
	}
	sortNodes(root)
	return root, icons
}

func copyResources(source, dest string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsupported docs resource symlink: %s", path)
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported docs resource: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return write(target, data)
	})
}

func write(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
